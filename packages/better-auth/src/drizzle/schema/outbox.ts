import { sql } from "drizzle-orm";
import { bigserial, index, jsonb, text, timestamp } from "drizzle-orm/pg-core";
import { authSchema } from "./better-auth";

export const outbox = authSchema.table(
  "outbox",
  {
    id: bigserial("id", { mode: "number" }).primaryKey(),
    topic: text("topic").notNull(),
    key: text("key").notNull(),
    payload: jsonb("payload"), // null = tombstone (delete)
    headers: jsonb("headers").$type<Record<string, string>>(),
    createdAt: timestamp("created_at", { withTimezone: true })
      .notNull()
      .defaultNow(),
    publishedAt: timestamp("published_at", { withTimezone: true }),
  },
  (t) => [
    index("outbox_unpublished_idx")
      .on(t.id)
      .where(sql`${t.publishedAt} is null`),
  ],
);
