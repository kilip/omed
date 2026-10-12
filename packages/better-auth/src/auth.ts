import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import {
  type BetterAuthOptions,
  betterAuth,
  type SocialProviders,
} from "better-auth";
import { admin, jwt, openAPI } from "better-auth/plugins";
import { authEnv } from "./authEnv";
import { authDb } from "./drizzle/db";
import * as schema from "./drizzle/schema/better-auth";

const socialProviders = {
  ...(authEnv.OMED_GOOGLE_ID && authEnv.OMED_GOOGLE_SECRET
    ? {
        google: {
          clientId: authEnv.OMED_GOOGLE_ID,
          clientSecret: authEnv.OMED_GOOGLE_SECRET,
        },
      }
    : {}),
  ...(authEnv.OMED_GITHUB_ID && authEnv.OMED_GITHUB_SECRET
    ? {
        github: {
          clientId: authEnv.OMED_GITHUB_ID,
          clientSecret: authEnv.OMED_GITHUB_SECRET,
        },
      }
    : {}),
} satisfies SocialProviders;

export const options = {
  baseURL: authEnv.OMED_AUTH_URL,
  basePath: authEnv.OMED_AUTH_PATH,
  trustedOrigins: authEnv.OMED_TRUSTED_ORIGINS,
  database: drizzleAdapter(authDb, {
    provider: "pg",
    schemaName: "auth",
    schema,
  }),
  socialProviders,
  plugins: [admin(), jwt(), openAPI()],
} satisfies BetterAuthOptions;

export const auth = betterAuth(options);
export { authEnv };
