"use client";
import type { paths as financePaths } from "@omed/finance";
import createClient, { type Middleware } from "openapi-fetch";

export interface APIClient {
  finance: ReturnType<typeof createClient<financePaths>>;
}

export const createApiClient = (accessToken: string): APIClient => {
  const auth: Middleware = {
    async onRequest({ request }) {
      request.headers.set("Authorization", `Bearer ${accessToken}`);
    },
  };

  const finance = createClient<financePaths>({
    baseUrl: process.env.NEXT_PUBLIC_API_FINANCE_URL,
  });

  finance.use(auth);
  return {
    finance,
  };
};
