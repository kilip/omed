import { defineRelationsPart, sql } from "drizzle-orm";
import { pgSchema, text, timestamp, boolean, integer, uuid, index, uniqueIndex } from "drizzle-orm/pg-core";

export const authSchema = pgSchema("auth");

export const user = authSchema.table("user", {
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
 banExpires: timestamp('ban_expires'),
 on_boarded: boolean('on_boarded').default(false),
 active_workspace: text('active_workspace')
					});

export const session = authSchema.table("session", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					expiresAt: timestamp('expires_at').notNull(),
 token: text('token').notNull().unique(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').$onUpdate(() => /* @__PURE__ */ new Date()).notNull(),
 ipAddress: text('ip_address'),
 userAgent: text('user_agent'),
 userId: uuid('user_id').notNull().references(()=> user.id, { onDelete: 'cascade' }),
 impersonatedBy: text('impersonated_by'),
 activeOrganizationId: text('active_organization_id'),
 activeTeamId: text('active_team_id')
					}, (table) => [
  index("session_userId_idx").on(table.userId),
]);

export const account = authSchema.table("account", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					accountId: text('account_id').notNull(),
 providerId: text('provider_id').notNull(),
 userId: uuid('user_id').notNull().references(()=> user.id, { onDelete: 'cascade' }),
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
  index("account_userId_idx").on(table.userId),
]);

export const verification = authSchema.table("verification", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					identifier: text('identifier').notNull(),
 value: text('value').notNull(),
 expiresAt: timestamp('expires_at').notNull(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 updatedAt: timestamp('updated_at').defaultNow().$onUpdate(() => /* @__PURE__ */ new Date()).notNull()
					}, (table) => [
  index("verification_identifier_idx").on(table.identifier),
]);

export const organization = authSchema.table("organization", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					name: text('name').notNull(),
 slug: text('slug').notNull().unique(),
 logo: text('logo'),
 createdAt: timestamp('created_at').notNull(),
 metadata: text('metadata'),
 personal: boolean('personal').default(false)
					}, (table) => [
  uniqueIndex("organization_slug_uidx").on(table.slug),
]);

export const team = authSchema.table("team", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					name: text('name').notNull(),
 memberCount: integer('member_count').default(0).notNull(),
 organizationId: uuid('organization_id').notNull().references(()=> organization.id, { onDelete: 'cascade' }),
 createdAt: timestamp('created_at').notNull(),
 updatedAt: timestamp('updated_at').$onUpdate(() => /* @__PURE__ */ new Date()),
 personal: boolean('personal').default(false)
					}, (table) => [
  index("team_organizationId_idx").on(table.organizationId),
]);

export const teamMember = authSchema.table("team_member", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					teamId: uuid('team_id').notNull().references(()=> team.id, { onDelete: 'cascade' }),
 userId: uuid('user_id').notNull().references(()=> user.id, { onDelete: 'cascade' }),
 membershipKey: text('membership_key').unique(),
 createdAt: timestamp('created_at')
					}, (table) => [
  index("teamMember_teamId_idx").on(table.teamId),
  index("teamMember_userId_idx").on(table.userId),
]);

export const member = authSchema.table("member", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					organizationId: uuid('organization_id').notNull().references(()=> organization.id, { onDelete: 'cascade' }),
 userId: uuid('user_id').notNull().references(()=> user.id, { onDelete: 'cascade' }),
 role: text('role').default("member").notNull(),
 createdAt: timestamp('created_at').notNull()
					}, (table) => [
  index("member_organizationId_idx").on(table.organizationId),
  index("member_userId_idx").on(table.userId),
]);

export const invitation = authSchema.table("invitation", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					organizationId: uuid('organization_id').notNull().references(()=> organization.id, { onDelete: 'cascade' }),
 email: text('email').notNull(),
 role: text('role'),
 teamId: text('team_id'),
 status: text('status').default("pending").notNull(),
 expiresAt: timestamp('expires_at').notNull(),
 createdAt: timestamp('created_at').defaultNow().notNull(),
 inviterId: uuid('inviter_id').notNull().references(()=> user.id, { onDelete: 'cascade' })
					}, (table) => [
  index("invitation_organizationId_idx").on(table.organizationId),
  index("invitation_email_idx").on(table.email),
]);

export const jwks = authSchema.table("jwks", {
					id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
					publicKey: text('public_key').notNull(),
 privateKey: text('private_key').notNull(),
 createdAt: timestamp('created_at').notNull(),
 expiresAt: timestamp('expires_at'),
 alg: text('alg'),
 crv: text('crv')
					});


export const authRelations = defineRelationsPart({ user, session, account, verification, organization, team, teamMember, member, invitation, jwks }, (r) => ({
  user: {
    sessions: r.many.session({
      from: r.user.id,
      to: r.session.userId,
    }),
    accounts: r.many.account({
      from: r.user.id,
      to: r.account.userId,
    }),
    teamMembers: r.many.teamMember({
      from: r.user.id,
      to: r.teamMember.userId,
    }),
    members: r.many.member({
      from: r.user.id,
      to: r.member.userId,
    }),
    invitations: r.many.invitation({
      from: r.user.id,
      to: r.invitation.inviterId,
    })
  },
  session: {
    user: r.one.user({
      from: r.session.userId,
      to: r.user.id,
    })
  },
  account: {
    user: r.one.user({
      from: r.account.userId,
      to: r.user.id,
    })
  },
  organization: {
    teams: r.many.team({
      from: r.organization.id,
      to: r.team.organizationId,
    }),
    members: r.many.member({
      from: r.organization.id,
      to: r.member.organizationId,
    }),
    invitations: r.many.invitation({
      from: r.organization.id,
      to: r.invitation.organizationId,
    })
  },
  team: {
    organization: r.one.organization({
      from: r.team.organizationId,
      to: r.organization.id,
    }),
    teamMembers: r.many.teamMember({
      from: r.team.id,
      to: r.teamMember.teamId,
    })
  },
  teamMember: {
    team: r.one.team({
      from: r.teamMember.teamId,
      to: r.team.id,
    }),
    user: r.one.user({
      from: r.teamMember.userId,
      to: r.user.id,
    })
  },
  member: {
    organization: r.one.organization({
      from: r.member.organizationId,
      to: r.organization.id,
    }),
    user: r.one.user({
      from: r.member.userId,
      to: r.user.id,
    })
  },
  invitation: {
    organization: r.one.organization({
      from: r.invitation.organizationId,
      to: r.organization.id,
    }),
    user: r.one.user({
      from: r.invitation.inviterId,
      to: r.user.id,
    })
  }
}));
