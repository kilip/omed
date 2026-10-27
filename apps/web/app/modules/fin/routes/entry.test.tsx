import type { FetchQueryOptions } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountsQueryOptions,
  entryQueryOptions,
  periodsQueryOptions,
} from "../queries";
import type { Route } from "./+types/entry";
import EntryRoute, { clientLoader } from "./entry";

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
    useLocation: () => ({ pathname: "/fin/entries/10" }),
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({ run: vi.fn(), pending: false }),
}));

vi.mock("../hooks", () => ({
  useFinPermissions: () => ({
    canEditEntry: true,
    canDeleteEntry: true,
    canPostEntry: true,
    canVoidEntry: true,
    canReverseEntry: true,
  }),
}));

const mockEntryData = {
  id: 10,
  number: "JE-2026-0099",
  date: "2026-02-15",
  description: "Server hardware purchase",
  status: "DRAFT",
  periodId: 2,
  transactionId: 5,
  postings: [
    {
      id: 1,
      accountId: 101,
      direction: "DEBIT",
      amount: "15000000",
      baseAmount: "15000000",
      quantity: "3",
      unitPrice: "5000000",
      description: "Hardware asset",
    },
    {
      id: 2,
      accountId: 102,
      direction: "CREDIT",
      amount: "15000000",
      baseAmount: "15000000",
      description: "Bank transfer",
    },
  ],
};

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQuery: vi.fn().mockReturnValue({
      data: {
        id: 5,
        currency: "IDR",
        exchangeRate: 1,
      },
    }),
    useSuspenseQuery: vi.fn().mockImplementation((opts: FetchQueryOptions) => {
      const key = opts.queryKey as unknown[];
      if (key?.includes("entries")) {
        return { data: mockEntryData };
      }
      if (key?.includes("accounts")) {
        return {
          data: [
            { id: 101, code: "1500", name: "Equipment" },
            { id: 102, code: "1100", name: "Main Bank" },
          ],
        };
      }
      if (key?.includes("periods")) {
        return {
          data: [{ id: 2, name: "February 2026", status: "OPEN" }],
        };
      }
      return { data: [] };
    }),
  };
});

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Entry route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("ensures entry, accounts, periods, and permissions query data", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(99);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader({
        params: { id: "10" },
        request: new Request("http://localhost"),
      } as unknown as Route.ClientLoaderArgs);

      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: entryQueryOptions(99, 10).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(99).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: periodsQueryOptions(99).queryKey }),
      );
      expect(result).toEqual({ wid: 99, id: 10 });
    });
  });

  describe("EntryRoute component", () => {
    it("renders entry details, postings and action buttons for DRAFT status", () => {
      const html = renderToString(
        <EntryRoute
          loaderData={{ wid: 99, id: 10 }}
          params={{ id: "10" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("JE-2026-0099");
      expect(html).toContain("Server hardware purchase");
      expect(html).toContain("February 2026");
      expect(html).toContain("fin.journal.post");
      expect(html).toContain("fin.journal.editEntry");
      expect(html).toContain("common.delete");
    });

    it("renders reverse action button for POSTED status", () => {
      const html = renderToString(
        <EntryRoute
          loaderData={{ wid: 99, id: 10 }}
          params={{ id: "10" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("fin.journal.post");
    });
  });
});
