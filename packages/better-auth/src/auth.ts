import { betterAuth } from "better-auth";
import { authOptions } from "./options";
import { AuthService } from "./service";
import type { AuthContext, Session, User } from "./type";

let authSvc: AuthService | null = null;
export async function getAuthService(): Promise<AuthService> {
  if (null === authSvc) {
    const { adapter, internalAdapter } =
      (await auth.$context) as unknown as AuthContext;
    authSvc = new AuthService(adapter, internalAdapter, auth.api);
  }
  return authSvc;
}

export const auth = betterAuth({
  ...authOptions,
  databaseHooks: {
    user: {
      create: {
        after: async (user) => {
          // Example: Add custom logic after a user is created
          const svc = await getAuthService();
          await svc.createWorkspace(user as User);
        },
      },
    },
    session: {
      create: {
        before: async (session: Session) => {
          const service = await getAuthService();
          const team = await service.findActiveTeam(session.userId);
          if (!team) return { data: session };
          return {
            data: {
              ...session,
              activeTeamId: team.id,
              activeOrganizationId: team.organizationId,
              activeWorkspaceId: team.activeWorkspaceId,
              activeWorkspaceName: team.activeWorkspaceName,
              activeWorkspaceRoles: team.activeWorkspaceRoles,
            },
          };
        },
      },
    },
  },
});
