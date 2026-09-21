import { defineRelationsPart } from "drizzle-orm";
import {
  date,
  decimal,
  text,
  timestamp,
  uuid,
  varchar,
} from "drizzle-orm/pg-core";
import { createInsertSchema } from "drizzle-orm/zod";
import { v7 } from "uuid";
import { organization } from "./better-auth";
import { omedSchema } from "./omed";

export const accountTypeEnum = omedSchema.enum("account_type", [
  "asset",
  "liability",
  "equity",
  "income",
  "expense",
]);

export const ledgerPeriodStatusEnum = omedSchema.enum("ledger_period_status", [
  "open",
  "closed",
  "locked",
]);

export const entryTypeEnum = omedSchema.enum("entry_type", [
  "standard",
  "opening_balance",
  // add more here (a djustment, closing, etc.) as needed
]);

export const accounts = omedSchema.table("accounts", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  name: varchar("name", { length: 50 }).notNull(),
  type: accountTypeEnum("type").notNull(),
  parentId: uuid("parentId"),
  tenantId: uuid("tenantId")
    .notNull()
    .references(() => organization.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy"),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedBy: uuid("updatedBy"),
  updatedAt: timestamp("updatedAt").$onUpdate(() => new Date()),
});

export const createAccountSchema = createInsertSchema(accounts);
export type NewAccount = typeof accounts.$inferInsert;
export type AccountItem = typeof accounts.$inferSelect;

export const ledgerPeriods = omedSchema.table("ledger_periods", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  status: ledgerPeriodStatusEnum("status").notNull().default("open"),
  startDate: date("startDate").notNull(),
  endDate: date("endDate").notNull(),
});

export const entries = omedSchema.table("entries", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  entryDate: timestamp("entryDate").notNull().defaultNow(),
  description: text("description"),
  periodId: uuid("periodId")
    .notNull()
    .references(() => ledgerPeriods.id, { onDelete: "cascade" }),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
});

export const postings = omedSchema.table("postings", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  entryId: uuid("entryId")
    .notNull()
    .references(() => entries.id, { onDelete: "cascade" }),
  accountId: uuid("accountId")
    .notNull()
    .references(() => accounts.id),
  debitAmount: decimal("debitAmount", { precision: 18, scale: 2 })
    .notNull()
    .default("0"),
  creditAmount: decimal("creditAmount", { precision: 18, scale: 2 })
    .notNull()
    .default("0"),
});

export const accountingRelations = defineRelationsPart(
  { organization, accounts, entries, postings, ledgerPeriods },
  (r) => ({
    organization: {
      accounts: r.many.accounts({
        from: r.organization.id,
        to: r.accounts.tenantId,
      }),
    },
    accounts: {
      tenant: r.one.organization({
        from: r.accounts.tenantId,
        to: r.organization.id,
      }),
      parent: r.one.accounts({
        from: r.accounts.parentId,
        to: r.accounts.id,
      }),
      children: r.many.accounts(),
      postings: r.many.postings(),
    },
    ledgerPeriods: {
      entries: r.many.entries(),
    },
    entries: {
      period: r.one.ledgerPeriods({
        from: r.entries.periodId,
        to: r.ledgerPeriods.id,
      }),
      postings: r.many.postings(),
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
    },
  }),
);
