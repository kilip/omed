import { appEnv } from "./config";
import { defineConfig } from "./libs/better-auth/defineConfig";

export const auth = defineConfig({
  ...(appEnv.OMED_COOKIE_PREFIX && { cookiePrefix: appEnv.OMED_COOKIE_PREFIX }),
  plugins: [],
});

export type Session = typeof auth.$Infer.Session.session;
export type User = typeof auth.$Infer.Session.user;
