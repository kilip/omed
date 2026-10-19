import { queryOptions } from "@tanstack/react-query";
import { api, orForbidden, unwrapList, unwrapOne } from "~/lib/api";

export const memberKeys = {
  all: (wid: number) => ["core", "workspaces", wid, "members"] as const,
};

export const invitationKeys = {
  all: (wid: number) => ["core", "workspaces", wid, "invitations"] as const,
};

export const roleKeys = {
  all: (wid: number) => ["core", "workspaces", wid, "roles"] as const,
};

export const permissionKeys = {
  all: (wid: number) => ["core", "workspaces", wid, "permissions"] as const,
};

/** Current user permissions and roles in a workspace. */
export const permissionsQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: permissionKeys.all(wid),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/permissions", { params: { path: { wid } } }),
      ).catch(orForbidden({ roles: [], permissions: [], owner: false })),
  });

/** Roles that can be assigned in a workspace, with their display names. */
export const rolesQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: roleKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.GET("/workspaces/{wid}/roles", { params: { path: { wid } } }),
      ),
  });

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
