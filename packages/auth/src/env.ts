import { createEnv } from "@t3-oss/env-core";
import z from "zod";

export const authEnvConfig = () => {
  return createEnv({
    server: {
      AUTH_BASE_URL: z.string().default("http://localhost:3000"),
      AUTH_BASE_PATH: z.string().default("/api"),
      AUTH_SECRET: z.string(),
      AUTH_DB_URL: z.string(),
      AUTH_DB_DRIVER: z.enum(["node", "neon"]),
      AUTH_SCHEMA_NAME: z.string().default("auth"),
      AUTH_GOOGLE_ID: z.string().default("google-client-id"),
      AUTH_GOOGLE_SECRET: z.string().default("google-client-secret"),
      AUTH_GITHUB_ID: z.string().default("github-client-id"),
      AUTH_GITHUB_SECRET: z.string().default("github-client-secret"),
    },
    runtimeEnv: process.env,
    isServer: typeof window === "undefined" || process.env.NODE_ENV === "test",
  });
};

export const authEnv = authEnvConfig();
