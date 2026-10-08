import { inArray, isNull, sql } from "drizzle-orm";
import { Kafka, logLevel, type Producer } from "kafkajs";
import { authEnv } from "./authEnv";
import { authDB } from "./drizzle";
import { outbox } from "./drizzle/schema/outbox";
import { TOPIC } from "./topics";

const BATCH = 100;
const IDLE_MS = 500;
const BACKOFF_MIN_MS = 1_000;
const BACKOFF_MAX_MS = 30_000;
const RELAY_LOCK = 724001; // advisory lock key, bebas asal konstan

const sleep = (ms: number, signal: AbortSignal) =>
  new Promise<void>((resolve) => {
    if (signal.aborted) return resolve();
    const t = setTimeout(done, ms);
    function done() {
      clearTimeout(t);
      signal.removeEventListener("abort", done);
      resolve();
    }
    signal.addEventListener("abort", done, { once: true });
  });

async function ensureTopics(kafka: Kafka) {
  const admin = kafka.admin();
  await admin.connect();
  try {
    await admin.createTopics({
      topics: Object.values(TOPIC).map((topic) => ({
        topic,
        numPartitions: 3,
        configEntries: [{ name: "cleanup.policy", value: "compact" }],
      })),
    });
  } finally {
    await admin.disconnect();
  }
}

/** Publish satu batch. Return jumlah row yang terkirim. */
async function publishBatch(producer: Producer): Promise<number> {
  return authDB.transaction(async (tx) => {
    // hanya satu relay aktif per batch -> urutan per key terjaga
    const res = (await tx.execute(
      sql`select pg_try_advisory_xact_lock(${RELAY_LOCK}) as locked`,
    )) as unknown as { rows?: { locked: boolean }[] } | { locked: boolean }[];
    const lockRows = Array.isArray(res) ? res : (res.rows ?? []);
    if (!lockRows[0]?.locked) return 0;

    const rows = await tx
      .select()
      .from(outbox)
      .where(isNull(outbox.publishedAt))
      .orderBy(outbox.id)
      .limit(BATCH);
    if (rows.length === 0) return 0;

    const byTopic = new Map<string, typeof rows>();
    for (const r of rows) {
      const list = byTopic.get(r.topic);
      if (list) list.push(r);
      else byTopic.set(r.topic, [r]);
    }

    await producer.sendBatch({
      acks: -1,
      topicMessages: [...byTopic].map(([topic, msgs]) => ({
        topic,
        messages: msgs.map((m) => ({
          key: m.key,
          value: m.payload === null ? null : JSON.stringify(m.payload),
          headers: m.headers ?? {},
        })),
      })),
    });

    await tx
      .update(outbox)
      .set({ publishedAt: new Date() })
      .where(
        inArray(
          outbox.id,
          rows.map((r) => r.id),
        ),
      );
    return rows.length;
  });
}

export async function runRelay(signal: AbortSignal): Promise<void> {
  const kafka = new Kafka({
    clientId: "omed-auth-relay",
    brokers: authEnv.KAFKA_BROKERS,
    logLevel: logLevel.WARN,
  });

  let backoff = BACKOFF_MIN_MS;

  while (!signal.aborted) {
    const producer = kafka.producer({
      idempotent: true,
      maxInFlightRequests: 1,
    });
    try {
      await ensureTopics(kafka);
      await producer.connect();
      console.log("[relay] connected to kafka");
      backoff = BACKOFF_MIN_MS;

      while (!signal.aborted) {
        const sent = await publishBatch(producer);
        if (sent === 0) await sleep(IDLE_MS, signal);
      }
    } catch (e) {
      console.error(`[relay] error, retry in ${backoff}ms`, e);
      await sleep(backoff, signal);
      backoff = Math.min(backoff * 2, BACKOFF_MAX_MS);
    } finally {
      await producer.disconnect().catch(() => {});
    }
  }
  console.log("[relay] stopped");
}
