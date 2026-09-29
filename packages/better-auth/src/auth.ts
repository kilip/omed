import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import { betterAuth } from "better-auth";
import { admin, jwt, openAPI, organization } from "better-auth/plugins";
import { authEnv } from "../src/config";
import { authDB } from "../src/drizzle";
import * as schema from "../src/drizzle/schema";

export const auth = betterAuth({
  baseURL: authEnv.AUTH_BASE_URL,
  secret: authEnv.AUTH_SECRET,
  basePath: authEnv.AUTH_BASE_PATH,
  trustedOrigins: [
    ...(authEnv.DEVELOPMENT
      ? [
          // dashboard:
          "http://localhost:3001",
          // finance api
          "http://localhost:9002",
        ]
      : []),
    "https://auth.itstoni.com",
    "https://auth.doyolabs.workers.dev",
  ],
  database: drizzleAdapter(authDB, {
    provider: "pg",
    schemaName: "auth",
    schema,
  }),
  plugins: [
    admin(),
    organization({ teams: { enabled: true } }),
    jwt({
      jwt: {
        definePayload(session) {
          return {
            id: session.user.id,
            name: session.user.name,
            avatar: session.user?.image,
          };
        },
      },
    }),
    openAPI(),
  ],
  socialProviders: {
    google: {
      clientId: authEnv.AUTH_GOOGLE_ID,
      clientSecret: authEnv.AUTH_GOOGLE_SECRET,
      scope: ["email", "profile", "openid"],
    },
  },
});
