import type { financePaths } from "@omed/api";
import createClient from "openapi-fetch";
import { useMemo } from "react";
import { getValidAuthToken } from "~/shared/auth/cache";
import { useAuth } from "../providers/AuthProvider";

export function useApi() {
  const { token } = useAuth();

  const finance = useMemo(() => {
    const client = createClient<financePaths>();

    client.use({
      async onRequest({ request }) {
        const validToken = await getValidAuthToken(token);
        if (validToken) {
          request.headers.set("Authorization", `Bearer ${validToken}`);
        }
        return request;
      },
    });

    return client;
  }, [token]);

  return {
    finance,
  };
}
