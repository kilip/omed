import { queryOptions } from "@tanstack/react-query";
import { api, unwrapList } from "~/lib/api";

export const memberKeys = {
  all: (wid: number) => ["sys", "workspaces", wid, "members"] as const,
};

export const invitationKeys = {
  all: (wid: number) => ["sys", "workspaces", wid, "invitations"] as const,
};

export const workspaceKeys = {
  list: (name?: string) => ["sys", "workspace-list", { name }] as const,
};

/** Members of a workspace (first 100, matching the previous loader). */
export const membersQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: memberKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/{wid}/members/search", {
          params: { path: { wid } },
          body: { limit: 100 },
        }),
      ),
  });

/** Invitations of a workspace (first 100). */
export const invitationsQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: invitationKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/{wid}/invitations/search", {
          params: { path: { wid } },
          body: { limit: 100 },
        }),
      ),
  });

/** Workspaces visible to the caller, optionally filtered by name. */
export const workspacesQueryOptions = (name?: string) =>
  queryOptions({
    queryKey: workspaceKeys.list(name),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/search", { body: { name, limit: 100 } }),
      ),
  });
