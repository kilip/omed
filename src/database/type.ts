import type { NodePgDatabase } from "drizzle-orm/node-postgres";
import { createInsertSchema } from "drizzle-orm/zod";
import type { relations } from "./relations";
import { organization, user } from "./schema";

export type OmedDatabase = NodePgDatabase<typeof relations>;

export const createUserSchema = createInsertSchema(user);
export type NewUser = typeof user.$inferInsert;
export type UserItem = typeof user.$inferSelect;

export const createOrganizationSchema = createInsertSchema(organization);
export type NewOrganization = typeof organization.$inferInsert;
export type OrganizationItem = typeof organization.$inferSelect;
