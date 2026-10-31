import type { components } from "@omed/openapi";
import type { FetchQueryOptions } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  groupPaymentAccounts,
  uploadWorkspaceFile,
} from "../components/cashflow-helpers";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  expensesInfiniteQueryOptions,
  incomesInfiniteQueryOptions,
} from "../queries";
import type { Route as ExpenseRouteType } from "./+types/expense";
import type { Route as ExpenseNewRouteType } from "./+types/expense-new";
import type { Route as IncomeRouteType } from "./+types/income";
import type { Route as IncomeNewRouteType } from "./+types/income-new";
import ExpenseRoute, {
  clientLoader as expenseListClientLoader,
} from "./expense";
import ExpenseNewRoute, {
  clientLoader as expenseNewClientLoader,
} from "./expense-new";
import IncomeRoute, { clientLoader as incomeListClientLoader } from "./income";
import IncomeNewRoute, {
  clientLoader as incomeNewClientLoader,
} from "./income-new";

type Account = components["schemas"]["model.AccountResponse"];
type CashFlowItem = components["schemas"]["model.CashFlowItemResponse"];

const mockNavigate = vi.fn();

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn(),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      DELETE: vi.fn(),
    },
    unwrapOne: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data;
    }),
    unwrapPage: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data;
    }),
    unwrapList: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data ?? [];
    }),
  };
});

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, options?: unknown) => {
        if (
          typeof options === "object" &&
          options !== null &&
          "account" in options
        ) {
          return `${key}:${String((options as { account: unknown }).account)}`;
        }
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
    Link: ({
      children,
      to,
      className,
    }: {
      children: React.ReactNode;
      to: string;
      className?: string;
    }) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: "/fin/expense" }),
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({
    run: async (fn: () => Promise<unknown>) => fn(),
    pending: false,
  }),
}));

vi.mock("../hooks", () => ({
  useFinPermissions: () => ({
    canCreateTransaction: true,
  }),
}));

const mockAccounts: Account[] = [
  {
    id: 10,
    code: "1110",
    name: "Kas & E-Wallet",
    type: "ASSET",
    isPostable: false,
    isActive: true,
  },
  {
    id: 11,
    code: "1111",
    name: "Kas Tunai",
    type: "ASSET",
    isPostable: true,
    isActive: true,
    parentId: 10,
  },
  {
    id: 20,
    code: "1120",
    name: "Bank Operasional",
    type: "ASSET",
    isPostable: false,
    isActive: true,
  },
  {
    id: 21,
    code: "1121",
    name: "Bank BCA",
    type: "ASSET",
    isPostable: true,
    isActive: true,
    parentId: 20,
  },
  {
    id: 30,
    code: "1140",
    name: "Piutang Usaha",
    type: "ASSET",
    isPostable: false,
    isActive: true,
  },
  {
    id: 31,
    code: "1141",
    name: "Piutang Klien",
    type: "ASSET",
    isPostable: true,
    isActive: true,
    parentId: 30,
  },
  {
    id: 40,
    code: "1200",
    name: "Aset Tetap",
    type: "ASSET",
    isPostable: false,
    isActive: true,
  },
  {
    id: 41,
    code: "1211",
    name: "Laptop",
    type: "ASSET",
    isPostable: true,
    isActive: true,
    parentId: 40,
  },
  {
    id: 50,
    code: "4100",
    name: "Pendapatan Usaha",
    type: "REVENUE",
    isPostable: false,
    isActive: true,
  },
  {
    id: 51,
    code: "4101",
    name: "Pendapatan Jasa",
    type: "REVENUE",
    isPostable: true,
    isActive: true,
    parentId: 50,
  },
  {
    id: 60,
    code: "5100",
    name: "Beban Operasional",
    type: "EXPENSE",
    isPostable: false,
    isActive: true,
  },
  {
    id: 61,
    code: "5101",
    name: "Beban Sewa & Kantor",
    type: "EXPENSE",
    isPostable: true,
    isActive: true,
    parentId: 60,
  },
];

const mockCurrencies = [
  { code: "IDR", name: "Indonesian Rupiah", isBase: true },
  { code: "USD", name: "US Dollar", isBase: false },
];

const mockExpenses: CashFlowItem[] = [
  {
    id: 101,
    number: "TRX-2026-0001",
    transactionDate: "2026-01-15",
    type: "EXPENSE",
    amount: "250000.00",
    currency: "IDR",
    description: "Office supplies & snacks",
    reference: "RCPT-001",
    paymentAccount: { id: 11, code: "1111", name: "Kas Tunai", type: "ASSET" },
    categoryAccount: {
      id: 61,
      code: "5101",
      name: "Beban Sewa & Kantor",
      type: "EXPENSE",
    },
    items: [
      {
        id: 1,
        account: {
          id: 61,
          code: "5101",
          name: "Beban Sewa & Kantor",
          type: "EXPENSE",
        },
        quantity: "1",
        unitPrice: "250000.00",
        amount: "250000.00",
      },
    ],
    attachments: [
      { id: 901, name: "receipt.jpg", size: 1024, mimeType: "image/jpeg" },
    ],
  },
];

const mockIncomes: CashFlowItem[] = [
  {
    id: 201,
    number: "TRX-2026-0002",
    transactionDate: "2026-01-16",
    type: "INCOME",
    amount: "5000000.00",
    currency: "IDR",
    description: "Web development consulting",
    reference: "INV-2026-01",
    paymentAccount: { id: 21, code: "1121", name: "Bank BCA", type: "ASSET" },
    categoryAccount: {
      id: 51,
      code: "4101",
      name: "Pendapatan Jasa",
      type: "REVENUE",
    },
    items: [
      {
        id: 2,
        account: {
          id: 51,
          code: "4101",
          name: "Pendapatan Jasa",
          type: "REVENUE",
        },
        quantity: "1",
        unitPrice: "5000000.00",
        amount: "5000000.00",
      },
    ],
    attachments: [],
  },
];

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
        return { data: mockAccounts };
      }
      if (key?.includes("currencies")) {
        return { data: mockCurrencies };
      }
      return { data: [] };
    }),
    useInfiniteQuery: vi
      .fn()
      .mockImplementation((opts: { queryKey: unknown[] }) => {
        const key = opts.queryKey as unknown[];
        if (key?.includes("expense")) {
          return {
            data: {
              pages: [{ items: mockExpenses, cursor: null }],
            },
            fetchNextPage: vi.fn(),
            hasNextPage: false,
            isFetchingNextPage: false,
          };
        }
        if (key?.includes("income")) {
          return {
            data: {
              pages: [{ items: mockIncomes, cursor: null }],
            },
            fetchNextPage: vi.fn(),
            hasNextPage: false,
            isFetchingNextPage: false,
          };
        }
        return {
          data: { pages: [{ items: [], cursor: null }] },
          fetchNextPage: vi.fn(),
          hasNextPage: false,
          isFetchingNextPage: false,
        };
      }),
  };
});

describe("Income and Expense Quick Entry", () => {
  const wid = 7;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("groupPaymentAccounts helper", () => {
    it("groups 1111 Kas Tunai under Kas & E-Wallet and 1121 Bank BCA under Bank Operasional, excluding 1141 and 1211", () => {
      const groups = groupPaymentAccounts(mockAccounts);

      expect(groups).toHaveLength(2);

      const cashGroup = groups.find((g) => g.label.includes("Kas & E-Wallet"));
      expect(cashGroup).toBeDefined();
      expect(cashGroup?.options.some((o) => o.account.code === "1111")).toBe(
        true,
      );

      const bankGroup = groups.find((g) =>
        g.label.includes("Bank Operasional"),
      );
      expect(bankGroup).toBeDefined();
      expect(bankGroup?.options.some((o) => o.account.code === "1121")).toBe(
        true,
      );

      // Verify excluded non-payment accounts (1141 Piutang Klien, 1211 Laptop)
      const allOptionCodes = groups.flatMap((g) =>
        g.options.map((o) => o.account.code),
      );
      expect(allOptionCodes).not.toContain("1141");
      expect(allOptionCodes).not.toContain("1211");
    });
  });

  describe("uploadWorkspaceFile helper", () => {
    it("uploads file to POST /workspaces/{wid}/files and returns res.id", async () => {
      const mockFile = new File(["dummy content"], "invoice.pdf", {
        type: "application/pdf",
      });

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: {
          data: { id: 88, name: "invoice.pdf" },
          status: 201,
        },
      } as never);

      const fileId = await uploadWorkspaceFile(wid, mockFile);
      expect(fileId).toBe(88);
      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/files",
        expect.objectContaining({
          params: { path: { wid } },
        }),
      );
    });
  });

  describe("ExpenseNewRoute", () => {
    it("renders payment select, date, description, items table, and live double-entry preview (D) Expense / (K) Payment", () => {
      const html = renderToString(
        <ExpenseNewRoute
          loaderData={{ wid }}
          params={{}}
          matches={
            [] as unknown as ExpenseNewRouteType.ComponentProps["matches"]
          }
        />,
      );

      // Verify form labels and elements
      expect(html).toContain("fin.expense.paymentAccount");
      expect(html).toContain("fin.expense.date");
      expect(html).toContain("fin.expense.description");

      // Verify double-entry preview with (D) and (K)
      expect(html).toContain("(D)");
      expect(html).toContain("(K)");
      expect(html).toContain("Payment");

      // Verify buttons
      expect(html).toContain("fin.cashflow.posting");
      expect(html).toContain("fin.cashflow.draft");
    });

    it("clientLoader ensures required accounts and currencies queries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(wid);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await expenseNewClientLoader();
      expect(result).toEqual({ wid });
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(wid).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: currenciesQueryOptions(wid).queryKey,
        }),
      );
    });
  });

  describe("IncomeNewRoute", () => {
    it("renders deposit select, date, description, items table, and live double-entry preview (D) Payment / (K) Revenue", () => {
      const html = renderToString(
        <IncomeNewRoute
          loaderData={{ wid }}
          params={{}}
          matches={
            [] as unknown as IncomeNewRouteType.ComponentProps["matches"]
          }
        />,
      );

      // Verify form labels and elements
      expect(html).toContain("fin.income.depositAccount");
      expect(html).toContain("fin.income.date");
      expect(html).toContain("fin.income.description");

      // Verify double-entry preview with (D) and (K)
      expect(html).toContain("(D)");
      expect(html).toContain("(K)");
      expect(html).toContain("Payment");

      // Verify buttons
      expect(html).toContain("fin.cashflow.posting");
      expect(html).toContain("fin.cashflow.draft");
    });

    it("clientLoader ensures required accounts and currencies queries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(wid);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await incomeNewClientLoader();
      expect(result).toEqual({ wid });
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accountsQueryOptions(wid).queryKey,
        }),
      );
      expect(mockEnsure).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: currenciesQueryOptions(wid).queryKey,
        }),
      );
    });
  });

  describe("ExpenseListRoute and IncomeListRoute", () => {
    it("ExpenseListRoute displays history rows and action linking to /fin/expense/new", () => {
      const html = renderToString(
        <ExpenseRoute
          loaderData={{ wid }}
          params={{}}
          matches={[] as unknown as ExpenseRouteType.ComponentProps["matches"]}
        />,
      );

      // Verify row data rendered
      expect(html).toContain("TRX-2026-0001");
      expect(html).toContain('href="/fin/expense/101"');
      expect(html).toContain("Kas Tunai");
      expect(html).toContain("Beban Sewa &amp; Kantor");
      expect(html).toContain("Office supplies &amp; snacks");
      expect(html).toContain("RCPT-001");

      // Verify new action
      expect(html).toContain("fin.expense.new");
    });

    it("IncomeListRoute displays history rows and action linking to /fin/income/new", () => {
      const html = renderToString(
        <IncomeRoute
          loaderData={{ wid }}
          params={{}}
          matches={[] as unknown as IncomeRouteType.ComponentProps["matches"]}
        />,
      );

      // Verify row data rendered
      expect(html).toContain("TRX-2026-0002");
      expect(html).toContain('href="/fin/income/201"');
      expect(html).toContain("Bank BCA");
      expect(html).toContain("Pendapatan Jasa");
      expect(html).toContain("Web development consulting");
      expect(html).toContain("INV-2026-01");

      // Verify new action
      expect(html).toContain("fin.income.new");
    });

    it("list clientLoaders ensure infinite query and accounts queries", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValue(wid);
      const mockEnsure = vi.fn().mockResolvedValue([]);
      const mockEnsureInfinite = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsure,
        ensureInfiniteQueryData: mockEnsureInfinite,
      } as unknown as ReturnType<typeof getQueryClient>);

      await expenseListClientLoader();
      expect(mockEnsureInfinite).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: expensesInfiniteQueryOptions(wid, {}).queryKey,
        }),
      );

      await incomeListClientLoader();
      expect(mockEnsureInfinite).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: incomesInfiniteQueryOptions(wid, {}).queryKey,
        }),
      );
    });
  });
});
