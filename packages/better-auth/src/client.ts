import { adminClient } from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";

export const { signIn, signOut, getSession } = createAuthClient({
  baseURL: process.env.VITE_AUTH_URL,
  basePath: "",
  plugins: [adminClient()],
});
