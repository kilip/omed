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
    organization({ teams: { enabled: true } }),
    jwt(),
    testUtils(),
  ],
  user: {
    additionalFields: {
      activeWorkspace: {
        type: "string",
        returned: true,
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
          await svc.initUser(user as User);
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
