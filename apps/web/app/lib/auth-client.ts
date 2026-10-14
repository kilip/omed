import { createOmedAuthClient } from "@omed/better-auth/client";
import { env } from "./env";

export const authClient = createOmedAuthClient({ baseURL: env.authUrl });
