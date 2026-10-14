import { queryOptions } from "@tanstack/react-query";
import { api, unwrapList } from "~/lib/api";

export const accountKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "accounts"] as const,
};

/** Full chart of accounts for a workspace (filtering and tree-building happen client-side). */
export const accountsQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: accountKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/{wid}/fin/accounts/search", {
          params: { path: { wid } },
          body: {},
        }),
      ),
  });
