import type { FetchQueryOptions } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  periodsQueryOptions,
  transactionsInfiniteQueryOptions,
} from "../queries";
import type { Route } from "./+types/entry-new";
import EntryNewRoute, { clientLoader } from "./entry-new";

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
    useLocation: () => ({ pathname: "/fin/entries/new" }),
    useSearchParams: () => [new URLSearchParams(), vi.fn()],
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({ run: vi.fn(), pending: false }),
}));

vi.mock("../hooks", () => ({
  useFinPermissions: () => ({
    canCreateEntry: true,
  }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
    }),
    useSuspenseQuery: vi.fn().mockImplementation((opts: FetchQueryOptions) => {
      const key = opts.queryKey as unknown[];
      if (key?.includes("accounts")) {
        return {
          data: [
            { id: 1, code: "1000", name: "Cash" },
            { id: 2, code: "2000", name: "Accounts Payable" },
          ],
        };
      }
      if (key?.includes("currencies")) {
        return {
          data: [
            { code: "IDR", name: "Indonesian Rupiah", isBase: true },
            { code: "USD", name: "US Dollar", isBase: false },
          ],
        };
      }
      if (key?.includes("periods")) {
        return {
          data: [{ id: 1, name: "January 2026", status: "OPEN" }],
        };
      }
      return { data: [] };
    }),
    useSuspenseInfiniteQuery: vi.fn().mockReturnValue({
      data: {
        pages: [{ items: [] }],
      },
    }),
  };
});

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("EntryNew route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("ensures accounts, currencies, periods, permissions, and transactions queries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(77);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      const mockEnsureInfinite = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
        ensureInfiniteQueryData: mockEnsureInfinite,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();
      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(77).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: currenciesQueryOptions(77).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: periodsQueryOptions(77).queryKey }),
      );
      expect(mockEnsureInfinite).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: transactionsInfiniteQueryOptions(77, {}).queryKey,
        }),
      );
      expect(result).toEqual({ wid: 77 });
    });
  });

  describe("EntryNewRoute component", () => {
    it("renders create entry form with required inputs", () => {
      const html = renderToString(
        <EntryNewRoute
          loaderData={{ wid: 77 }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("fin.journal.new");
      expect(html).toContain("fin.journal.addLine");
      expect(html).toContain("common.save");
    });
  });
});
