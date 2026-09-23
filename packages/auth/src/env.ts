import { createEnv } from "@t3-oss/env-core";
import z from "zod";

export const authEnvConfig = () => {
  return createEnv({
    server: {
      AUTH_BASE_URL: z.string().default("http://localhost:3000"),
      AUTH_SECRET: z.string(),
      AUTH_DB_URL: z.string(),
      AUTH_DB_DRIVER: z.enum(["node", "neon"]),
      AUTH_SCHEMA_NAME: z.string().default("auth"),
    },
    runtimeEnv: process.env,
    isServer: typeof window === "undefined" || process.env.NODE_ENV === "test",
  });
};

export const authEnv = authEnvConfig();
