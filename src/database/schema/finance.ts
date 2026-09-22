import { defineRelationsPart } from "drizzle-orm";
import {
  date,
  decimal,
  pgSchema,
  text,
  timestamp,
  uuid,
  varchar,
} from "drizzle-orm/pg-core";
import { createInsertSchema } from "drizzle-orm/zod";
import { v7 } from "uuid";
import { users } from "./better-auth";
import { workspaces } from "./workspaces";

export const finance = pgSchema("finance");
export const accountTypeEnum = finance.enum("account_type", [
  "asset",
  "liability",
  "equity",
  "income",
  "expense",
]);
export const ledgerPeriodStatusEnum = finance.enum("ledger_period_status", [
  "open",
  "closed",
  "locked",
]);
export const entryTypeEnum = finance.enum("entry_type", [
  "standard",
  "opening_balance",
  "adjustment",
  // add more here (adjustment, closing, etc.) as needed
]);

export const accounts = finance.table("accounts", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  code: varchar("code", { length: 20 }).notNull(),
  name: varchar("name", { length: 255 }).notNull(),
  type: accountTypeEnum("type").notNull(),
  parentId: uuid("parentId"), // self-reference, see relations below
  workspaceId: uuid("workspaceId")
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("created_at").defaultNow().notNull(),
});

export const createAccountSchema = createInsertSchema(accounts);
export type NewAccount = typeof accounts.$inferInsert;
export type AccountItem = typeof accounts.$inferSelect;

export const ledgerPeriods = finance.table("ledger_periods", {
  id: uuid("id").primaryKey().defaultRandom(),
  status: ledgerPeriodStatusEnum("status").notNull().default("open"),
  startDate: date("startDate").notNull(),
  endDate: date("endDate").notNull(),
  workspaceId: uuid("workspaceId")
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("created_at").defaultNow().notNull(),
});

export const entries = finance.table("entries", {
  id: uuid("id").primaryKey().defaultRandom(),
  entryDate: date("entry_date").notNull(),
  entryType: entryTypeEnum("entry_type").notNull().default("standard"),
  description: text("description"),
  periodId: uuid("period_id")
    .notNull()
    .references(() => ledgerPeriods.id, { onDelete: "cascade" }),
  workspaceId: uuid("workspaceId")
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("created_at").defaultNow().notNull(),
});

export const postings = finance.table("postings", {
  id: uuid("id").primaryKey().defaultRandom(),
  entryId: uuid("entry_id")
    .notNull()
    .references(() => entries.id, { onDelete: "cascade" }),
  accountId: uuid("account_id")
    .notNull()
    .references(() => accounts.id),
  debitAmount: decimal("debit_amount", { precision: 18, scale: 2 })
    .notNull()
    .default("0"),
  creditAmount: decimal("credit_amount", { precision: 18, scale: 2 })
    .notNull()
    .default("0"),
  workspaceId: uuid("workspaceId")
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("created_at").defaultNow().notNull(),
});

export const financeRelations = defineRelationsPart(
  { accounts, ledgerPeriods, entries, postings, users, workspaces },
  (r) => ({
    accounts: {
      parent: r.one.accounts({
        from: r.accounts.parentId,
        to: r.accounts.id,
      }),
      children: r.many.accounts(),
      postings: r.many.postings(),
      creator: r.one.users({
        from: r.accounts.createdBy,
        to: r.users.id,
      }),
      workspace: r.one.workspaces({
        from: r.accounts.workspaceId,
        to: r.workspaces.id,
      }),
    },
    ledgerPeriods: {
      entries: r.many.entries(),
      workspace: r.one.workspaces({
        from: r.ledgerPeriods.workspaceId,
        to: r.workspaces.id,
      }),
    },
    entries: {
      period: r.one.ledgerPeriods({
        from: r.entries.periodId,
        to: r.ledgerPeriods.id,
      }),
      postings: r.many.postings(),
      workspace: r.one.workspaces({
        from: r.entries.workspaceId,
        to: r.workspaces.id,
      }),
    },
    postings: {
      entry: r.one.entries({
        from: r.postings.entryId,
        to: r.entries.id,
      }),
      account: r.one.accounts({
        from: r.postings.accountId,
        to: r.accounts.id,
      }),
      workspace: r.one.workspaces({
        from: r.postings.workspaceId,
        to: r.workspaces.id,
      }),
    },
  }),
);
