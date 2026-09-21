import { createEnv } from "@t3-oss/env-nextjs";
import z from "zod";

const {
  BASE_URL,
  AUTH_SECRET,
  VAULT_SECRET,
  NODE_ENV,
  DATABASE_DRIVER,
  DATABASE_URL,
} = process.env;

export const appEnvConfig = () => {
  return createEnv({
    server: {
      BASE_URL: z.string(),
      AUTH_SECRET: z.string(),
      VAULT_SECRET: z.string(),
      DATABASE_DRIVER: z.enum(["pglite", "pglite-memory", "node", "neon"]),
      DATABASE_URL: z.string(),
      TESTING: z.boolean().default(false),
      DEVELOPMENT: z.boolean().default(false),
    },
    runtimeEnv: {
      BASE_URL: BASE_URL ?? "http://localhost:3000",
      AUTH_SECRET,
      VAULT_SECRET,
      DATABASE_DRIVER,
      DATABASE_URL,
      TESTING: NODE_ENV === "test",
      DEVELOPMENT: NODE_ENV !== "production",
    },
    isServer: typeof window === "undefined" || NODE_ENV === "test",
  });
};

export const appEnv = appEnvConfig();
