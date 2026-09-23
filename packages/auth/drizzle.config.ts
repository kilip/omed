import { defineConfig } from "drizzle-kit";
import { authEnv } from "./src/env.ts";

export default defineConfig({
  out: "./migrations",
  schema: "./src/drizzle/schema/index.ts",
  strict: true,
  dialect: "postgresql",
  dbCredentials: {
    url: authEnv.AUTH_DB_URL,
  },
});
