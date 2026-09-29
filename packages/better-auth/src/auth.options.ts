import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import type { Auth, BetterAuthOptions } from "better-auth";
import {
  admin,
  jwt,
  openAPI,
  organization,
  testUtils,
} from "better-auth/plugins";
import { authEnv } from "../src/config";
import { authDB } from "../src/drizzle";
import * as schema from "../src/drizzle/schema";
import type { Session } from "./type";

export const betterAuthOptions = {
  baseURL: authEnv.AUTH_BASE_URL,
  secret: authEnv.AUTH_SECRET,
  basePath: authEnv.AUTH_BASE_PATH,
  trustedOrigins: authEnv.AUTH_TRUSTED_ORIGINS,
  database: drizzleAdapter(authDB, {
    provider: "pg",
    schemaName: "auth",
    schema,
  }),
  advanced: {
    database: {
      generateId: "uuid",
    },
  },
  plugins: [
    admin(),
    organization({
      teams: {
        enabled: true,
        defaultTeam: {
          enabled: false,
        },
      },
      schema: {
        team: {
          additionalFields: {
            personal: {
              fieldName: "personal",
              type: "boolean",
              defaultValue: false,
              returned: true,
              input: true,
            },
          },
        },
        organization: {
          additionalFields: {
            personal: {
              type: "boolean",
              defaultValue: false,
              returned: true,
              input: true,
            },
          },
        },
      },
    }),
    jwt({
      jwt: {
        definePayload(session) {
          return {
            id: session.user.id,
            name: session.user.name,
            avatar: session.user?.image,
            activeWorkspace: session.user.activeWorkspace,
          };
        },
      },
    }),
    openAPI(),
    ...(authEnv.DEVELOPMENT ? [testUtils()] : []),
  ],
  socialProviders: {
    ...(authEnv.AUTH_GOOGLE_ID && authEnv.AUTH_GOOGLE_SECRET
      ? {
          google: {
            clientId: authEnv.AUTH_GOOGLE_ID,
            clientSecret: authEnv.AUTH_GOOGLE_SECRET,
            scope: ["email", "profile", "openid"],
          },
        }
      : {}),
    ...(authEnv.AUTH_GITHUB_ID && authEnv.AUTH_GITHUB_SECRET
      ? {
          github: {
            clientId: authEnv.AUTH_GITHUB_ID,
            clientSecret: authEnv.AUTH_GITHUB_SECRET,
            scope: ["email", "profile", "openid"],
          },
        }
      : {}),
  },
  user: {
    additionalFields: {
      onBoarded: {
        type: "boolean",
        defaultValue: false,
        returned: true,
      },
      activeWorkspace: {
        type: "string",
        input: true,
        returned: true,
      },
    },
  },
  /*
  databaseHooks: {
    user: {
      create: {
        async after(u) {
          service.createPersonalWorkspace(u);
        },
      },
    },
    session: {
      create: {
        async before(session) {
          const team = await service.findActiveTeam(session.userId);
          if (!team) return { data: session };
          return {
            data: {
              ...session,
              activeTeamId: team.id,
              activeOrganizationId: team.organizationId,
            },
          };
        },
      },
    },
  },
  */
} satisfies BetterAuthOptions;

export type BaseOptions = typeof betterAuthOptions;
export type AuthType = Auth<BaseOptions>;
