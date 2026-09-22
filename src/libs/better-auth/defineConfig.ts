import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import {
  type BetterAuthOptions,
  type BetterAuthPlugin,
  betterAuth,
} from "better-auth";
import { admin, jwt, organization, testUtils } from "better-auth/plugins";
import type { User } from "@/auth";
import { appEnv } from "@/config";
import { schema, serverDB } from "@/database";
import { UserService } from "@/service/user";
import { initSocialProviders } from "./sso";

const { socialProviders } = initSocialProviders();
interface CustomBetterAuthOptions {
  plugins: BetterAuthPlugin[];
}

const baseOptions = {
  baseURL: appEnv.APP_URL,
  secret: appEnv.AUTH_SECRET,
  database: drizzleAdapter(serverDB, {
    provider: "pg",
    schemaName: "better-auth",
    schema,
    usePlural: true,
  }),
  socialProviders,
  plugins: [
    admin(),
    organization({
      teams: { enabled: true },
      schema: {
        organization: {
          additionalFields: {
            isPersonal: {
              type: "boolean",
              defaultValue: false,
              input: true,
              returned: true,
            },
          },
        },
        team: {
          additionalFields: {
            isPersonal: {
              type: "boolean",
              defaultValue: false,
              input: true,
              returned: true,
            },
          },
        },
      },
    }),
    jwt(),
    testUtils(),
  ],
  session: {
    additionalFields: {
      activeWorkspaceId: {
        type: "string",
        required: true,
      },
    },
  },
  advanced: {
    database: {
      generateId: "uuid",
    },
  },
  databaseHooks: {
    user: {
      create: {
        async after(user) {
          const svc = new UserService(serverDB);
          await svc.initPersonalWorkspace(user as User);
        },
      },
    },
    session: {
      create: {
        async before(session) {
          const svc = new UserService(serverDB);
          const ws = await svc.findPersonalWorkspace(session.userId);

          if (ws?.team) {
            session.activeOrganizationId = ws.team.organizationId;
            session.activeTeamId = ws.team.id;
            session.activeWorkspaceId = ws.id;
          }

          return { data: { ...session } };
        },
      },
    },
  },
  account: {
    modelName: "userAccount",
  },
} satisfies BetterAuthOptions;

export function defineConfig(customOptions: CustomBetterAuthOptions) {
  const options = {
    ...baseOptions,
    plugins: [
      ...customOptions.plugins,
      ...(appEnv.TESTING ? [testUtils()] : []),
      ...baseOptions.plugins,
    ],
  } satisfies BetterAuthOptions;

  const auth = betterAuth(options) as unknown as ReturnType<
    typeof betterAuth<typeof baseOptions>
  >;

  return auth;
}
