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

export const authClient = createAuthClient({
  baseURL: appEnv.VITE_AUTH_URL,
  basePath: "",
  plugins: [
    adminClient(),
    organizationClient({ teams: { enabled: true } }),
    jwtClient(),
    inferAdditionalFields<typeof auth>(),
  ],
});

export const { signOut, signIn, getSession, token } = authClient;

export type User = typeof authClient.$Infer.Session.user;
export type Session = typeof authClient.$Infer.Session.session;
export * from "./cache";
