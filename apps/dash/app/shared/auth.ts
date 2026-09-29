"use client";
import type { auth } from "@omed/better-auth";
import {
  adminClient,
  inferAdditionalFields,
  jwtClient,
  organizationClient,
} from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";
import { appEnv } from "~/env";

export const { signOut, signIn, getSession, token } = createAuthClient({
  baseURL: appEnv.VITE_AUTH_URL,
  basePath: "",
  plugins: [
    adminClient(),
    organizationClient({ teams: { enabled: true } }),
    jwtClient(),
    inferAdditionalFields<typeof auth>(),
  ],
});

export type User = typeof auth.$Infer.Session.user;
