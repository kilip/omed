import type { auth } from "./better-auth/auth";

export type SessionData = typeof auth.$Infer.Session;
export type Session = typeof auth.$Infer.Session.session;
export type User = typeof auth.$Infer.Session.user;
export type Organization = typeof auth.$Infer.Organization;
export type Team = typeof auth.$Infer.Team;
