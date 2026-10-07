import { authEnv } from "@omed/better-auth";
import { defineConfig } from "drizzle-kit";

export default defineConfig({
	out: "../../packages/better-auth/migrations",
	schema: "../../packages/better-auth/src/drizzle/schema/index.ts",
	dialect: "postgresql",
	schemaFilter: ["auth"],
	dbCredentials: {
		url: authEnv.AUTH_DB_URL,
	},
});
