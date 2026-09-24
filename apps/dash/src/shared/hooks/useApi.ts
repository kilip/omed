import { User } from "@omed/auth";
import { getSession } from "@omed/auth/client";
import { useEffect, useState } from "react";
import createApiClient from "@omed/api";

type APIClient = ReturnType<typeof createApiClient>;

export function useApi() {
  const [token, setToken] = useState<string>();
  const [user, setUser] = useState<User>();
  const [api, setApi] = useState<APIClient>();

  useEffect(() => {
    const fetchSession = async () => {
      const { data } = await getSession({
        fetchOptions: {
          onSuccess(ctx) {
            setToken(ctx.response.headers.get("set-auth-jwt") ?? undefined);
          },
        },
      });
      setUser(data?.user);
    };
    fetchSession();
  }, []);

  useEffect(() => {
    if (token) {
      setApi(createApiClient(token));
    }
  }, [token]);

  return {
    token,
    user,
    api,
  };
}
