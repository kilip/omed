import type { BetterAuthPlugin } from "better-auth";
import { createAuthEndpoint, sessionMiddleware } from "better-auth/api";
import { z } from "zod";
import { serverDB } from "@/database";

export const workspacePlugin = () =>
  ({
    id: "workspacePlugin",
    endpoints: {
      switchWorkspace: createAuthEndpoint(
        "/workspace/switch",
        {
          method: "POST",
          body: z.object({
            workspaceId: z.string(),
          }),
          use: [sessionMiddleware],
        },
        async (ctx) => {
          const { workspaceId } = ctx.body;
          const { session, user } = ctx.context.session;

          const workspace = await serverDB.query.workspaces.findFirst({
            where: {
              id: workspaceId,
            },
            with: {
              team: true,
            },
          });

          if (!workspace) {
            return ctx.json({ session });
          }

          const updated = await ctx.context.internalAdapter.updateSession(
            session.id,
            {
              activeWorkspaceId: workspaceId,
              activeTeamId: workspace.teamId,
              ...(workspace.team
                ? { activeOrganizationId: workspace.team.organizationId }
                : {}),
            },
          );

          return ctx.json({ session: updated });
        },
      ),
    },
  }) satisfies BetterAuthPlugin;
