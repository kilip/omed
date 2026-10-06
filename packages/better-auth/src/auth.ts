import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import { betterAuth } from "better-auth";
import { admin, jwt, openAPI, organization } from "better-auth/plugins";
import { authEnv } from "./authEnv";
import { authDB, schema } from "./drizzle";

export const auth = betterAuth({
	baseURL: authEnv.AUTH_URL,
  basePath: "",
	secret: authEnv.AUTH_SECRET,
  trustedOrigins: authEnv.AUTH_TRUSTED_ORIGINS,
	database: drizzleAdapter(authDB, {
		provider: "pg",
		schema,
		schemaName: "auth",
	}),
	plugins: [admin(), organization(), jwt(), openAPI()],
});
