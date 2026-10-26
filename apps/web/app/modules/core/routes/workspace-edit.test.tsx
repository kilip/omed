import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import type { Route } from "./+types/workspace-edit";
import WorkspaceEdit, { clientLoader } from "./workspace-edit";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      PUT: vi.fn(),
      DELETE: vi.fn(),
    },
  };
});

vi.mock("~/lib/workspace", () => ({
  getActiveWorkspace: vi.fn(),
  getStoredWorkspaceId: vi.fn(),
  refreshWorkspaces: vi.fn(),
  setActiveWorkspaceId: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: "/core/workspaces/1/edit" }),
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({ run: vi.fn(), pending: false }),
}));

type ApiGetResult = Awaited<ReturnType<typeof api.GET>>;
const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("WorkspaceEdit route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("loads workspace and permissions by wid from params", async () => {
      const mockWorkspace = {
        id: 15,
        name: "Acme HQ",
        description: "Headquarters",
        personal: false,
      };
      const mockPermissions = {
        roles: ["admin"],
        permissions: ["all"],
        owner: true,
      };

      vi.mocked(api.GET).mockImplementation((path: string) => {
        if (path === "/workspaces/{wid}") {
          return Promise.resolve({
            data: { data: mockWorkspace },
            response: new Response(null, { status: 200 }),
          } as unknown as ApiGetResult);
        }
        if (path === "/workspaces/{wid}/permissions") {
          return Promise.resolve({
            data: { data: mockPermissions },
            response: new Response(null, { status: 200 }),
          } as unknown as ApiGetResult);
        }
        return Promise.reject(new Error("Unknown route"));
      });

      const result = await clientLoader({
        params: { id: "15" },
        request: new Request("http://localhost"),
      } as unknown as Route.ClientLoaderArgs);

      expect(api.GET).toHaveBeenCalledWith("/workspaces/{wid}", {
        params: { path: { wid: 15 } },
      });
      expect(api.GET).toHaveBeenCalledWith("/workspaces/{wid}/permissions", {
        params: { path: { wid: 15 } },
      });
      expect(result).toEqual({
        workspace: mockWorkspace,
        permissions: mockPermissions,
        wid: 15,
      });
    });

    it("falls back to default permissions on 403 Forbidden", async () => {
      const mockWorkspace = {
        id: 15,
        name: "Acme HQ",
        personal: false,
      };

      vi.mocked(api.GET).mockImplementation((path: string) => {
        if (path === "/workspaces/{wid}") {
          return Promise.resolve({
            data: { data: mockWorkspace },
            response: new Response(null, { status: 200 }),
          } as unknown as ApiGetResult);
        }
        if (path === "/workspaces/{wid}/permissions") {
          return Promise.resolve({
            error: { error: { code: "FORBIDDEN", message: "Forbidden" } },
            response: new Response(null, { status: 403 }),
          } as unknown as ApiGetResult);
        }
        return Promise.reject(new Error("Unknown route"));
      });

      const result = await clientLoader({
        params: { id: "15" },
        request: new Request("http://localhost"),
      } as unknown as Route.ClientLoaderArgs);

      expect(result.permissions).toEqual({
        roles: [],
        permissions: [],
        owner: false,
      });
    });
  });

  describe("WorkspaceEdit component", () => {
    it("renders error view when workspace is null", () => {
      const html = renderToString(
        <WorkspaceEdit
          loaderData={{
            workspace:
              null as unknown as Route.ComponentProps["loaderData"]["workspace"],
            permissions: { roles: [], permissions: [], owner: false },
            wid: 15,
          }}
          params={{ id: "15" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("common.error");
      expect(html).toContain("common.cancel");
    });

    it("renders form and delete button for non-personal owned workspace", () => {
      const html = renderToString(
        <WorkspaceEdit
          loaderData={{
            workspace: {
              id: 15,
              name: "Acme HQ",
              description: "Main workspace",
              personal: false,
            },
            permissions: { roles: ["admin"], permissions: [], owner: true },
            wid: 15,
          }}
          params={{ id: "15" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("Acme HQ");
      expect(html).toContain("Main workspace");
      expect(html).toContain("common.delete");
    });

    it("does not render delete button for personal workspace", () => {
      const html = renderToString(
        <WorkspaceEdit
          loaderData={{
            workspace: {
              id: 1,
              name: "Personal",
              personal: true,
            },
            permissions: { roles: [], permissions: [], owner: true },
            wid: 1,
          }}
          params={{ id: "1" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("Personal");
      expect(html).not.toContain("core.workspaces.delete");
    });
  });
});
