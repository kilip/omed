import { createEnv } from "@t3-oss/env-core";
import z from "zod";

const {
  APP_URL,
  AUTH_SECRET,
  VAULT_SECRET,
  NODE_ENV,
  DATABASE_DRIVER,
  DATABASE_URL,
  AUTH_GOOGLE_ID,
  AUTH_GOOGLE_SECRET,
} = process.env;

export const appEnvConfig = () => {
  return createEnv({
    server: {
      APP_URL: z.string(),
      AUTH_SECRET: z.string(),
      VAULT_SECRET: z.string(),
      DATABASE_DRIVER: z.enum(["pglite", "pglite-memory", "node", "neon"]),
      DATABASE_URL: z.string(),
      TESTING: z.boolean().default(false),
      DEVELOPMENT: z.boolean().default(false),
      AUTH_GOOGLE_ID: z.string().optional(),
      AUTH_GOOGLE_SECRET: z.string().optional(),
    },
    runtimeEnv: {
      APP_URL: APP_URL ?? "http://localhost:3000",
      AUTH_SECRET,
      VAULT_SECRET,
      DATABASE_DRIVER,
      DATABASE_URL,
      TESTING: NODE_ENV === "test",
      DEVELOPMENT: NODE_ENV !== "production",
      AUTH_GOOGLE_ID,
      AUTH_GOOGLE_SECRET,
    },
    isServer: typeof window === "undefined" || NODE_ENV === "test",
  });
};

export const appEnv = appEnvConfig();
