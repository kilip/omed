import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

export const appEnvConfig = () => {
  return createEnv({
    /*
     * Tell T3 Env to look for variables starting with VITE_
     */
    clientPrefix: "VITE_",

    /*
     * Define your client-side environment variables here.
     * They MUST start with your chosen clientPrefix.
     */
    client: {
      VITE_AUTH_URL: z.string().default("http://localhost:9001"),
      VITE_PUBLIC_URL: z.string().default("http://localhost:3001"),
    },

    /*
     * Pass Vite's native environment object so T3 Env can read it at runtime.
     */
    runtimeEnv: import.meta.env,

    /*
     * Treats empty strings ("") as undefined, triggering Zod validation errors.
     */
    emptyStringAsUndefined: true,
  });
};

type AppEnv = ReturnType<typeof appEnvConfig>;
let cachedEnv: AppEnv | null = null;

const buildEnv = () => {
  if (null === cachedEnv) {
    cachedEnv = appEnvConfig();
  }

  return cachedEnv;
};

export const appEnv: AppEnv = buildEnv();
