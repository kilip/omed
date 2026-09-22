import { defineRelationsPart, sql } from "drizzle-orm";
import { pgSchema, text, timestamp, boolean, integer, uuid, index, uniqueIndex } from "drizzle-orm/pg-core";

export const betterAuthSchema = pgSchema("better-auth");

export const users = betterAuthSchema.table("users", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					name: text('name').notNull(),
 email: text('email').notNull().unique(),
 emailVerified: boolean('email_verified').default(false).notNull(),
 image: text('image'),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').defaultNow().$onUpdate(() => /* @__PURE__ */ new Date()).notNull(),
 role: text('role'),
 banned: boolean('banned').default(false),
 banReason: text('ban_reason'),
 banExpires: timestamp('ban_expires')
					});

export const sessions = betterAuthSchema.table("sessions", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					expiresAt: timestamp('expires_at').notNull(),
 token: text('token').notNull().unique(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').$onUpdate(() => /* @__PURE__ */ new Date()).notNull(),
 ipAddress: text('ip_address'),
 userAgent: text('user_agent'),
 userId: uuid('user_id').notNull().references(()=> users.id, { onDelete: 'cascade' }),
 impersonatedBy: text('impersonated_by'),
 activeOrganizationId: text('active_organization_id'),
 activeTeamId: text('active_team_id'),
 activeWorkspaceId: text('active_workspace_id').notNull()
					}, (table) => [
  index("sessions_userId_idx").on(table.userId),
]);

export const userAccounts = betterAuthSchema.table("user_accounts", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					accountId: text('account_id').notNull(),
 providerId: text('provider_id').notNull(),
 userId: uuid('user_id').notNull().references(()=> users.id, { onDelete: 'cascade' }),
 accessToken: text('access_token'),
 refreshToken: text('refresh_token'),
 idToken: text('id_token'),
 accessTokenExpiresAt: timestamp('access_token_expires_at'),
 refreshTokenExpiresAt: timestamp('refresh_token_expires_at'),
 scope: text('scope'),
 password: text('password'),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').$onUpdate(() => /* @__PURE__ */ new Date()).notNull()
					}, (table) => [
  index("userAccounts_userId_idx").on(table.userId),
]);

export const verifications = betterAuthSchema.table("verifications", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					identifier: text('identifier').notNull(),
 value: text('value').notNull(),
 expiresAt: timestamp('expires_at').notNull(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').defaultNow().$onUpdate(() => /* @__PURE__ */ new Date()).notNull()
					}, (table) => [
  index("verifications_identifier_idx").on(table.identifier),
]);

export const organizations = betterAuthSchema.table("organizations", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					name: text('name').notNull(),
 slug: text('slug').notNull().unique(),
 logo: text('logo'),
 createdAt: timestamp('created_at').notNull(),
 metadata: text('metadata'),
 isPersonal: boolean('is_personal').default(false)
					}, (table) => [
  uniqueIndex("organizations_slug_uidx").on(table.slug),
]);

export const teams = betterAuthSchema.table("teams", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					name: text('name').notNull(),
 memberCount: integer('member_count').default(0).notNull(),
 organizationId: uuid('organization_id').notNull().references(()=> organizations.id, { onDelete: 'cascade' }),
 createdAt: timestamp('created_at').notNull(),
 updatedAt: timestamp('updated_at').$onUpdate(() => /* @__PURE__ */ new Date()),
 isPersonal: boolean('is_personal').default(false)
					}, (table) => [
  index("teams_organizationId_idx").on(table.organizationId),
]);

export const teamMembers = betterAuthSchema.table("team_members", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					teamId: uuid('team_id').notNull().references(()=> teams.id, { onDelete: 'cascade' }),
 userId: uuid('user_id').notNull().references(()=> users.id, { onDelete: 'cascade' }),
 membershipKey: text('membership_key').unique(),
 createdAt: timestamp('created_at')
					}, (table) => [
  index("teamMembers_teamId_idx").on(table.teamId),
  index("teamMembers_userId_idx").on(table.userId),
]);

export const members = betterAuthSchema.table("members", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					organizationId: uuid('organization_id').notNull().references(()=> organizations.id, { onDelete: 'cascade' }),
 userId: uuid('user_id').notNull().references(()=> users.id, { onDelete: 'cascade' }),
 role: text('role').default("member").notNull(),
 createdAt: timestamp('created_at').notNull()
					}, (table) => [
  index("members_organizationId_idx").on(table.organizationId),
  index("members_userId_idx").on(table.userId),
]);

export const invitations = betterAuthSchema.table("invitations", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					organizationId: uuid('organization_id').notNull().references(()=> organizations.id, { onDelete: 'cascade' }),
 email: text('email').notNull(),
 role: text('role'),
 teamId: text('team_id'),
 status: text('status').default("pending").notNull(),
 expiresAt: timestamp('expires_at').notNull(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 inviterId: uuid('inviter_id').notNull().references(()=> users.id, { onDelete: 'cascade' })
					}, (table) => [
  index("invitations_organizationId_idx").on(table.organizationId),
  index("invitations_email_idx").on(table.email),
]);

export const jwkss = betterAuthSchema.table("jwkss", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					publicKey: text('public_key').notNull(),
 privateKey: text('private_key').notNull(),
 createdAt: timestamp('created_at').notNull(),
 expiresAt: timestamp('expires_at'),
 alg: text('alg'),
 crv: text('crv')
					});


export const authRelations = defineRelationsPart({ users, sessions, userAccounts, verifications, organizations, teams, teamMembers, members, invitations, jwkss }, (r) => ({
  users: {
    sessions: r.many.sessions({
      from: r.users.id,
      to: r.sessions.userId,
    }),
    userAccounts: r.many.userAccounts({
      from: r.users.id,
      to: r.userAccounts.userId,
    }),
    teamMembers: r.many.teamMembers({
      from: r.users.id,
      to: r.teamMembers.userId,
    }),
    members: r.many.members({
      from: r.users.id,
      to: r.members.userId,
    }),
    invitations: r.many.invitations({
      from: r.users.id,
      to: r.invitations.inviterId,
    })
  },
  sessions: {
    user: r.one.users({
      from: r.sessions.userId,
      to: r.users.id,
    })
  },
  userAccounts: {
    user: r.one.users({
      from: r.userAccounts.userId,
      to: r.users.id,
    })
  },
  organizations: {
    teams: r.many.teams({
      from: r.organizations.id,
      to: r.teams.organizationId,
    }),
    members: r.many.members({
      from: r.organizations.id,
      to: r.members.organizationId,
    }),
    invitations: r.many.invitations({
      from: r.organizations.id,
      to: r.invitations.organizationId,
    })
  },
  teams: {
    organization: r.one.organizations({
      from: r.teams.organizationId,
      to: r.organizations.id,
    }),
    teamMembers: r.many.teamMembers({
      from: r.teams.id,
      to: r.teamMembers.teamId,
    })
  },
  teamMembers: {
    team: r.one.teams({
      from: r.teamMembers.teamId,
      to: r.teams.id,
    }),
    user: r.one.users({
      from: r.teamMembers.userId,
      to: r.users.id,
    })
  },
  members: {
    organization: r.one.organizations({
      from: r.members.organizationId,
      to: r.organizations.id,
    }),
    user: r.one.users({
      from: r.members.userId,
      to: r.users.id,
    })
  },
  invitations: {
    organization: r.one.organizations({
      from: r.invitations.organizationId,
      to: r.organizations.id,
    }),
    user: r.one.users({
      from: r.invitations.inviterId,
      to: r.users.id,
    })
  }
}));
