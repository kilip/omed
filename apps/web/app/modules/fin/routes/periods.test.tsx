import type { components } from "@omed/openapi";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { periodsQueryOptions } from "../queries";
import type { Route } from "./+types/periods";
import Periods, {
  clientLoader,
  filterPeriods,
  getFiscalYears,
} from "./periods";

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
    useLocation: () => ({ pathname: "/fin/periods" }),
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
          name: "January 2026",
          fiscalYear: 2026,
          startDate: "2026-01-01",
          endDate: "2026-01-31",
          status: "OPEN",
          description: "First period",
        },
        {
          id: 2,
          name: "December 2025",
          fiscalYear: 2025,
          startDate: "2025-12-01",
          endDate: "2025-12-31",
          status: "CLOSED",
          description: "Year end",
        },
      ],
    }),
  };
});

type Period = components["schemas"]["model.PeriodResponse"];
const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

describe("Periods route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getFiscalYears", () => {
    it("extracts unique fiscal years sorted descending", () => {
      const periods: Partial<Period>[] = [
        { id: 1, fiscalYear: 2024 },
        { id: 2, fiscalYear: 2026 },
        { id: 3, fiscalYear: 2025 },
        { id: 4, fiscalYear: 2026 },
        { id: 5, fiscalYear: undefined },
      ];

      expect(getFiscalYears(periods as Period[])).toEqual([2026, 2025, 2024]);
    });
  });

  describe("filterPeriods", () => {
    const samplePeriods: Partial<Period>[] = [
      {
        id: 1,
        name: "January 2026",
        fiscalYear: 2026,
        status: "OPEN",
        description: "Q1 start",
      },
      {
        id: 2,
        name: "February 2026",
        fiscalYear: 2026,
        status: "CLOSED",
        description: "Mid quarter",
      },
      {
        id: 3,
        name: "December 2025",
        fiscalYear: 2025,
        status: "CLOSED",
        description: "Year end audit",
      },
    ];

    it("returns all periods when no filters are set", () => {
      const result = filterPeriods(samplePeriods as Period[], "ALL", "ALL", "");
      expect(result).toHaveLength(3);
    });

    it("filters by status", () => {
      const openOnly = filterPeriods(
        samplePeriods as Period[],
        "OPEN",
        "ALL",
        "",
      );
      expect(openOnly).toHaveLength(1);
      expect(openOnly[0].id).toBe(1);

      const closedOnly = filterPeriods(
        samplePeriods as Period[],
        "CLOSED",
        "ALL",
        "",
      );
      expect(closedOnly).toHaveLength(2);
    });

    it("filters by fiscal year", () => {
      const year2025 = filterPeriods(
        samplePeriods as Period[],
        "ALL",
        2025,
        "",
      );
      expect(year2025).toHaveLength(1);
      expect(year2025[0].fiscalYear).toBe(2025);
    });

    it("filters by searchQuery matching name, description, or year", () => {
      const matchName = filterPeriods(
        samplePeriods as Period[],
        "ALL",
        "ALL",
        "february",
      );
      expect(matchName).toHaveLength(1);
      expect(matchName[0].id).toBe(2);

      const matchDesc = filterPeriods(
        samplePeriods as Period[],
        "ALL",
        "ALL",
        "audit",
      );
      expect(matchDesc).toHaveLength(1);
      expect(matchDesc[0].id).toBe(3);

      const matchYear = filterPeriods(
        samplePeriods as Period[],
        "ALL",
        "ALL",
        "2025",
      );
      expect(matchYear).toHaveLength(1);
      expect(matchYear[0].id).toBe(3);
    });
  });

  describe("clientLoader", () => {
    it("ensures periodsQueryOptions and returns wid", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(55);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();
      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: periodsQueryOptions(55).queryKey }),
      );
      expect(result).toEqual({ wid: 55 });
    });
  });

  describe("Periods component", () => {
    it("renders period names and action elements", () => {
      const html = renderToString(
        <Periods loaderData={{ wid: 55 }} params={{}} matches={emptyMatches} />,
      );

      expect(html).toContain("January 2026");
      expect(html).toContain("December 2025");
      expect(html).toContain("fin.periods.new");
    });
  });
});
