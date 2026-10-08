import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

export const appEnvConfig = () => {
  return createEnv({
    clientPrefix: "VITE_",
    client: {
      VITE_AUTH_URL: z.string().default("http://localhost:8001"),
      VITE_AUTH_PATH: z.string().default(""),
      VITE_FINANCE_URL: z.string().default("http://localhost:8002"),
    },
    runtimeEnv: import.meta.env,
    emptyStringAsUndefined: true,
  });
};

export const appEnv = appEnvConfig();
