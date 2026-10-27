import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { authClient } from "~/lib/auth-client";
import { refreshWorkspaces, requireWorkspaceId } from "~/lib/workspace";
import Shell, { clientLoader } from "./shell";

const mockRevalidate = vi.fn();

vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
  useRevalidator: () => ({ revalidate: mockRevalidate, state: "idle" }),
  Outlet: () => <div data-testid="outlet">Outlet Content</div>,
  redirect: vi.fn((path: string) => {
    const error = new Error(`Redirect to ${path}`);
    (error as unknown as { status: number; location: string }).status = 302;
    (error as unknown as { status: number; location: string }).location = path;
    throw error;
  }),
}));

vi.mock("~/components/NavBar", () => ({
  NavBar: () => <div data-testid="navbar">NavBar</div>,
}));

vi.mock("~/lib/auth-client", () => ({
  authClient: {
    getSession: vi.fn(),
    signOut: vi.fn(),
  },
}));

vi.mock("~/lib/workspace", () => ({
  refreshWorkspaces: vi.fn(),
  requireWorkspaceId: vi.fn(),
  setActiveWorkspaceId: vi.fn(),
}));

describe("Shell clientLoader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("redirects to /login when session is absent", async () => {
    vi.mocked(authClient.getSession).mockResolvedValueOnce({
      data: null,
      error: null,
    });

    await expect(clientLoader()).rejects.toThrow("Redirect to /login");
  });

  it("returns user, workspaces and activeId on success", async () => {
    const mockUser = {
      id: 1,
      name: "Toni",
      email: "toni@example.com",
    };
    const mockWorkspaces = [{ id: 1, name: "Workspace 1", personal: true }];

    vi.mocked(authClient.getSession).mockResolvedValueOnce({
      data: { user: mockUser, session: { id: 1 } },
      error: null,
    });
    vi.mocked(refreshWorkspaces).mockResolvedValueOnce(mockWorkspaces);
    vi.mocked(requireWorkspaceId).mockResolvedValueOnce(1);

    const result = await clientLoader();
    expect(result).toEqual({
      user: mockUser,
      workspaces: mockWorkspaces,
      activeId: 1,
      apiError: undefined,
    });
  });

  it("handles empty workspaces without apiError when user has no workspace yet", async () => {
    const mockUser = {
      id: 1,
      name: "Toni",
      email: "toni@example.com",
    };

    vi.mocked(authClient.getSession).mockResolvedValueOnce({
      data: { user: mockUser, session: { id: 1 } },
      error: null,
    });
    vi.mocked(refreshWorkspaces).mockResolvedValueOnce([]);

    const result = await clientLoader();
    expect(result).toEqual({
      user: mockUser,
      workspaces: [],
      activeId: undefined,
      apiError: undefined,
    });
    expect(requireWorkspaceId).not.toHaveBeenCalled();
  });

  it("catches errors and returns apiError when API fails", async () => {
    const mockUser = {
      id: 1,
      name: "Toni",
      email: "toni@example.com",
    };

    vi.mocked(authClient.getSession).mockResolvedValueOnce({
      data: { user: mockUser, session: { id: 1 } },
      error: null,
    });
    vi.mocked(refreshWorkspaces).mockRejectedValueOnce(
      new Error("Network connection refused"),
    );

    const result = await clientLoader();
    expect(result).toEqual({
      user: mockUser,
      workspaces: [],
      activeId: undefined,
      apiError: "Network connection refused",
    });
  });
});

describe("Shell component", () => {
  const dummyUser = {
    id: 1,
    name: "Toni",
    email: "toni@example.com",
    locale: "en",
  };

  it("renders without apiError (no error banner, outlet visible)", () => {
    const html = renderToString(
      <Shell
        loaderData={{
          user: dummyUser,
          workspaces: [{ id: 1, name: "Workspace 1", personal: true }],
          activeId: 1,
          apiError: undefined,
        }}
        params={{}}
        matches={[]}
      />,
    );

    expect(html).toContain("NavBar");
    expect(html).toContain("Outlet Content");
    expect(html).not.toContain("ant-alert");
    expect(html).not.toContain("ant-result");
  });

  it("renders error banner and retry button when apiError is present with workspaces", () => {
    const html = renderToString(
      <Shell
        loaderData={{
          user: dummyUser,
          workspaces: [{ id: 1, name: "Workspace 1", personal: true }],
          activeId: 1,
          apiError: "Failed to sync recent data",
        }}
        params={{}}
        matches={[]}
      />,
    );

    expect(html).toContain("NavBar");
    expect(html).toContain("Failed to sync recent data");
    expect(html).toContain("ant-alert");
    expect(html).toContain("Outlet Content");
  });

  it("renders prominent error result when apiError is present and workspaces could not load", () => {
    const html = renderToString(
      <Shell
        loaderData={{
          user: dummyUser,
          workspaces: [],
          activeId: undefined,
          apiError: "API service unavailable",
        }}
        params={{}}
        matches={[]}
      />,
    );

    expect(html).toContain("NavBar");
    expect(html).toContain("API service unavailable");
    expect(html).toContain("ant-alert");
    expect(html).toContain("ant-result");
    expect(html).not.toContain("Outlet Content");
  });
});
