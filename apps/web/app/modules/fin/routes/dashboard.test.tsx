import type { FetchQueryOptions } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  financialSummaryQueryOptions,
  recentEntriesQueryOptions,
} from "../queries";
import type { Route } from "./+types/dashboard";
import FinDashboard, { clientLoader } from "./dashboard";

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
    useLocation: () => ({ pathname: "/fin/overview" }),
  };
});

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn().mockImplementation((opts: FetchQueryOptions) => {
      const key = opts.queryKey as unknown[];
      if (key?.includes("summary")) {
        return {
          data: {
            baseCurrency: "IDR",
            totals: {
              assets: "5000000",
              liabilities: "1000000",
              equity: "4000000",
              revenue: "5000000",
              expense: "0",
              netIncome: "5000000",
            },
            monthly: [{ month: "2026-01", revenue: "5000000", expense: "0" }],
          },
        };
      }
      if (key?.includes("entries")) {
        return {
          data: [
            {
              id: 1,
              number: "JE-001",
              entryDate: "2026-01-10",
              description: "Sample Entry",
              status: "POSTED",
              postings: [{ direction: "DEBIT", amount: "5000000" }],
            },
          ],
        };
      }
      return { data: [] };
    }),
  };
});

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("FinDashboard (/fin/overview)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("clientLoader ensures financial summary and recent entries queries", async () => {
    vi.mocked(requireWorkspaceId).mockResolvedValueOnce(15);
    const mockEnsure = vi.fn().mockResolvedValue({});
    vi.mocked(getQueryClient).mockReturnValue({
      ensureQueryData: mockEnsure,
    } as unknown as ReturnType<typeof getQueryClient>);

    const res = await clientLoader();
    expect(res).toEqual({ wid: 15 });
    expect(mockEnsure).toHaveBeenCalledTimes(2);
    expect(mockEnsure).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: financialSummaryQueryOptions(15, 6).queryKey,
      }),
    );
    expect(mockEnsure).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: recentEntriesQueryOptions(15, 5).queryKey,
      }),
    );
  });

  it("renders summary statistics, revenue vs expense chart, and recent entries without accounts count or accounts by type", () => {
    const html = renderToString(
      <FinDashboard
        loaderData={{ wid: 15 }}
        params={{}}
        matches={emptyMatches}
      />,
    );

    expect(html).toContain("fin.dashboard.assets");
    expect(html).toContain("fin.dashboard.liabilities");
    expect(html).toContain("fin.dashboard.netIncome");
    expect(html).toContain("fin.dashboard.revenueVsExpense");
    expect(html).toContain("fin.dashboard.recentEntries");
    expect(html).not.toContain("fin.dashboard.accounts");
    expect(html).not.toContain("fin.dashboard.accountsByType");
  });
});
