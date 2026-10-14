import { jwtClient } from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";

export type OmedAuthClientOptions = {
  /** Public URL of the auth service (the JWT `iss`/`aud`). */
  baseURL: string;
  /** Mount path of Better Auth on the auth service. */
  basePath?: string;
};

export const createOmedAuthClient = ({
  baseURL,
  basePath = "/auth",
}: OmedAuthClientOptions) =>
  createAuthClient({
    baseURL,
    basePath,
    plugins: [jwtClient()],
  });

export type OmedAuthClient = ReturnType<typeof createOmedAuthClient>;
