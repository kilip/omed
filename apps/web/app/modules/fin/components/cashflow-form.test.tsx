import type { QueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import {
  entryKeys,
  expenseKeys,
  incomeKeys,
  transactionKeys,
} from "../queries";
import {
  buildCashflowPayload,
  CashflowActions,
  type CashflowFormValues,
  submitCashflowForm,
} from "./cashflow-form";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      PUT: vi.fn(),
      DELETE: vi.fn(),
    },
    unwrapOne: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data;
    }),
  };
});

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, defaultValue?: string) => defaultValue ?? key,
      i18n: { language: "en" },
    }),
  };
});

describe("CashflowForm and actions", () => {
  const wid = 1;
  const mockNavigate = vi.fn();
  const mockQueryClient: Pick<QueryClient, "invalidateQueries"> = {
    invalidateQueries: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("CashflowActions", () => {
    it("contains Posting and Draft when pendingStatus is null", () => {
      const html = renderToString(
        <CashflowActions
          pendingStatus={null}
          onPosting={vi.fn()}
          onDraft={vi.fn()}
          onCancel={vi.fn()}
        />,
      );
      expect(html).toContain("Posting");
      expect(html).toContain("Draft");
      expect(html).not.toContain("ant-btn-loading");
      expect(html).not.toContain("disabled");
    });

    it("disables both buttons and only Posting has ant-btn-loading when pendingStatus is POSTED (RF5)", () => {
      const html = renderToString(
        <CashflowActions
          pendingStatus="POSTED"
          onPosting={vi.fn()}
          onDraft={vi.fn()}
          onCancel={vi.fn()}
        />,
      );
      // Posting button should have loading class
      expect(html).toContain("ant-btn-loading");
      // Both buttons should carry disabled attribute / class
      const disabledCount = (html.match(/disabled/g) || []).length;
      expect(disabledCount).toBeGreaterThanOrEqual(2);

      // Ensure Draft does not have loading class
      // Ant Design puts ant-btn-loading on the loading button
      // We can check button chunks
      const parts = html.split(/<button/);
      const postingBtn = parts.find((p) => p.includes("Posting"));
      const draftBtn = parts.find((p) => p.includes("Draft"));
      expect(postingBtn).toBeDefined();
      expect(draftBtn).toBeDefined();
      expect(postingBtn).toContain("ant-btn-loading");
      expect(postingBtn).toContain("disabled");
      expect(draftBtn).not.toContain("ant-btn-loading");
      expect(draftBtn).toContain("disabled");
    });
  });

  describe("buildCashflowPayload", () => {
    it("formats quantity and unit price as strings", () => {
      const values: CashflowFormValues = {
        transactionDate: dayjs("2026-01-15"),
        paymentAccountId: 11,
        currency: "IDR",
        exchangeRate: 1,
        reference: "inv-123456",
        description: "Website work",
        items: [
          {
            accountId: 4110,
            description: "Scaffold",
            quantity: 6,
            unitPrice: 500000,
          },
        ],
      };

      const payload = buildCashflowPayload(values, "POSTED", [101]);
      expect(payload).toEqual({
        transactionDate: "2026-01-15",
        paymentAccountId: 11,
        currency: "IDR",
        exchangeRate: "1",
        reference: "inv-123456",
        description: "Website work",
        status: "POSTED",
        attachmentFileIds: [101],
        items: [
          {
            accountId: 4110,
            description: "Scaffold",
            quantity: "6",
            unitPrice: "500000",
          },
        ],
      });
    });
  });

  describe("submitCashflowForm", () => {
    const baseValues: CashflowFormValues = {
      transactionDate: dayjs("2026-01-15"),
      paymentAccountId: 11,
      currency: "IDR",
      exchangeRate: 1,
      reference: "inv-123456",
      description: "Website work",
      items: [
        {
          accountId: 4110,
          description: "Scaffold",
          quantity: 6,
          unitPrice: 500000,
        },
      ],
    };

    it("posts to /workspaces/{wid}/fin/income with status: POSTED, reference: inv-123456, items quantity 6 and unit price 500000 and navigates to /fin/income", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: { id: 1, number: "TRX-001" } },
      } as never);

      await submitCashflowForm({
        wid,
        flow: "INCOME",
        status: "POSTED",
        values: baseValues,
        pendingFiles: [],
        queryClient: mockQueryClient,
        navigate: mockNavigate,
      });

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/fin/income",
        expect.objectContaining({
          params: { path: { wid } },
          body: expect.objectContaining({
            status: "POSTED",
            reference: "inv-123456",
            items: [
              expect.objectContaining({
                quantity: "6",
                unitPrice: "500000",
              }),
            ],
          }),
        }),
      );

      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: incomeKeys.all(wid),
      });
      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: transactionKeys.all(wid),
      });
      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: entryKeys.all(wid),
      });
      expect(mockNavigate).toHaveBeenCalledWith("/fin/income");
    });

    it("the Draft path sends status: DRAFT", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: { id: 2, number: "TRX-002" } },
      } as never);

      await submitCashflowForm({
        wid,
        flow: "INCOME",
        status: "DRAFT",
        values: baseValues,
        pendingFiles: [],
        queryClient: mockQueryClient,
        navigate: mockNavigate,
      });

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/fin/income",
        expect.objectContaining({
          params: { path: { wid } },
          body: expect.objectContaining({
            status: "DRAFT",
          }),
        }),
      );
    });

    it("with documentId it calls PUT /workspaces/{wid}/fin/expense/{id} and navigates to /fin/expense", async () => {
      vi.mocked(api.PUT).mockResolvedValueOnce({
        data: { data: { id: 42, number: "TRX-042" } },
      } as never);

      await submitCashflowForm({
        wid,
        flow: "EXPENSE",
        documentId: 42,
        status: "POSTED",
        values: baseValues,
        pendingFiles: [],
        queryClient: mockQueryClient,
        navigate: mockNavigate,
      });

      expect(api.PUT).toHaveBeenCalledWith(
        "/workspaces/{wid}/fin/expense/{id}",
        expect.objectContaining({
          params: { path: { wid, id: 42 } },
          body: expect.objectContaining({
            status: "POSTED",
          }),
        }),
      );

      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: expenseKeys.all(wid),
      });
      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: transactionKeys.all(wid),
      });
      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: entryKeys.all(wid),
      });
      expect(mockNavigate).toHaveBeenCalledWith("/fin/expense");
    });
  });
});
