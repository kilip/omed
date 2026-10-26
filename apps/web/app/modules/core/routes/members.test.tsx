import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { getActiveWorkspace } from "~/lib/workspace";
import { membersQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/members";
import Members, { clientLoader } from "./members";

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
    useLocation: () => ({ pathname: "/core/members" }),
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
          userId: 101,
          roles: ["core:admin"],
          user: { id: 101, name: "Alice", email: "alice@example.com" },
          createdAt: "2026-01-01T00:00:00Z",
        },
      ],
    }),
  };
});

vi.mock("../hooks/use-roles", () => ({
  useRoles: () => ({
    roles: [{ id: "core:admin", name: "Admin" }],
    options: [{ value: "core:admin", label: "Admin" }],
    label: (id: string) => (id === "core:admin" ? "Admin" : id),
  }),
}));

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Members route", () => {
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

    it("prefetches members and roles for non-personal workspaces", async () => {
      const mockEnsureQueryData = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsureQueryData,
      } as unknown as ReturnType<typeof getQueryClient>);

      vi.mocked(getActiveWorkspace).mockResolvedValueOnce({
        id: 42,
        name: "Acme Corp",
        personal: false,
      });

      const result = await clientLoader();
      expect(result).toEqual({ wid: 42, isPersonal: false });
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: membersQueryOptions(42).queryKey }),
      );
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: rolesQueryOptions(42).queryKey }),
      );
    });
  });

  describe("Members component", () => {
    it("renders personal workspace notice when isPersonal is true", () => {
      const html = renderToString(
        <Members
          loaderData={{ wid: 1, isPersonal: true }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("core.members.personalNotice");
      expect(html).toContain("ant-empty");
      expect(html).not.toContain("Alice");
    });

    it("renders members table content when isPersonal is false", () => {
      const html = renderToString(
        <Members
          loaderData={{ wid: 42, isPersonal: false }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("Alice");
      expect(html).toContain("alice@example.com");
      expect(html).not.toContain("core.members.personalNotice");
    });
  });
});
