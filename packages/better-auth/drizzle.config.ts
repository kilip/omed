import { defineConfig } from "drizzle-kit";
import { authEnv } from "./src";

export default defineConfig({
  out: "./migrations",
  schema: "./src/drizzle/schema/index.ts",
  dialect: "postgresql",
  dbCredentials: {
    url: authEnv.AUTH_DB_URL,
  },
});
