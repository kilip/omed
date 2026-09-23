import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import { betterAuth } from "better-auth";
import { admin, jwt, organization } from "better-auth/plugins";
import { authDB } from "./drizzle";
import * as schema from "./drizzle/schema";
import { authEnv } from "./env";

export const auth = betterAuth({
  baseURL: authEnv.AUTH_BASE_URL,
  secret: authEnv.AUTH_SECRET,
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
    }),
    jwt({}),
  ],
});
