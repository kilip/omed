import { createEnv } from "@t3-oss/env-core";
import z from "zod";

export const authEnvConfig = () => {
  return createEnv({
    server: {
      DEVELOPMENT: z.boolean().default(process.env.NODE_ENV !== "production"),
      AUTH_URL: z.string().default("http://localhost:9001"),
      AUTH_PATH: z.string().default(""),
      AUTH_TRUSTED_ORIGINS: z
        .string()
        .transform((v) => v.split(/[\s,]+/).filter(Boolean))
        .pipe(z.array(z.url()))
        .default(["http://localhost:3001"]),
      AUTH_SECRET: z.string(),
      AUTH_DB_DRIVER: z.enum(["node", "neon"]),
      AUTH_DB_URL: z.string("postgres://omed:omed@localhost/omed"),
      AUTH_GOOGLE_ID: z.string().optional(),
      AUTH_GOOGLE_SECRET: z.string().optional(),
      AUTH_GITHUB_ID: z.string().optional(),
      AUTH_GITHUB_SECRET: z.string().optional(),
    },
    runtimeEnv: process.env,
  });
};

export const authEnv = authEnvConfig();
