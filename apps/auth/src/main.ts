import { runRelay } from "@omed/better-auth/relay";
import { Elysia } from "elysia";
import { server } from "./server";

const port = Number(process.env.AUTH_PORT ?? 8001);

const app = new Elysia().use(server).listen({ port, hostname: "0.0.0.0" });
console.log(`[auth] listening on 0.0.0.0:${port}`);

const ac = new AbortController();
const relay = runRelay(ac.signal).catch((e) => {
  console.error("[relay] fatal", e);
});

let stopping = false;
const shutdown = async () => {
  if (stopping) return;
  stopping = true;
  ac.abort();
  await relay;
  await app.stop();
  process.exit(0);
};
for (const s of ["SIGINT", "SIGTERM"] as const) process.on(s, shutdown);
