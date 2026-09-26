import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import { type BetterAuthOptions, betterAuth } from "better-auth";
import {
  admin,
  bearer,
  jwt,
  organization,
  testUtils,
} from "better-auth/plugins";
import { authDB } from "../drizzle";
import * as schema from "../drizzle/schema";
import { authEnv } from "../env";
import { UserService } from "../service";
import type { User } from "../type";
import { ac as orgAc, roles as orgRoles } from "./orgPermissions";
import { socialBearer } from "./plugins/socialBearer";

const userService = new UserService(authDB);

export const authDefaultOptions = {
  baseURL: authEnv.AUTH_BASE_URL,
  secret: authEnv.AUTH_SECRET,
  basePath: authEnv.AUTH_BASE_PATH,
  database: drizzleAdapter(authDB, {
    provider: "pg",
    camelCase: true,
    schemaName: authEnv.AUTH_SCHEMA_NAME,
    schema,
    usePlural: true,
  }),
  plugins: [
    admin(),
    organization({
      teams: {
        enabled: true,
      },
      schema: {
        organization: {
          additionalFields: {
            isPersonal: {
              type: "boolean",
              defaultValue: false,
              input: true,
              index: true,
            },
          },
        },
        team: {
          additionalFields: {
            isPersonal: {
              type: "boolean",
              defaultValue: false,
              input: true,
              index: true,
            },
          },
        },
      },
      ac: orgAc,
      roles: { ...orgRoles },
    }),
    jwt({
      sessionCookieCache: true,
      jwt: {
        definePayload: async ({ user, session }) => {
          let activeOrganizationRole = ["member"];
          let activeOrganizationName = "Undefined";
          let activeTeamName = "undefined";

          if (session?.activeTeamId) {
            const team = await userService.findTeam(session.activeTeamId);
            if (team) {
              activeTeamName = team?.name;
            }
          }

          if (session?.activeOrganizationId) {
            const membership = await userService.findMembership(
              user.id,
              session.ActiveOrganizationId,
            );
            if (membership) {
              activeOrganizationRole = membership.role.split(" ");
            }
            if (membership?.organization) {
              activeOrganizationName = membership?.organization?.name;
            }
          }
          return {
            id: user.id,
            name: user.name,
            avatar: user.image,
            role: user.role.split(" "),
            sessionId: session.id,
            activeOrganizationId: session?.activeOrganizationId,
            activeOrganizationName,
            activeTeamId: session?.activeTeamId,
            activeTeamName,
            activeOrganizationRole,
          };
        },
      },
    }),
    bearer(),
    socialBearer(),
    ...(process.env.NODE_ENV === "test" ? [testUtils()] : []),
  ],
  socialProviders: {
    google: {
      clientId: authEnv.AUTH_GOOGLE_ID,
      clientSecret: authEnv.AUTH_GOOGLE_SECRET,
    },
    github: {
      clientId: authEnv.AUTH_GITHUB_ID,
      clientSecret: authEnv.AUTH_GITHUB_SECRET,
    },
  },
  databaseHooks: {
    user: {
      create: {
        async after(user) {
          await userService.initPersonalTeam(user as User);
        },
      },
    },
    session: {
      create: {
        async before(session) {
          let activeOrganizationId = session.activeOrganizationId as
            | string
            | undefined;
          let activeTeamId = session.activeTeamId as string | undefined;

          if (!activeTeamId || !activeOrganizationId) {
            const team = await userService.findPersonalTeam(session.userId);
            if (!team) {
              throw new Error("can't find user personal team");
            }
            activeTeamId = team.id;
            activeOrganizationId = team.organizationId;
          }

          return {
            data: {
              ...session,
              activeOrganizationId,
              activeTeamId,
            },
          };
        },
      },
    },
  },
  advanced: {
    database: {
      generateId: "uuid",
    },
  },
  session: {
    cookieCache: {
      strategy: "jwt",
    },
  },
} satisfies BetterAuthOptions;

export const auth = betterAuth(authDefaultOptions);
