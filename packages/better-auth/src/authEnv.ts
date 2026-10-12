import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

export const authEnvConfig = () => {
  return createEnv({
    server: {
      OMED_DB_URL: z.string(),
      OMED_DB_DRIVER: z.enum(["node", "neon"]).default("node"),
      OMED_AUTH_URL: z.url(),
      OMED_AUTH_PATH: z.string().optional().default("/auth"),
      OMED_API_URL: z.string(),
      OMED_WEB_URL: z.string(),
      OMED_TRUSTED_ORIGINS: z
        .string()
        .optional()
        .default("")
        .transform((value) =>
          value
            ? value
                .split(",")
                .map((origin) => origin.trim())
                .filter(Boolean)
            : [],
        ),
      OMED_AUTH_SECRET: z.string(),
      OMED_GOOGLE_ID: z.string().optional(),
      OMED_GOOGLE_SECRET: z.string().optional(),
      OMED_GITHUB_ID: z.string().optional(),
      OMED_GITHUB_SECRET: z.string().optional(),
      OMED_JWKS_URL: z.string().optional(),
    },
    runtimeEnv: process.env,
  });
};

export const authEnv = authEnvConfig();
