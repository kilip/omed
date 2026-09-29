import type { Auth, BetterAuthOptions } from "better-auth";
import type { BaseOptions } from "./auth.options";

export type User = Auth<BaseOptions>["$Infer"]["Session"]["user"];
export type Session = Auth<BaseOptions>["$Infer"]["Session"]["session"];
export type Organization = Auth<BaseOptions>["$Infer"]["Organization"];
export type DatabaseHooks = BetterAuthOptions["databaseHooks"];
export type ActiveTeam = {
  id: string;
  name: string;
  organizationId: string;
  personal: boolean;
};

export type ActiveWorkspace = ActiveTeam & {
  activeWorkspaceId: string;
  activeWorkspaceName: string;
  activeWorkspaceRoles: string[];
};
