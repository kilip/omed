import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import { betterAuth } from "better-auth";
import { admin, openAPI, organization } from "better-auth/plugins";
import { authEnv } from "./config";
import { authDB } from "./drizzle";
import * as schema from "./drizzle/schema";

export const auth = betterAuth({
  telemetry: {
    enabled: false,
  },
  baseURL: authEnv.AUTH_BASE_URL,
  secret: authEnv.AUTH_SECRET,
  basePath: authEnv.AUTH_BASE_PATH,
  database: drizzleAdapter(authDB, {
    provider: "pg",
    schemaName: "auth",
    schema,
  }),
  plugins: [admin(), organization({ teams: { enabled: true } }), openAPI()],
});
