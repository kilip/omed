import createClient, { type Middleware } from "openapi-fetch";
import type { paths as financePaths } from "./schema/finance";

interface ApiClientOptions {
  accessToken: string;
  financeUrl: string;
}

export const createApiClient = (options: ApiClientOptions) => {
  const authMiddleware: Middleware = {
    async onRequest({ request }) {
      request.headers.set("Authorization", `Bearer ${options.accessToken}`);
    },
  };
  return {
    finance: createClient<financePaths>({
      baseUrl: options.financeUrl,
    }).use(authMiddleware),
  };
};

export default createApiClient;
