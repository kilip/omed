import type { paths as financePaths } from "@omed/openapi/finance";
import createClient from "openapi-fetch";
import { useMemo } from "react";
import { appEnv } from "~/lib/appEnv";
import { getCachedAuth, getValidAuthToken } from "~/middleware/auth";
import { useOptionalAuth } from "~/shared/providers/AuthProvider";

export function useApi() {
  const auth = useOptionalAuth();
  const token = auth?.token ?? getCachedAuth()?.token ?? "";

  const finance = useMemo(() => {
    const client = createClient<financePaths>({
      baseUrl: appEnv.VITE_FINANCE_URL,
    });

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
