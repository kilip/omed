import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { accountsQueryOptions } from "../queries";
import type { Route } from "./+types/accounts";
import Accounts, { buildTree, clientLoader, defaultNormal } from "./accounts";

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn(),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => key,
      i18n: { language: "en" },
    }),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: "/fin/accounts" }),
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
          code: "1000",
          name: "Assets",
          type: "ASSET",
          normalBalance: "DEBIT",
          isReconciled: true,
        },
        {
          id: 2,
          code: "1100",
          name: "Cash",
          type: "ASSET",
          normalBalance: "DEBIT",
          parentId: 1,
        },
      ],
    }),
  };
});

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Accounts route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("defaultNormal", () => {
    it("returns DEBIT for ASSET and EXPENSE", () => {
      expect(defaultNormal("ASSET")).toBe("DEBIT");
      expect(defaultNormal("EXPENSE")).toBe("DEBIT");
    });

    it("returns CREDIT for LIABILITY, EQUITY, and REVENUE", () => {
      expect(defaultNormal("LIABILITY")).toBe("CREDIT");
      expect(defaultNormal("EQUITY")).toBe("CREDIT");
      expect(defaultNormal("REVENUE")).toBe("CREDIT");
    });
  });

  describe("buildTree", () => {
    it("returns empty array for empty input", () => {
      expect(buildTree([])).toEqual([]);
    });

    it("builds and sorts parent-child relationships", () => {
      const flatAccounts = [
        {
          id: 3,
          code: "2000",
          name: "Liabilities",
          type: "LIABILITY" as const,
        },
        {
          id: 2,
          code: "1100",
          name: "Bank",
          parentId: 1,
          type: "ASSET" as const,
        },
        { id: 1, code: "1000", name: "Current Assets", type: "ASSET" as const },
        {
          id: 4,
          code: "1050",
          name: "Petty Cash",
          parentId: 1,
          type: "ASSET" as const,
        },
      ];

      const tree = buildTree(flatAccounts);

      // Roots should be 1000 and 2000 in code order
      expect(tree).toHaveLength(2);
      expect(tree[0].code).toBe("1000");
      expect(tree[1].code).toBe("2000");

      // Children of 1000 should be sorted by code: 1050 then 1100
      expect(tree[0].children).toBeDefined();
      expect(tree[0].children?.[0].code).toBe("1050");
      expect(tree[0].children?.[1].code).toBe("1100");
    });

    it("handles orphaned accounts as roots if parentId is missing from map", () => {
      const accounts = [
        {
          id: 10,
          code: "9999",
          name: "Unknown Parent Child",
          parentId: 999,
          type: "ASSET" as const,
        },
      ];

      const tree = buildTree(accounts);
      expect(tree).toHaveLength(1);
      expect(tree[0].code).toBe("9999");
    });
  });

  describe("clientLoader", () => {
    it("ensures accountsQueryOptions and returns wid", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(33);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();
      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(33).queryKey,
        }),
      );
      expect(result).toEqual({ wid: 33 });
    });
  });

  describe("Accounts component", () => {
    it("renders account tree and action buttons", () => {
      const html = renderToString(
        <Accounts
          loaderData={{ wid: 33 }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("1000");
      expect(html).toContain("Assets");
      expect(html).toContain("1100");
      expect(html).toContain("Cash");
      expect(html).toContain("fin.accounts.new");
    });
  });
});
