import type { auth } from "@omed/better-auth";
import {
  adminClient,
  inferAdditionalFields,
  jwtClient,
  organizationClient,
} from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";
import { appEnv } from "./appEnv";

export const authClient = createAuthClient({
  baseURL: appEnv.VITE_AUTH_URL,
  basePath: appEnv.VITE_AUTH_PATH,
  fetchOptions: {
    credentials: "include",
  },
  plugins: [
    adminClient(),
    organizationClient({
      teams: {
        enabled: true,
      },
    }),
    jwtClient(),
    inferAdditionalFields<typeof auth>(),
  ],
});

export const { getSession, getAccessToken, signIn, signOut, token } =
  authClient;
export type { Organization, Session, User } from "@omed/better-auth";
