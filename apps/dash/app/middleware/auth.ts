import { redirect } from "react-router";
import { type AuthContext, authContext } from "~/contexts/auth";
import { getSession, token } from "~/lib/auth";
import i18n from "~/lib/i18n";
import type { Route } from "../+types/root";

let cache: AuthContext | null = null;

export function isFresh(token: string, skewMs = 30_000) {
  try {
    const payload = JSON.parse(
      atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")),
    );
    return payload.exp * 1000 - Date.now() > skewMs;
  } catch {
    return false;
  }
}

export const getCachedAuth = () =>
  cache && isFresh(cache.token) ? cache : null;
export const setCachedAuth = (v: AuthContext) => (cache = v);
export const clearAuthCache = () => (cache = null);
export const updateCachedAuthUser = (patch: Partial<AuthContext["user"]>) => {
  if (cache) cache = { ...cache, user: { ...cache.user, ...patch } };
};

let refreshPromise: Promise<string> | null = null;

export async function getValidAuthToken(
  currentToken?: string,
): Promise<string> {
  if (currentToken && isFresh(currentToken)) {
    return currentToken;
  }

  const cached = getCachedAuth();
  if (cached && isFresh(cached.token)) {
    return cached.token;
  }

  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const result = await token();
      const newToken = result.data?.token;
      if (newToken) {
        if (cache) {
          cache = { ...cache, token: newToken };
        }
        return newToken;
      }
    } catch (e) {
      console.error("Failed to refresh token:", e);
    } finally {
      refreshPromise = null;
    }
    return currentToken ?? cache?.token ?? "";
  })();

  return refreshPromise;
}

export const authMiddleware: Route.ClientMiddlewareFunction = async (
  { context },
  next,
) => {
  const cached = getCachedAuth();
  if (cached) {
    context.set(authContext, cached);
    if (
      cached.user.locale &&
      (cached.user.locale === "en" || cached.user.locale === "id") &&
      i18n.language !== cached.user.locale
    ) {
      await i18n.changeLanguage(cached.user.locale);
    }
    return next();
  }

  const { data, error } = await getSession();
  if (data === null || error) throw redirect("/login");

  const result = await token();
  const value: AuthContext = {
    user: data.user,
    token: result.data?.token ?? "",
    session: data.session,
    loading: false,
  };
  setCachedAuth(value);
  context.set(authContext, value);

  if (
    data.user.locale &&
    (data.user.locale === "en" || data.user.locale === "id") &&
    i18n.language !== data.user.locale
  ) {
    await i18n.changeLanguage(data.user.locale);
  }

  return next();
};

export const clientMiddleware: Route.ClientMiddlewareFunction[] = [
  authMiddleware,
];
