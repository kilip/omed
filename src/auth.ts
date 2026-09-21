import { defineConfig } from "./libs/better-auth";

export const auth = defineConfig({
  plugins: [],
});

export type Session = typeof auth.$Infer.Session.session;
export type User = typeof auth.$Infer.Session.user;
export type Organization = typeof auth.$Infer.Organization;
export type Team = typeof auth.$Infer.Team;
export type Member = typeof auth.$Infer.Member;
