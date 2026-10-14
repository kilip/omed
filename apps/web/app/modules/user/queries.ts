import { queryOptions } from "@tanstack/react-query";
import { api, unwrapList } from "~/lib/api";

export const userKeys = {
  list: (name?: string) => ["user", "list", { name }] as const,
};

/** Users matching an optional name filter (first 100). */
export const usersQueryOptions = (name?: string) =>
  queryOptions({
    queryKey: userKeys.list(name),
    queryFn: () =>
      unwrapList(api.POST("/users/search", { body: { name, limit: 100 } })),
  });
