import type { AuthContext } from "~/shared/contexts/auth";

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
      const { token: fetchToken } = await import("./index");
      const result = await fetchToken({});
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
