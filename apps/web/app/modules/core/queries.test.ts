import type { QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import {
  invitationKeys,
  invitationsQueryOptions,
  memberKeys,
  membersQueryOptions,
  permissionKeys,
  permissionsQueryOptions,
  roleKeys,
  rolesQueryOptions,
} from "./queries";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
    },
  };
});

type ApiGetResult = Awaited<ReturnType<typeof api.GET>>;
type ApiPostResult = Awaited<ReturnType<typeof api.POST>>;
const mockQueryContext = {} as unknown as QueryFunctionContext;

describe("core queries", () => {
  const wid = 42;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("query keys", () => {
    it("generates correct memberKeys", () => {
      expect(memberKeys.all(wid)).toEqual([
        "core",
        "workspaces",
        wid,
        "members",
      ]);
    });

    it("generates correct invitationKeys", () => {
      expect(invitationKeys.all(wid)).toEqual([
        "core",
        "workspaces",
        wid,
        "invitations",
      ]);
    });

    it("generates correct roleKeys", () => {
      expect(roleKeys.all(wid)).toEqual(["core", "workspaces", wid, "roles"]);
    });

    it("generates correct permissionKeys", () => {
      expect(permissionKeys.all(wid)).toEqual([
        "core",
        "workspaces",
        wid,
        "permissions",
      ]);
    });
  });

  describe("permissionsQueryOptions", () => {
    it("fetches permissions on success", async () => {
      const opts = permissionsQueryOptions(wid);
      expect(opts.queryKey).toEqual(["core", "workspaces", wid, "permissions"]);

      const mockPerms = {
        roles: ["admin"],
        permissions: ["workspace:manage"],
        owner: true,
      };

      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: mockPerms },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      const result = await opts.queryFn?.(mockQueryContext);
      expect(api.GET).toHaveBeenCalledWith("/workspaces/{wid}/permissions", {
        params: { path: { wid } },
      });
      expect(result).toEqual(mockPerms);
    });

    it("returns default fallback when 403 Forbidden is received", async () => {
      const opts = permissionsQueryOptions(wid);

      vi.mocked(api.GET).mockResolvedValueOnce({
        error: { error: { code: "FORBIDDEN", message: "Forbidden" } },
        response: new Response(null, { status: 403 }),
      } as unknown as ApiGetResult);

      const result = await opts.queryFn?.(mockQueryContext);
      expect(result).toEqual({ roles: [], permissions: [], owner: false });
    });
  });

  describe("rolesQueryOptions", () => {
    it("fetches roles list", async () => {
      const opts = rolesQueryOptions(wid);
      expect(opts.queryKey).toEqual(["core", "workspaces", wid, "roles"]);

      const mockRoles = [
        { id: "admin", name: "Admin", description: "Admin role" },
        { id: "member", name: "Member", description: "Member role" },
      ];

      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: mockRoles },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      const result = await opts.queryFn?.(mockQueryContext);
      expect(api.GET).toHaveBeenCalledWith("/workspaces/{wid}/roles", {
        params: { path: { wid } },
      });
      expect(result).toEqual(mockRoles);
    });
  });

  describe("membersQueryOptions", () => {
    it("searches members with limit 100", async () => {
      const opts = membersQueryOptions(wid);
      expect(opts.queryKey).toEqual(["core", "workspaces", wid, "members"]);

      const mockMembers = [{ id: 1, userId: 10, roles: ["admin"] }];

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: mockMembers },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPostResult);

      const result = await opts.queryFn?.(mockQueryContext);
      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/members/search",
        {
          params: { path: { wid } },
          body: { limit: 100 },
        },
      );
      expect(result).toEqual(mockMembers);
    });
  });

  describe("invitationsQueryOptions", () => {
    it("searches invitations with limit 100", async () => {
      const opts = invitationsQueryOptions(wid);
      expect(opts.queryKey).toEqual(["core", "workspaces", wid, "invitations"]);

      const mockInvitations = [
        { id: 1, token: "tok1", workspaceId: wid, roles: ["member"] },
      ];

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: mockInvitations },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPostResult);

      const result = await opts.queryFn?.(mockQueryContext);
      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/invitations/search",
        {
          params: { path: { wid } },
          body: { limit: 100 },
        },
      );
      expect(result).toEqual(mockInvitations);
    });
  });
});
