import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import {
  type BetterAuthOptions,
  type BetterAuthPlugin,
  betterAuth,
} from "better-auth";
import { admin, organization, testUtils } from "better-auth/plugins";
import { appEnv } from "@/config";
import { schema, serverDB } from "@/database";
import type { OmedDatabase } from "@/database/type";

interface CustomBetterAuthOptions {
  db?: OmedDatabase;
  cookieDomain?: string;
  cookiePrefix?: string;
  plugins: BetterAuthPlugin[];
}

export function defineConfig(customOptions: CustomBetterAuthOptions) {
  const options = {
    baseURL: appEnv.OMED_URL,
    secret: appEnv.OMED_SECRET,
    database: drizzleAdapter(customOptions.db ?? serverDB, {
      provider: "pg",
      schemaName: "better-auth",
      schema,
    }),
    plugins: [
      ...customOptions.plugins,
      admin(),
      organization(),
      ...(process.env.NODE_ENV === "test" ? [testUtils()] : []),
    ],
    advanced: {
      database: {
        generateId: "uuid",
      },
    },
    socialProviders: {
      google: {
        clientId: appEnv.GOOGLE_CLIENT_ID,
        clientSecret: appEnv.GOOGLE_CLIENT_SECRET,
      },
    },
  } satisfies BetterAuthOptions;

  const auth = betterAuth(options);
  return auth;
}
