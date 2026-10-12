import { defineConfig } from "drizzle-kit";
import { authEnv } from "./src/authEnv";

export default defineConfig({
  out: "./migrations",
  schema: "./src/drizzle/schema/index.ts",
  schemaFilter: ["auth"],
  dialect: "postgresql",
  dbCredentials: {
    url: authEnv.OMED_DB_URL,
  },
});
