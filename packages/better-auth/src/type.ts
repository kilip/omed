import type { Auth } from "better-auth";
import type { authOptions } from "./options";

export type AuthType = Auth<typeof authOptions>;
export type AuthContext = Awaited<AuthType["$context"]>;
export type User = AuthType["$Infer"]["Session"]["user"];
export type Organization = AuthType["$Infer"]["Organization"];
export type Session = AuthType["$Infer"]["Session"]["session"];
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
