import type { components } from "@omed/openapi";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountsQueryOptions,
  entriesInfiniteQueryOptions,
  periodsQueryOptions,
  transactionsInfiniteQueryOptions,
} from "../queries";
import type { Route } from "./+types/journal";
import JournalRoute, {
  calculateEntryDebitTotal,
  clientLoader,
} from "./journal";

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
      t: (key: string, options?: Record<string, unknown>) => {
        if (options?.amount)
          return `${options.amount} ${options.currency ?? ""}`;
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
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: "/fin/journal" }),
  };
});

vi.mock("../hooks", () => ({
  useFinPermissions: () => ({
    canCreateEntry: true,
  }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn().mockReturnValue({ data: [] }),
    useInfiniteQuery: vi.fn().mockReturnValue({
      data: {
        pages: [
          {
            items: [
              {
                id: 1,
                number: "JE-2026-0001",
                date: "2026-01-15",
                description: "Monthly subscription",
                status: "POSTED",
                currency: "USD",
                postings: [
                  { direction: "DEBIT", amount: "120.50" },
                  { direction: "CREDIT", amount: "120.50" },
                ],
              },
            ],
          },
        ],
      },
      fetchNextPage: vi.fn(),
      hasNextPage: false,
      isFetchingNextPage: false,
    }),
  };
});

type Posting = components["schemas"]["model.PostingResponse"];
const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Journal route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("calculateEntryDebitTotal", () => {
    it("sums only DEBIT postings with string amounts", () => {
      const postings: Partial<Posting>[] = [
        { direction: "DEBIT", amount: "150.25" },
        { direction: "CREDIT", amount: "150.25" },
        { direction: "DEBIT", amount: "49.75" },
      ];

      expect(calculateEntryDebitTotal(postings as Posting[])).toBe(200);
    });

    it("handles numeric amount values and zero amounts", () => {
      const postings: Partial<Posting>[] = [
        { direction: "DEBIT", amount: "50" },
        { direction: "DEBIT", amount: "0" },
      ];

      expect(calculateEntryDebitTotal(postings as Posting[])).toBe(50);
    });

    it("returns 0 for undefined or empty postings", () => {
      expect(calculateEntryDebitTotal(undefined)).toBe(0);
      expect(calculateEntryDebitTotal([])).toBe(0);
    });
  });

  describe("clientLoader", () => {
    it("ensures entries, transactions, accounts, periods, and permissions queries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(88);
      const mockEnsureInfinite = vi.fn().mockResolvedValue([]);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureInfiniteQueryData: mockEnsureInfinite,
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();
      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsureInfinite).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: entriesInfiniteQueryOptions(88, {}).queryKey,
        }),
      );
      expect(mockEnsureInfinite).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: transactionsInfiniteQueryOptions(88, {}).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(88).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: periodsQueryOptions(88).queryKey }),
      );
      expect(result).toEqual({ wid: 88 });
    });
  });

  describe("JournalRoute component", () => {
    it("renders journal entries list and action buttons", () => {
      const html = renderToString(
        <JournalRoute
          loaderData={{ wid: 88 }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("JE-2026-0001");
      expect(html).toContain("Monthly subscription");
      expect(html).toContain("POSTED");
      expect(html).toContain("fin.journal.new");
    });
  });
});
