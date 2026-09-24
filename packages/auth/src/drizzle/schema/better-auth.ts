import { defineRelationsPart, sql } from "drizzle-orm";
import {
  pgSchema,
  text,
  timestamp,
  boolean,
  integer,
  uuid,
  index,
  uniqueIndex,
} from "drizzle-orm/pg-core";

export const authSchema = pgSchema("auth");

export const users = authSchema.table("users", {
  id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: boolean("emailVerified").default(false).notNull(),
  image: text("image"),
  createdAt: timestamp("createdAt").notNull(),
  updatedAt: timestamp("updatedAt")
    .$onUpdate(() => new Date())
    .notNull(),
  role: text("role"),
  banned: boolean("banned").default(false),
  banReason: text("banReason"),
  banExpires: timestamp("banExpires"),
});

export const sessions = authSchema.table(
  "sessions",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    expiresAt: timestamp("expiresAt").notNull(),
    token: text("token").notNull().unique(),
    createdAt: timestamp("createdAt").notNull(),
    updatedAt: timestamp("updatedAt")
      .$onUpdate(() => new Date())
      .notNull(),
    ipAddress: text("ipAddress"),
    userAgent: text("userAgent"),
    userId: uuid("userId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    impersonatedBy: text("impersonatedBy"),
    activeOrganizationId: text("activeOrganizationId"),
    activeTeamId: text("activeTeamId"),
  },
  (table) => [index("sessions_userId_idx").on(table.userId)],
);

export const accounts = authSchema.table(
  "accounts",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    accountId: text("accountId").notNull(),
    providerId: text("providerId").notNull(),
    userId: uuid("userId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    accessToken: text("accessToken"),
    refreshToken: text("refreshToken"),
    idToken: text("idToken"),
    accessTokenExpiresAt: timestamp("accessTokenExpiresAt"),
    refreshTokenExpiresAt: timestamp("refreshTokenExpiresAt"),
    scope: text("scope"),
    password: text("password"),
    createdAt: timestamp("createdAt").notNull(),
    updatedAt: timestamp("updatedAt")
      .$onUpdate(() => new Date())
      .notNull(),
  },
  (table) => [index("accounts_userId_idx").on(table.userId)],
);

export const verifications = authSchema.table(
  "verifications",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    identifier: text("identifier").notNull(),
    value: text("value").notNull(),
    expiresAt: timestamp("expiresAt").notNull(),
    createdAt: timestamp("createdAt").notNull(),
    updatedAt: timestamp("updatedAt")
      .$onUpdate(() => new Date())
      .notNull(),
  },
  (table) => [index("verifications_identifier_idx").on(table.identifier)],
);

export const organizations = authSchema.table(
  "organizations",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    name: text("name").notNull(),
    slug: text("slug").notNull().unique(),
    logo: text("logo"),
    createdAt: timestamp("createdAt").notNull(),
    metadata: text("metadata"),
    isPersonal: boolean("isPersonal").default(false),
  },
  (table) => [
    uniqueIndex("organizations_slug_uidx").on(table.slug),
    index("organizations_isPersonal_idx").on(table.isPersonal),
  ],
);

export const teams = authSchema.table(
  "teams",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    name: text("name").notNull(),
    memberCount: integer("memberCount").default(0).notNull(),
    organizationId: uuid("organizationId")
      .notNull()
      .references(() => organizations.id, { onDelete: "cascade" }),
    createdAt: timestamp("createdAt").notNull(),
    updatedAt: timestamp("updatedAt").$onUpdate(() => new Date()),
    isPersonal: boolean("isPersonal").default(false),
  },
  (table) => [
    index("teams_organizationId_idx").on(table.organizationId),
    index("teams_isPersonal_idx").on(table.isPersonal),
  ],
);

export const teamMembers = authSchema.table(
  "teamMembers",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    teamId: uuid("teamId")
      .notNull()
      .references(() => teams.id, { onDelete: "cascade" }),
    userId: uuid("userId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    membershipKey: text("membershipKey").unique(),
    createdAt: timestamp("createdAt"),
  },
  (table) => [
    index("teamMembers_teamId_idx").on(table.teamId),
    index("teamMembers_userId_idx").on(table.userId),
  ],
);

export const members = authSchema.table(
  "members",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    organizationId: uuid("organizationId")
      .notNull()
      .references(() => organizations.id, { onDelete: "cascade" }),
    userId: uuid("userId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    role: text("role").default("member").notNull(),
    createdAt: timestamp("createdAt").notNull(),
  },
  (table) => [
    index("members_organizationId_idx").on(table.organizationId),
    index("members_userId_idx").on(table.userId),
  ],
);

export const invitations = authSchema.table(
  "invitations",
  {
    id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
    organizationId: uuid("organizationId")
      .notNull()
      .references(() => organizations.id, { onDelete: "cascade" }),
    email: text("email").notNull(),
    role: text("role"),
    teamId: text("teamId"),
    status: text("status").default("pending").notNull(),
    expiresAt: timestamp("expiresAt").notNull(),
    createdAt: timestamp("createdAt").notNull(),
    inviterId: uuid("inviterId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
  },
  (table) => [
    index("invitations_organizationId_idx").on(table.organizationId),
    index("invitations_email_idx").on(table.email),
  ],
);

export const jwkss = authSchema.table("jwkss", {
  id: uuid("id").default(sql`pg_catalog.gen_random_uuid()`).primaryKey(),
  publicKey: text("publicKey").notNull(),
  privateKey: text("privateKey").notNull(),
  createdAt: timestamp("createdAt").notNull(),
  expiresAt: timestamp("expiresAt"),
  alg: text("alg"),
  crv: text("crv"),
});

export const authRelations = defineRelationsPart(
  {
    users,
    sessions,
    accounts,
    verifications,
    organizations,
    teams,
    teamMembers,
    members,
    invitations,
    jwkss,
  },
  (r) => ({
    users: {
      sessions: r.many.sessions({
        from: r.users.id,
        to: r.sessions.userId,
      }),
      accounts: r.many.accounts({
        from: r.users.id,
        to: r.accounts.userId,
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
      }),
    },
    sessions: {
      user: r.one.users({
        from: r.sessions.userId,
        to: r.users.id,
      }),
    },
    accounts: {
      user: r.one.users({
        from: r.accounts.userId,
        to: r.users.id,
      }),
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
      }),
    },
    teams: {
      organization: r.one.organizations({
        from: r.teams.organizationId,
        to: r.organizations.id,
      }),
      teamMembers: r.many.teamMembers({
        from: r.teams.id,
        to: r.teamMembers.teamId,
      }),
    },
    teamMembers: {
      team: r.one.teams({
        from: r.teamMembers.teamId,
        to: r.teams.id,
      }),
      user: r.one.users({
        from: r.teamMembers.userId,
        to: r.users.id,
      }),
    },
    members: {
      organization: r.one.organizations({
        from: r.members.organizationId,
        to: r.organizations.id,
      }),
      user: r.one.users({
        from: r.members.userId,
        to: r.users.id,
      }),
    },
    invitations: {
      organization: r.one.organizations({
        from: r.invitations.organizationId,
        to: r.organizations.id,
      }),
      user: r.one.users({
        from: r.invitations.inviterId,
        to: r.users.id,
      }),
    },
  }),
);
