import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import {
  type BetterAuthOptions,
  type BetterAuthPlugin,
  betterAuth,
} from "better-auth";
import { admin, organization } from "better-auth/plugins";
import { appEnv } from "@/config";
import { schema, serverDB } from "@/database";

interface CustomBetterAuthOptions {
  plugins: BetterAuthPlugin[];
}
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
      ...(customOptions.plugins ? customOptions.plugins : []),
      admin(),
      organization(),
    ],
  } satisfies BetterAuthOptions;

  const auth = betterAuth(options);

  return auth;
}
