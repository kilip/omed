import dayjs from "dayjs";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  CoaTemplateFields,
  CurrencyFormatFields,
  FiscalPeriodFields,
} from "../components";
import { finSettingsQueryOptions } from "../queries";
import Onboarding, { clientLoader } from "./onboarding";

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn(),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

let mockLang = "en";
vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.count !== undefined ? `${key}_${params.count}` : key,
      i18n: {
        get language() {
          return mockLang;
        },
      },
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
    useLocation: () => ({ pathname: "/fin/onboarding" }),
    redirect: vi.fn((path: string) => {
      const response = new Response(null, {
        status: 302,
        headers: { Location: path },
      });
      return response;
    }),
  };
});

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
      removeQueries: vi.fn(),
    }),
    useQuery: vi.fn().mockImplementation((options) => {
      // If query is for currencies
      if (options?.queryKey?.[3] === "currencies") {
        return {
          data: [
            { code: "USD", name: "US Dollar", symbol: "$" },
            { code: "IDR", name: "Indonesian Rupiah", symbol: "Rp" },
          ],
          isLoading: false,
        };
      }
      // If query is for templates
      if (options?.queryKey?.[3] === "templates") {
        return {
          data: {
            accounts: [
              {
                code: "1000",
                name: "Assets",
                type: "ASSET",
                isPostable: false,
              },
              { code: "1100", name: "Cash", type: "ASSET", isPostable: true },
            ],
          },
          isLoading: false,
        };
      }
      return { data: undefined, isLoading: false };
    }),
  };
});

describe("Onboarding route", () => {
  const wid = 42;
  const mockEnsureQueryData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockLang = "en";
    vi.mocked(requireWorkspaceId).mockResolvedValue(wid);
    vi.mocked(getQueryClient).mockReturnValue({
      ensureQueryData: mockEnsureQueryData,
    } as unknown as ReturnType<typeof getQueryClient>);
  });

  describe("clientLoader", () => {
    it("redirects to /fin when settings.onboarding is true", async () => {
      mockEnsureQueryData.mockResolvedValueOnce({
        baseCurrency: "USD",
        decimalStyle: "dot",
        onboarding: true,
      });

      const res = await clientLoader();

      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: finSettingsQueryOptions(wid).queryKey,
        }),
      );
      expect(res).toBeInstanceOf(Response);
      expect((res as Response).status).toBe(302);
      expect((res as Response).headers.get("Location")).toBe("/fin");
    });

    it("returns { wid } when settings.onboarding is false", async () => {
      mockEnsureQueryData.mockResolvedValueOnce({
        baseCurrency: "IDR",
        decimalStyle: "comma",
        onboarding: false,
      });

      const res = await clientLoader();

      expect(requireWorkspaceId).toHaveBeenCalled();
      expect(res).toEqual({ wid });
    });
  });

  describe("Onboarding component rendering", () => {
    it("renders step 1 with currency, decimal style (dot), and fiscal period fields for english", () => {
      mockLang = "en";
      const html = renderToString(<Onboarding loaderData={{ wid }} />);

      // Steps indicator
      expect(html).toContain("fin.onboarding.step1Title");
      expect(html).toContain("fin.onboarding.step2Title");

      // Currency & format
      expect(html).toContain("fin.onboarding.baseCurrency");
      expect(html).toContain("fin.onboarding.decimalStyle");
      expect(html).toContain("USD");

      // Initial period
      expect(html).toContain("fin.onboarding.fiscalYear");
      expect(html).toContain("fin.onboarding.periodName");
      expect(html).toContain("fin.onboarding.periodDates");

      // Action button
      expect(html).toContain("fin.onboarding.next");
    });

    it("defaults to IDR and comma decimal style for indonesian", () => {
      mockLang = "id";
      const html = renderToString(<Onboarding loaderData={{ wid }} />);

      expect(html).toContain("IDR");
      expect(html).toContain("fin.onboarding.commaDesc");
    });
  });

  describe("Reusable form components", () => {
    it("renders CurrencyFormatFields with live preview", () => {
      const html = renderToString(
        <CurrencyFormatFields
          wid={wid}
          baseCurrency="USD"
          onBaseCurrencyChange={vi.fn()}
          decimalStyle="dot"
          onDecimalStyleChange={vi.fn()}
        />,
      );

      expect(html).toContain("fin.onboarding.baseCurrency");
      expect(html).toContain("fin.onboarding.decimalStyle");
      expect(html).toContain("1,234,567.89");
    });

    it("renders CurrencyFormatFields with disabled currency when baseCurrencyDisabled is true", () => {
      const html = renderToString(
        <CurrencyFormatFields
          wid={wid}
          baseCurrency="USD"
          onBaseCurrencyChange={vi.fn()}
          decimalStyle="comma"
          onDecimalStyleChange={vi.fn()}
          baseCurrencyDisabled
        />,
      );

      expect(html).toContain("1.234.567,89");
      expect(html).toContain("ant-select-disabled");
    });

    it("renders FiscalPeriodFields", () => {
      const html = renderToString(
        <FiscalPeriodFields
          fiscalYear={2026}
          onFiscalYearChange={vi.fn()}
          periodName="FY 2026"
          onPeriodNameChange={vi.fn()}
          dateRange={[dayjs("2026-01-01"), dayjs("2026-12-31")]}
          onDateRangeChange={vi.fn()}
          description="Fiscal Year 2026"
          onDescriptionChange={vi.fn()}
        />,
      );

      expect(html).toContain("2026");
      expect(html).toContain("FY 2026");
      expect(html).toContain("fin.onboarding.periodName");
      expect(html).toContain("fin.onboarding.fiscalYear");
    });

    it("renders CoaTemplateFields in template mode", () => {
      const html = renderToString(
        <CoaTemplateFields
          wid={wid}
          coaMode="template"
          onCoaModeChange={vi.fn()}
          template="freelancer"
          onTemplateChange={vi.fn()}
          lang="en"
          onLangChange={vi.fn()}
        />,
      );

      expect(html).toContain("fin.onboarding.modeTemplate");
      expect(html).toContain("fin.onboarding.modeBlank");
      expect(html).toContain("1000");
      expect(html).toContain("Assets");
    });

    it("renders CoaTemplateFields in blank mode", () => {
      const html = renderToString(
        <CoaTemplateFields
          wid={wid}
          coaMode="blank"
          onCoaModeChange={vi.fn()}
          template="freelancer"
          onTemplateChange={vi.fn()}
          lang="en"
          onLangChange={vi.fn()}
        />,
      );

      expect(html).toContain("fin.onboarding.modeBlank");
      expect(html).toContain("fin.onboarding.modeBlankDesc");
    });
  });
});
