import { createEnv } from "@t3-oss/env-core";
import z from "zod";

export const authEnvConfig = () => {
  return createEnv({
    server: {
      AUTH_BASE_URL: z.string().default("http://localhost:9001"),
      AUTH_BASE_PATH: z.string().default("/auth"),
      AUTH_SECRET: z.string().default("0vK4SBoPHS2Rq0i9zfFwRmpqWH6XAvf7"),
      AUTH_DB_DRIVER: z.enum(["node", "neon"]).default("neon"),
      AUTH_DB_URL: z.string().default("database-url"),
    },
    runtimeEnv: process.env,
  });
};

export const authEnv = authEnvConfig();
