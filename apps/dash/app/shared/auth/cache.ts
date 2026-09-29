import type { AuthContext } from "~/shared/contexts/auth";

let cache: AuthContext | null = null;

function isFresh(token: string, skewMs = 30_000) {
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
