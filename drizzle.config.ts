import { defineConfig } from "drizzle-kit";
import { appEnv } from "@/config";

export default defineConfig({
  dialect: "postgresql",
  out: "./src/database/migrations",
  schema: "./src/database/schema/index.ts",
  driver: "pglite",
  dbCredentials: {
    url: appEnv.DATABASE_URL,
  },
});
