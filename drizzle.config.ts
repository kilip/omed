import { defineConfig } from "drizzle-kit";
import { appEnv } from "@/config";

const _pglite = {
  driver: "pglite",
};

export default defineConfig({
  dialect: "postgresql",
  out: "./src/database/migrations",
  schema: "./src/database/schema/index.ts",
  dbCredentials: {
    url: appEnv.DATABASE_URL,
  },
});
