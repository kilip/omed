import { defineConfig } from "drizzle-kit";
import { appEnv } from "@/config";

export default defineConfig({
  dbCredentials: {
    url: appEnv.OMED_DB_URL,
  },
  dialect: "postgresql",
  out: "./src/database/migrations",
  schema: "./src/database/schema/index.ts",
  strict: true,
});
