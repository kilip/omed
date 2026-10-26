import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import { authClient } from "~/lib/auth-client";
import type { Route } from "./+types/invite";
import Invite, { clientLoader } from "./invite";

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

vi.mock("~/lib/auth-client", () => ({
  authClient: {
    getSession: vi.fn(),
    signIn: {
      social: vi.fn(),
    },
  },
}));

vi.mock("~/lib/workspace", () => ({
  setActiveWorkspaceId: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, options?: Record<string, unknown>) => {
        if (options?.workspace) return `${key}:${options.workspace}`;
        if (options?.name) return `${key}:${options.name}`;
        return key;
      },
      i18n: { language: "en" },
    }),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  };
});

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return {
    ...actual,
    App: {
      ...actual.App,
      useApp: () => ({
        message: {
          success: vi.fn(),
          error: vi.fn(),
        },
      }),
    },
  };
});

type ApiGetResult = Awaited<ReturnType<typeof api.GET>>;
type SessionResult = Awaited<ReturnType<typeof authClient.getSession>>;
const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Invite route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("loads invitation preview and session state", async () => {
      const mockPreview = {
        workspaceId: 10,
        workspaceName: "Omega Corp",
        inviterName: "Bob",
        roles: ["core:member"],
        isValid: true,
      };

      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: mockPreview },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      vi.mocked(authClient.getSession).mockResolvedValueOnce({
        data: { user: { id: 1 }, session: { id: 2 } },
        error: null,
      } as unknown as SessionResult);

      const result = await clientLoader({
        params: { token: "secret-token" },
        request: new Request("http://localhost"),
      } as unknown as Route.ClientLoaderArgs);

      expect(api.GET).toHaveBeenCalledWith("/invitations/{token}", {
        params: { path: { token: "secret-token" } },
      });
      expect(result).toEqual({
        preview: mockPreview,
        signedIn: true,
      });
    });

    it("identifies unauthenticated visitors", async () => {
      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: { workspaceName: "Public Team", isValid: true } },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      vi.mocked(authClient.getSession).mockResolvedValueOnce({
        data: null,
        error: null,
      });

      const result = await clientLoader({
        params: { token: "tok" },
        request: new Request("http://localhost"),
      } as unknown as Route.ClientLoaderArgs);

      expect(result.signedIn).toBe(false);
    });
  });

  describe("Invite component", () => {
    it("renders accept button when signed in with valid invitation", () => {
      const html = renderToString(
        <Invite
          loaderData={{
            preview: {
              workspaceId: 1,
              workspaceName: "Acme",
              inviterName: "Bob",
              roles: ["member"],
              isValid: true,
            },
            signedIn: true,
          }}
          params={{ token: "tok-123" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("invite.title:Acme");
      expect(html).toContain("invite.invitedBy:Bob");
      expect(html).toContain("invite.accept");
      expect(html).not.toContain("invite.signInFirst");
    });

    it("renders sign in buttons when not signed in with valid invitation", () => {
      const html = renderToString(
        <Invite
          loaderData={{
            preview: {
              workspaceId: 1,
              workspaceName: "Acme",
              roles: [],
              isValid: true,
            },
            signedIn: false,
          }}
          params={{ token: "tok-123" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("invite.title:Acme");
      expect(html).toContain("invite.signInFirst");
      expect(html).toContain("login.google");
      expect(html).toContain("login.github");
      expect(html).not.toContain("invite.accept");
    });

    it("renders invalid invitation warning when isValid is false", () => {
      const html = renderToString(
        <Invite
          loaderData={{
            preview: {
              workspaceId: 1,
              workspaceName: "Acme",
              roles: [],
              isValid: false,
            },
            signedIn: true,
          }}
          params={{ token: "tok-123" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("invite.invalid");
      expect(html).not.toContain("invite.accept");
    });
  });
});
