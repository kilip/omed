import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import {
  type BetterAuthOptions,
  type BetterAuthPlugin,
  betterAuth,
} from "better-auth";
import { admin, jwt, organization, testUtils } from "better-auth/plugins";
import { appEnv } from "@/config";
import { schema, serverDB } from "@/database";

interface CustomBetterAuthOptions {
  plugins: BetterAuthPlugin[];
}

const baseOptions = {
  plugins: [admin(), organization(), jwt()],
} satisfies BetterAuthOptions;

export function defineConfig(customOptions: CustomBetterAuthOptions) {
  const options = {
    baseURL: appEnv.BASE_URL,
    secret: appEnv.AUTH_SECRET,
    database: drizzleAdapter(serverDB, {
      provider: "pg",
      schemaName: "better-auth",
      schema,
    }),
    plugins: [
      ...customOptions.plugins,
      ...(appEnv.TESTING ? [testUtils()] : []),
      admin(),
      organization(),
      jwt(),
    ],
    advanced: {
      database: {
        generateId: "uuid",
      },
    },
  } satisfies BetterAuthOptions;

  const auth = betterAuth(options) as unknown as ReturnType<
    typeof betterAuth<typeof baseOptions>
  >;

  return auth;
}
