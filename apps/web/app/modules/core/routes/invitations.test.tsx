import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { getActiveWorkspace } from "~/lib/workspace";
import { invitationsQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/invitations";
import Invitations, { clientLoader } from "./invitations";

vi.mock("~/lib/workspace", () => ({
  getActiveWorkspace: vi.fn(),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
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
    useLocation: () => ({ pathname: "/core/invitations" }),
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({ run: vi.fn(), pending: false }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn().mockReturnValue({
      data: [
        {
          id: 1,
          token: "invite-token-abc",
          roles: ["core:member"],
          email: "invitee@example.com",
          usesCount: 0,
          maxUses: 1,
          createdAt: "2026-01-01T00:00:00Z",
          isRevoked: false,
        },
      ],
    }),
  };
});

vi.mock("../hooks/use-roles", () => ({
  useRoles: () => ({
    roles: [{ id: "core:member", name: "Member" }],
    options: [{ value: "core:member", label: "Member" }],
    label: (id: string) => (id === "core:member" ? "Member" : id),
  }),
}));

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Invitations route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("returns isPersonal: true for personal workspaces without prefetching", async () => {
      vi.mocked(getActiveWorkspace).mockResolvedValueOnce({
        id: 1,
        name: "Personal",
        personal: true,
      });

      const result = await clientLoader();
      expect(result).toEqual({ wid: 1, isPersonal: true });
      expect(getQueryClient).not.toHaveBeenCalled();
    });

    it("prefetches invitations and roles for non-personal workspaces", async () => {
      const mockEnsureQueryData = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsureQueryData,
      } as unknown as ReturnType<typeof getQueryClient>);

      vi.mocked(getActiveWorkspace).mockResolvedValueOnce({
        id: 10,
        name: "Acme Corp",
        personal: false,
      });

      const result = await clientLoader();
      expect(result).toEqual({ wid: 10, isPersonal: false });
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: invitationsQueryOptions(10).queryKey,
        }),
      );
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: rolesQueryOptions(10).queryKey,
        }),
      );
    });
  });

  describe("Invitations component", () => {
    it("renders personal workspace notice when isPersonal is true", () => {
      const html = renderToString(
        <Invitations
          loaderData={{ wid: 1, isPersonal: true }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("core.invitations.personalNotice");
      expect(html).toContain("ant-empty");
      expect(html).not.toContain("invitee@example.com");
    });

    it("renders invitations list content when isPersonal is false", () => {
      const html = renderToString(
        <Invitations
          loaderData={{ wid: 10, isPersonal: false }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("oken-abc");
      expect(html).toContain("Member");
      expect(html).toContain("0 / 1");
      expect(html).not.toContain("core.invitations.personalNotice");
    });
  });
});
