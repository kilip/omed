import { pgSchema, bigserial, text, jsonb, timestamp } from "drizzle-orm/pg-core";

export const authSchema = pgSchema("auth");

export const outbox = authSchema.table("outbox", {
  id: bigserial("id", { mode: "number" }).primaryKey(),
  topic: text("topic").notNull(),
  key: text("key").notNull(),
  payload: jsonb("payload"), // null = tombstone (delete)
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  publishedAt: timestamp("published_at", { withTimezone: true }),
});
