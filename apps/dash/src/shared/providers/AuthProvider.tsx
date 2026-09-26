"use client";

import type { User } from "@omed/auth";
import { getSession } from "@omed/auth/client";
import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { type APIClient, createApiClient } from "./api";

export interface AuthContextValue {
  token: string | undefined;
  user: User | undefined;
  api: APIClient;
  isLoading: boolean;
}

const REFRESH_MARGIN_SEC = 30;

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function decodeExp(jwt: string): number {
  const payload = JSON.parse(atob(jwt.split(".")[1]));
  return payload.exp as number;
}

export function AuthProvider({ children }: PropsWithChildren) {
  const [token, setToken] = useState<string>();
  const [user, setUser] = useState<User>();
  const [api, setApi] = useState<APIClient>(createApiClient("init"));
  const [isLoading, setIsLoading] = useState(true);
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;

    const fetchSession = async () => {
      const { data } = await getSession({
        fetchOptions: {
          onSuccess(ctx) {
            const newToken =
              ctx.response.headers.get("set-auth-jwt") ?? undefined;
            if (cancelled) return;
            setToken(newToken);

            if (newToken) {
              const exp = decodeExp(newToken);
              const delayMs = Math.max(
                (exp - Math.floor(Date.now() / 1000) - REFRESH_MARGIN_SEC) *
                  1000,
                0,
              );
              timerRef.current = setTimeout(fetchSession, delayMs);
            }
          },
        },
      });
      if (cancelled) return;
      setUser(data?.user);
      setIsLoading(false);
    };

    fetchSession();

    return () => {
      cancelled = true;
      clearTimeout(timerRef.current);
    };
  }, []);

  useEffect(() => {
    if (token) {
      setApi(createApiClient(token));
    }
  }, [token]);

  return (
    <AuthContext.Provider value={{ token, user, api, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
