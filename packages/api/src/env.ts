import { createEnv } from "@t3-oss/env-core";
import z from "zod";

const apiEnvConfig = () => {
  return createEnv({
    server: {
      API_FINANCE_URL: z.string().optional().default("http://localhost:3001"),
    },
    runtimeEnv: process.env,
    isServer: typeof window === "undefined" || process.env.NODE_ENV === "test",
  });
};

export const apiEnv = apiEnvConfig();
