import { betterAuth } from "better-auth";
import { type AuthType, betterAuthOptions } from "./auth.options";
import { AuthService } from "./service";

async function getService() {
  const { adapter, internalAdapter } = await auth.$context;
  return new AuthService(adapter, internalAdapter, auth.api);
}

export const auth = betterAuth({
  ...betterAuthOptions,
  databaseHooks: {
    user: {
      create: {
        async after(u) {
          const service = await getService();
          await service.createPersonalWorkspace(u);
        },
      },
    },
    session: {
      create: {
        async before(session) {
          const service = await getService();
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
}) as unknown as AuthType;
