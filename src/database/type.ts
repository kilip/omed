import type { NodePgDatabase } from "drizzle-orm/node-postgres";
import { createInsertSchema } from "drizzle-orm/zod";
import type { relations } from "./relations";
import { organizations, type teams, users } from "./schema";

export type OmedDatabase = NodePgDatabase<typeof relations>;

export const createUserSchema = createInsertSchema(users);
export type NewUser = typeof users.$inferInsert;
export type UserItem = typeof users.$inferSelect;

export const createOrganizationSchema = createInsertSchema(organizations);
export type NewOrganization = typeof organizations.$inferInsert;
export type OrganizationItem = typeof organizations.$inferSelect;

export type TeamItem = typeof teams.$inferSelect;
