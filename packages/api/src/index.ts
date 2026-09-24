import createClient, { type Middleware } from "openapi-fetch";
import { apiEnv } from "./env";
import type { paths as financePaths } from "./schema/finance";

export const createApiClient = (accessToken: string) => {
  const authMiddleware: Middleware = {
    async onRequest({ request }) {
      request.headers.set("Authorization", `Bearer ${accessToken}`);
    },
  };
  return {
    finance: createClient<financePaths>({
      baseUrl: "http://localhost:3001",
    }).use(authMiddleware),
  };
};

export default createApiClient;
