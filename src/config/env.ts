import { createEnv } from "@t3-oss/env-nextjs";
import z from "zod";

const {
  OMED_URL,
  OMED_SECRET,
  OMED_DB_DRIVER,
  OMED_DB_URL,
  OMED_DB_STATEMENT_TIMEOUT,
  OMED_COOKIE_PREFIX,
  GOOGLE_CLIENT_ID,
  GOOGLE_CLIENT_SECRET,
} = process.env;

export const appEnvConfig = () => {
  return createEnv({
    server: {
      OMED_URL: z.string(),
      OMED_SECRET: z.string(),
      OMED_DB_DRIVER: z.enum(["neon", "node"]),
      OMED_DB_URL: z.string(),
      OMED_DB_STATEMENT_TIMEOUT: z.coerce.number().optional(),
      OMED_COOKIE_PREFIX: z.string().optional(),
      GOOGLE_CLIENT_ID: z.string(),
      GOOGLE_CLIENT_SECRET: z.string(),
    },
    runtimeEnv: {
      OMED_URL,
      OMED_SECRET,
      OMED_DB_DRIVER: OMED_DB_DRIVER || "neon",
      OMED_DB_URL,
      OMED_DB_STATEMENT_TIMEOUT,
      OMED_COOKIE_PREFIX,
      GOOGLE_CLIENT_ID,
      GOOGLE_CLIENT_SECRET,
    },
    isServer: typeof window === "undefined" || process.env.NODE_ENV === "test",
  });
};

export const appEnv = appEnvConfig();
