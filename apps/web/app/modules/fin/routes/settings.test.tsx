import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountsQueryOptions,
  finSettingsQueryOptions,
  periodsQueryOptions,
  recentEntriesQueryOptions,
} from "../queries";
import SettingsRoute, { clientLoader } from "./settings";

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
      t: (key: string, params?: Record<string, unknown>) =>
        params?.count !== undefined ? `${key}_${params.count}` : key,
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
    useLocation: () => ({ pathname: "/fin/settings" }),
  };
});

let mockPostedEntries: Array<{ id: number; status: string }> = [
  { id: 1, status: "POSTED" },
];
let mockAccounts: Array<{
  id: number;
  code: string;
  name: string;
  type: string;
  isPostable: boolean;
  isActive: boolean;
}> = [
  {
    id: 101,
    code: "7100",
    name: "FX Gain",
    type: "REVENUE",
    isPostable: true,
    isActive: true,
  },
  {
    id: 102,
    code: "7200",
    name: "FX Loss",
    type: "EXPENSE",
    isPostable: true,
    isActive: true,
  },
];
let mockPeriods: Array<{
  id: number;
  name: string;
  fiscalYear: number;
  startDate: string;
  endDate: string;
  status: string;
}> = [
  {
    id: 1,
    name: "FY 2026",
    fiscalYear: 2026,
    startDate: "2026-01-01",
    endDate: "2026-12-31",
    status: "OPEN",
  },
];
let mockSettings = {
  baseCurrency: "USD",
  decimalStyle: "dot",
  onboarding: true,
  fxGainAccountId: 101,
  fxLossAccountId: 102,
};

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
    }),
    useSuspenseQuery: vi.fn().mockImplementation((options) => {
      const key = options?.queryKey;
      if (key?.[3] === "settings") {
        return { data: mockSettings };
      }
      if (key?.[3] === "periods") {
        return { data: mockPeriods };
      }
      if (key?.[3] === "accounts") {
        return { data: mockAccounts };
      }
      if (key?.[3] === "entries") {
        return { data: mockPostedEntries };
      }
      return { data: undefined };
    }),
    useQuery: vi.fn().mockImplementation((options) => {
      const key = options?.queryKey;
      if (key?.[3] === "currencies") {
        return {
          data: [
            { code: "USD", name: "US Dollar", symbol: "$" },
            { code: "IDR", name: "Indonesian Rupiah", symbol: "Rp" },
          ],
          isLoading: false,
        };
      }
      if (key?.[3] === "templates") {
        return {
          data: {
            accounts: [
              {
                code: "1000",
                name: "Assets",
                type: "ASSET",
                isPostable: false,
              },
            ],
          },
          isLoading: false,
        };
      }
      return { data: undefined, isLoading: false };
    }),
  };
});

describe("Settings route", () => {
  const wid = 42;

  beforeEach(() => {
    vi.clearAllMocks();
    mockPostedEntries = [{ id: 1, status: "POSTED" }];
    mockAccounts = [
      {
        id: 101,
        code: "7100",
        name: "FX Gain",
        type: "REVENUE",
        isPostable: true,
        isActive: true,
      },
      {
        id: 102,
        code: "7200",
        name: "FX Loss",
        type: "EXPENSE",
        isPostable: true,
        isActive: true,
      },
    ];
    mockPeriods = [
      {
        id: 1,
        name: "FY 2026",
        fiscalYear: 2026,
        startDate: "2026-01-01",
        endDate: "2026-12-31",
        status: "OPEN",
      },
    ];
    mockSettings = {
      baseCurrency: "USD",
      decimalStyle: "dot",
      onboarding: true,
      fxGainAccountId: 101,
      fxLossAccountId: 102,
    };
  });

  describe("clientLoader", () => {
    it("fetches settings, periods, accounts, and recent posted entries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(wid);
      const ensureQueryData = vi.fn().mockResolvedValue({});
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();

      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(ensureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: finSettingsQueryOptions(wid).queryKey,
        }),
      );
      expect(ensureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: periodsQueryOptions(wid).queryKey,
        }),
      );
      expect(ensureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(wid).queryKey,
        }),
      );
      expect(ensureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: recentEntriesQueryOptions(wid, 1, "POSTED").queryKey,
        }),
      );
      expect(result).toEqual({ wid });
    });
  });

  describe("SettingsRoute rendering", () => {
    it("renders Currency & Number Format panel, Fiscal Periods panel, and CoA / FX Accounts panel", () => {
      const html = renderToString(<SettingsRoute loaderData={{ wid }} />);

      // Panels
      expect(html).toContain("fin.settings.currencyFormatTitle");
      expect(html).toContain("fin.settings.fiscalPeriodsTitle");
      expect(html).toContain("fin.settings.coaFxTitle");

      // Fiscal periods table content
      expect(html).toContain("FY 2026");
      expect(html).toContain("2026-01-01");
      expect(html).toContain("2026-12-31");
      expect(html).toContain("fin.periods.statusOpen");
      expect(html).toContain("fin.settings.newPeriod");
      expect(html).toContain('href="/fin/periods"');

      // FX Mappings
      expect(html).toContain("fin.settings.fxAccountsTitle");
      expect(html).toContain("fin.settings.realizedGainAccount");
      expect(html).toContain("fin.settings.realizedLossAccount");
    });

    it("disables base currency selector when posted entries exist", () => {
      mockPostedEntries = [{ id: 1, status: "POSTED" }];
      const html = renderToString(<SettingsRoute loaderData={{ wid }} />);

      // The base currency select component should be disabled
      expect(html).toContain("ant-select-disabled");
    });

    it("enables base currency selector when no posted entries exist", () => {
      mockPostedEntries = [];
      const html = renderToString(<SettingsRoute loaderData={{ wid }} />);

      expect(html).not.toContain("ant-select-disabled");
    });

    it("renders CoaTemplateFields when accounts list is empty", () => {
      mockAccounts = [];
      const html = renderToString(<SettingsRoute loaderData={{ wid }} />);

      // Should render template seeding options
      expect(html).toContain("fin.onboarding.modeTemplate");
      expect(html).toContain("fin.onboarding.templateFreelancer");
      expect(html).toContain("fin.settings.seedTemplateButton");
    });
  });
});
