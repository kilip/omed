import { defineConfig } from "drizzle-kit";
import { appEnv } from "@/config";

const pglite = {
  driver: "pglite",
};

export default defineConfig({
  dialect: "postgresql",
  out: "./src/database/migrations",
  schema: "./src/database/schema/index.ts",
  ...(appEnv.DATABASE_DRIVER === "pglite" ? pglite : {}),
  dbCredentials: {
    url: appEnv.DATABASE_URL,
  },
});
