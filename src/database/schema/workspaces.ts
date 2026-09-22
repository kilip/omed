import { defineRelationsPart } from "drizzle-orm";
import { boolean, uuid, varchar } from "drizzle-orm/pg-core";
import { createInsertSchema } from "drizzle-orm/zod";
import { v7 } from "uuid";
import { organizations, teams, users } from "./better-auth";
import { coreSchema } from "./core";

export const workspaces = coreSchema.table("workspaces", {
  id: uuid("id").primaryKey().$defaultFn(v7),
  name: varchar("name", { length: 50 }).notNull(),
  isPersonal: boolean("isPersonal").notNull().default(false),
  teamId: uuid("teamId")
    .notNull()
    .references(() => teams.id, { onDelete: "cascade" }),
  createdBy: uuid("createdBy")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
});

export const createWorkspaceSchema = createInsertSchema(workspaces);
export type NewWorkspace = typeof workspaces.$inferInsert;
export type WorkspaceItem = typeof workspaces.$inferSelect;

export const workspaceRelations = defineRelationsPart(
  { teams, workspaces, organizations },
  (r) => ({
    workspaces: {
      team: r.one.teams({
        from: r.workspaces.teamId,
        to: r.teams.id,
      }),
    },
  }),
);
