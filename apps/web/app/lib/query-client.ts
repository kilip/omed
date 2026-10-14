import { isServer, QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api";

const STALE_TIME_MS = 30 * 1000;
const MAX_RETRIES = 2;

function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: STALE_TIME_MS,
        refetchOnWindowFocus: false,
        // Client errors (401/403/404...) will not succeed on retry.
        retry: (failureCount, error) => {
          if (error instanceof ApiError && error.status < 500) return false;
          return failureCount < MAX_RETRIES;
        },
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

/**
 * Returns a QueryClient. On the server a fresh client is created per call so
 * cache never leaks between requests; in the browser a singleton is reused.
 */
export function getQueryClient(): QueryClient {
  if (isServer) return makeQueryClient();
  browserQueryClient ??= makeQueryClient();
  return browserQueryClient;
}
