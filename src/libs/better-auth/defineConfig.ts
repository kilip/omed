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
import { getTestDB } from "@/database/core/getTestDB";
import { UserService } from "@/service/user";

interface CustomBetterAuthOptions {
  plugins: BetterAuthPlugin[];
}

const db = appEnv.TESTING ? await getTestDB() : serverDB;
const baseOptions = {
  baseURL: appEnv.BASE_URL,
  secret: appEnv.AUTH_SECRET,
  database: drizzleAdapter(db, {
    provider: "pg",
    schemaName: "better-auth",
    schema,
  }),
  plugins: [admin(), organization(), jwt(), testUtils()],
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
          const svc = new UserService(db);
          await svc.initUser(user as User);
          console.log(user);
        },
      },
    },
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
