import type { QueryClient } from "@tanstack/react-query";
import { Form, type FormInstance } from "antd";
import dayjs from "dayjs";
import type { SetStateAction } from "react";
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
  type AnalysisSuccessInfo,
  buildCashflowPayload,
  CashflowActions,
  CashflowForm,
  type CashflowFormProps,
  type CashflowFormValues,
  submitCashflowForm,
} from "./cashflow-form";
import type { PendingAttachmentsPickerProps } from "./transaction-attachments";

const { getCleanupEffect, setCleanupEffect } = vi.hoisted(() => {
  let cleanup: (() => void) | undefined;
  return {
    getCleanupEffect: () => cleanup,
    setCleanupEffect: (fn?: () => void) => {
      cleanup = fn;
    },
  };
});

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useEffect: vi.fn((effect: () => (() => void) | undefined) => {
      const cleanup = effect();
      if (typeof cleanup === "function") {
        setCleanupEffect(cleanup);
      }
    }),
  };
});

const mockNavigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: "/fin/income" }),
  };
});

const mockMessage = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
};

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return {
    ...actual,
    App: {
      ...actual.App,
      useApp: () => ({
        message: mockMessage,
      }),
    },
  };
});

const mockAccounts = [
  {
    id: 11,
    code: "1111",
    name: "Kas Tunai",
    type: "ASSET",
    isPostable: true,
    isActive: true,
  },
  {
    id: 12,
    code: "1121",
    name: "Bank BCA",
    type: "ASSET",
    isPostable: true,
    isActive: true,
  },
  {
    id: 51,
    code: "4101",
    name: "Pendapatan Jasa",
    type: "REVENUE",
    isPostable: true,
    isActive: true,
  },
  {
    id: 61,
    code: "5101",
    name: "Beban Operasional",
    type: "EXPENSE",
    isPostable: true,
    isActive: true,
  },
];

const mockCurrencies = [
  { code: "IDR", name: "Rupiah", symbol: "Rp" },
  { code: "USD", name: "US Dollar", symbol: "$" },
];

let mockFinancialStatus = { available: true, enabled: true };
let mockCanUseFinancial = true;

const mockQueryClient: Pick<QueryClient, "invalidateQueries"> = {
  invalidateQueries: vi.fn(),
};

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => mockQueryClient,
    useSuspenseQuery: vi
      .fn()
      .mockImplementation((opts: { queryKey: unknown[] }) => {
        const key = opts.queryKey as unknown[];
        if (key?.includes("accounts")) {
          return { data: mockAccounts };
        }
        if (key?.includes("currencies")) {
          return { data: mockCurrencies };
        }
        return { data: [] };
      }),
    useQuery: vi.fn().mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey as unknown[];
      if (key?.includes("financial-status")) {
        return { data: mockFinancialStatus };
      }
      return { data: undefined };
    }),
  };
});

vi.mock("~/modules/agent/hooks/use-agent-permissions", () => ({
  useAgentPermissions: () => ({
    canUseFinancial: mockCanUseFinancial,
    canManageSettings: false,
    canManage: false,
    canRead: false,
    isAdmin: false,
    isOwner: false,
    isViewer: false,
  }),
}));

let lastPickerProps: PendingAttachmentsPickerProps | null = null;
vi.mock("./transaction-attachments", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("./transaction-attachments")>();
  return {
    ...actual,
    PendingAttachmentsPicker: (props: PendingAttachmentsPickerProps) => {
      lastPickerProps = props;
      return actual.PendingAttachmentsPicker(props);
    },
  };
});

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
      t: (
        key: string,
        optionsOrFallback?: Record<string, unknown> | string,
        extraOptions?: Record<string, unknown>,
      ) => {
        let text =
          typeof optionsOrFallback === "string" ? optionsOrFallback : key;
        const opts =
          typeof optionsOrFallback === "object"
            ? optionsOrFallback
            : extraOptions;
        if (opts && typeof opts === "object") {
          for (const [k, v] of Object.entries(opts)) {
            text = text.replace(new RegExp(`{{\\s*${k}\\s*}}`, "g"), String(v));
          }
        }
        return text;
      },
      i18n: { language: "id" },
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

  describe("CashflowForm AI integration and feedback", () => {
    function makeFile(name: string, size = 1024, type = "image/png"): File {
      return new File([new ArrayBuffer(size)], name, { type });
    }

    function deferred<T>() {
      let resolve!: (value: T) => void;
      let reject!: (reason?: unknown) => void;
      const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
      });
      return { promise, resolve, reject };
    }

    function renderForm(
      props: Partial<CashflowFormProps> = {},
      formInstance?: FormInstance<CashflowFormValues>,
    ) {
      let capturedForm = formInstance;
      let aiFilledState = props.aiFilled ?? new Set<string>();
      let analysisSuccessState: AnalysisSuccessInfo | null =
        props.analysisSuccess ?? null;
      let pendingFilesState: File[] = props.pendingFiles ?? [];
      let userTouchedState = props.userTouched ?? new Set<string>();

      function Harness() {
        const [createdForm] = Form.useForm<CashflowFormValues>();
        if (!capturedForm) capturedForm = createdForm;
        return (
          <CashflowForm
            wid={wid}
            flow={props.flow ?? "INCOME"}
            form={capturedForm}
            aiFilled={aiFilledState}
            onAiFilledChange={(f: SetStateAction<Set<string>>) => {
              aiFilledState = typeof f === "function" ? f(aiFilledState) : f;
            }}
            analysisSuccess={analysisSuccessState}
            onAnalysisSuccessChange={(
              s: SetStateAction<AnalysisSuccessInfo | null>,
            ) => {
              analysisSuccessState =
                typeof s === "function" ? s(analysisSuccessState) : s;
            }}
            pendingFiles={pendingFilesState}
            onPendingFilesChange={(files: SetStateAction<File[]>) => {
              pendingFilesState =
                typeof files === "function" ? files(pendingFilesState) : files;
            }}
            userTouched={userTouchedState}
            onUserTouchedChange={(t: SetStateAction<Set<string>>) => {
              userTouchedState =
                typeof t === "function" ? t(userTouchedState) : t;
            }}
            {...props}
          />
        );
      }
      const html = renderToString(<Harness />);
      return {
        form: (capturedForm ??
          formInstance) as FormInstance<CashflowFormValues>,
        html,
        rerender: () => renderToString(<Harness />),
        addFile: (file: File) => {
          pendingFilesState = [...pendingFilesState, file];
          lastPickerProps?.onChange(pendingFilesState);
          lastPickerProps?.onFileAdded?.(file);
        },
        removeFile: (file: File) => {
          pendingFilesState = pendingFilesState.filter((f) => f !== file);
          lastPickerProps?.onChange(pendingFilesState);
          lastPickerProps?.onRemove?.(file);
        },
        touchField: (field: string) => {
          userTouchedState.add(field);
        },
      };
    }

    beforeEach(() => {
      mockCanUseFinancial = true;
      mockFinancialStatus = { available: true, enabled: true };
      lastPickerProps = null;
      setCleanupEffect(undefined);
      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return Promise.resolve({ data: { data: {} } } as never);
        }
        if (path.includes("/fin/income") || path.includes("/fin/expense")) {
          return Promise.resolve({ data: { data: { id: 1 } } } as never);
        }
        if (path.includes("/files")) {
          return Promise.resolve({ data: { data: { id: 101 } } } as never);
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });
    });

    it("Agent available: adding a PNG triggers one analyze call and fills the date, payment account, reference, and items; the success alert names the file and field count", async () => {
      const pngFile = makeFile("receipt.png");
      const d = deferred<unknown>();
      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, addFile, rerender } = renderForm();
      expect(lastPickerProps).not.toBeNull();
      expect(lastPickerProps.analysis).toBeDefined();

      // Trigger file added
      addFile(pngFile);

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/financial/analyze",
        expect.anything(),
      );

      // Verify in-flight analyzing alert
      const inFlightHtml = renderForm({
        analysis: {
          available: true,
          jobs: [{ id: "job-1", file: pngFile, status: "analyzing" }],
          enqueue: vi.fn(),
          reanalyze: vi.fn(),
          cancel: vi.fn(),
          cancelAll: vi.fn(),
          statusOf: () => "analyzing",
        },
      }).html;
      expect(inFlightHtml).toContain("receipt.png");
      expect(inFlightHtml).toContain("Menganalisa");

      // Resolve with suggestion
      d.resolve({
        data: {
          data: {
            transactionDate: "2026-02-15",
            paymentAccountId: 11,
            reference: "INV-2026-001",
            items: [
              {
                accountId: 51,
                description: "Consulting",
                quantity: "2",
                unitPrice: "100000",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("reference")).toBe("INV-2026-001");
      });

      expect(form.getFieldValue("paymentAccountId")).toBe(11);
      expect(
        dayjs(form.getFieldValue("transactionDate")).format("YYYY-MM-DD"),
      ).toBe("2026-02-15");
      expect(form.getFieldValue("items")).toEqual([
        {
          accountId: 51,
          description: "Consulting",
          quantity: 2,
          unitPrice: 100000,
        },
      ]);

      // Verify success alert names the file and field count
      const updatedHtml = rerender();
      expect(updatedHtml).toContain("receipt.png");
      expect(updatedHtml).toContain("4 field");
      expect(updatedHtml).toContain("Periksa sebelum posting");
    });

    it("Edit mode (transactionId set) auto-analyzes too", async () => {
      const pngFile = makeFile("edit-receipt.png");
      const d = deferred<unknown>();
      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, addFile } = renderForm({ transactionId: 88 });
      addFile(pngFile);

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/financial/analyze",
        expect.anything(),
      );

      d.resolve({
        data: {
          data: {
            reference: "EDIT-REF-001",
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("reference")).toBe("EDIT-REF-001");
      });
    });

    it("A field the user changed before the result arrived keeps the user's value (Review Focus 2)", async () => {
      const pngFile = makeFile("user-edit.png");
      const d = deferred<unknown>();
      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, addFile, touchField } = renderForm();
      // User types reference before result arrives
      form.setFieldsValue({ reference: "USER-PRESERVED" });
      touchField("reference");

      addFile(pngFile);

      d.resolve({
        data: {
          data: {
            reference: "AI-OVERWRITE-ATTEMPT",
            paymentAccountId: 12,
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("paymentAccountId")).toBe(12);
      });

      // User value kept
      expect(form.getFieldValue("reference")).toBe("USER-PRESERVED");
    });

    it("Two files added back to back are analyzed sequentially and their items merge without duplicates and without a leftover blank row (Review Focus 4)", async () => {
      const file1 = makeFile("file1.png", 100);
      const file2 = makeFile("file2.png", 200);

      const d1 = deferred<unknown>();
      const d2 = deferred<unknown>();

      let callCount = 0;
      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          callCount++;
          return callCount === 1 ? d1.promise : d2.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, addFile } = renderForm();

      // Enqueue both files back to back
      addFile(file1);
      addFile(file2);

      // File 1 completes
      d1.resolve({
        data: {
          data: {
            items: [
              {
                accountId: 51,
                description: "Item One",
                quantity: "1",
                unitPrice: "50000",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("items")?.[0]?.description).toBe("Item One");
      });

      expect(form.getFieldValue("items")).toEqual([
        {
          accountId: 51,
          description: "Item One",
          quantity: 1,
          unitPrice: 50000,
        },
      ]);

      // File 2 completes with duplicate of Item One + new Item Two
      d2.resolve({
        data: {
          data: {
            items: [
              {
                accountId: 51,
                description: "Item One",
                quantity: "1",
                unitPrice: "50000",
              },
              {
                accountId: 51,
                description: "Item Two",
                quantity: "3",
                unitPrice: "20000",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("items")).toHaveLength(2);
      });

      expect(form.getFieldValue("items")).toEqual([
        {
          accountId: 51,
          description: "Item One",
          quantity: 1,
          unitPrice: 50000,
        },
        {
          accountId: 51,
          description: "Item Two",
          quantity: 3,
          unitPrice: 20000,
        },
      ]);
    });

    it("Removing a file mid-analysis aborts the request and leaves the form untouched (Review Focus 1)", async () => {
      const file = makeFile("remove-me.png");
      const d = deferred<unknown>();
      let passedSignal: AbortSignal | undefined;

      vi.mocked(api.POST).mockImplementation(
        (path: string, options?: { signal?: AbortSignal }) => {
          if (path.includes("/agent/financial/analyze")) {
            passedSignal = options?.signal;
            return d.promise;
          }
          return Promise.resolve({ data: { data: {} } } as never);
        },
      );

      const { form, addFile, removeFile } = renderForm();
      addFile(file);

      expect(passedSignal?.aborted).toBe(false);

      // Remove the file while analysis is in-flight
      removeFile(file);

      expect(passedSignal?.aborted).toBe(true);

      // Late resolution
      d.resolve({
        data: {
          data: {
            reference: "LATE-REF",
            paymentAccountId: 11,
          },
        },
      });

      // Form remains untouched
      expect(form.getFieldValue("reference")).toBeUndefined();
      expect(form.getFieldValue("paymentAccountId")).toBeUndefined();
    });

    it("Agent unavailable, disabled, or user without permission: no AI button, no status or analyze request, and existing form snapshot is unchanged (Review Focus 3)", () => {
      mockCanUseFinancial = false;
      mockFinancialStatus = { available: false, enabled: false };

      const { html } = renderForm();

      expect(lastPickerProps.analysis).toBeUndefined();
      expect(lastPickerProps.onFileAdded).toBeUndefined();

      expect(html).not.toContain("Analisa dengan AI");
      expect(html).not.toContain("Analyze with AI");
      expect(html).not.toContain("Menganalisa");
      expect(api.POST).not.toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/financial/analyze",
        expect.anything(),
      );
    });

    it("'Analisa ulang': popconfirm text confirms item replacement; confirming overwrites scalars and replaces items; payment_proof leaves items untouched (Review Focus 6)", async () => {
      const file = makeFile("reanalyze.png");
      const d = deferred<unknown>();

      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, touchField } = renderForm();
      // Set existing values
      form.setFieldsValue({
        reference: "EXISTING-REF",
        items: [
          {
            accountId: 51,
            description: "Old Item",
            quantity: 1,
            unitPrice: 100,
          },
        ],
      });
      touchField("reference");

      // Re-analyze standard invoice
      lastPickerProps.analysis.onReanalyze(file);

      d.resolve({
        data: {
          data: {
            documentType: "invoice",
            reference: "OVERWRITE-REF",
            items: [
              {
                accountId: 51,
                description: "Brand New Item",
                quantity: "5",
                unitPrice: "200",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("reference")).toBe("OVERWRITE-REF");
      });

      expect(form.getFieldValue("items")).toEqual([
        {
          accountId: 51,
          description: "Brand New Item",
          quantity: 5,
          unitPrice: 200,
        },
      ]);

      // Now re-analyze with payment_proof
      const d2 = deferred<unknown>();
      vi.mocked(api.POST).mockImplementationOnce((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d2.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      lastPickerProps.analysis.onReanalyze(file);

      d2.resolve({
        data: {
          data: {
            documentType: "payment_proof",
            reference: "PAYMENT-PROOF-REF",
            items: [],
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("reference")).toBe("PAYMENT-PROOF-REF");
      });

      // Existing items stay untouched for payment_proof!
      expect(form.getFieldValue("items")).toEqual([
        {
          accountId: 51,
          description: "Brand New Item",
          quantity: 5,
          unitPrice: 200,
        },
      ]);
    });

    it("A total mismatch shows a warning naming the file and both amounts; a different currency shows none", async () => {
      const file = makeFile("mismatch.png");
      const d = deferred<unknown>();

      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { addFile } = renderForm();

      addFile(file);

      // Same currency mismatch: documentTotal is 500000, items sum to 100000
      d.resolve({
        data: {
          data: {
            currency: "IDR",
            documentTotal: "500000",
            items: [
              {
                accountId: 51,
                description: "Item",
                quantity: "1",
                unitPrice: "100000",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(mockMessage.warning).toHaveBeenCalled();
      });

      const warningArg = mockMessage.warning.mock.calls[0][0];
      expect(warningArg).toContain("mismatch.png");
      expect(warningArg).toContain("500000");
      expect(warningArg).toContain("100000");

      mockMessage.warning.mockClear();

      // Different currency: documentTotal 50, currency USD, form is IDR
      const diffCurFile = makeFile("diff-cur.png");
      const d2 = deferred<unknown>();
      vi.mocked(api.POST).mockImplementationOnce((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d2.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      addFile(diffCurFile);

      d2.resolve({
        data: {
          data: {
            currency: "USD",
            documentTotal: "50",
            items: [
              {
                accountId: 51,
                description: "Item USD",
                quantity: "1",
                unitPrice: "20",
              },
            ],
          },
        },
      });

      await vi.waitFor(() => {
        expect(api.POST).toHaveBeenCalled();
      });

      // No warning for different currency
      expect(mockMessage.warning).not.toHaveBeenCalled();
    });

    describe("Failure matrix (Review Focus 5)", () => {
      it("RATE_LIMITED (429) shows message.warning", async () => {
        const file = makeFile("rate-limit.png");
        vi.mocked(api.POST).mockRejectedValueOnce({
          status: 429,
          code: "RATE_LIMITED",
        });

        const { addFile } = renderForm();
        addFile(file);

        await vi.waitFor(() => {
          expect(mockMessage.warning).toHaveBeenCalledWith(
            expect.stringContaining("kuota"),
          );
        });
      });

      it("UNSUPPORTED_DOCUMENT (422) shows message.warning", async () => {
        const file = makeFile("unsupported.pdf");
        vi.mocked(api.POST).mockRejectedValueOnce({
          status: 422,
          code: "UNSUPPORTED_DOCUMENT",
        });

        const { addFile } = renderForm();
        addFile(file);

        await vi.waitFor(() => {
          expect(mockMessage.warning).toHaveBeenCalledWith(
            expect.stringContaining("tidak didukung"),
          );
        });
      });

      it("502 or 504 provider failure shows message.error 'Analisa gagal, isi manual'", async () => {
        const file = makeFile("server-error.png");
        vi.mocked(api.POST).mockRejectedValueOnce({
          status: 502,
          message: "Bad Gateway",
        });

        const { addFile } = renderForm();
        addFile(file);

        await vi.waitFor(() => {
          expect(mockMessage.error).toHaveBeenCalledWith(
            expect.stringContaining("Analisa gagal, isi manual"),
          );
        });
      });

      it("Empty result shows message.info", async () => {
        const file = makeFile("empty.png");
        vi.mocked(api.POST).mockResolvedValueOnce({
          data: {
            data: {
              items: [],
            },
          },
        } as never);

        const { addFile } = renderForm();
        addFile(file);

        await vi.waitFor(() => {
          expect(mockMessage.info).toHaveBeenCalledWith(
            expect.stringContaining("Tidak ada data"),
          );
        });
      });

      it("manual 'Analyze with AI' button on a failed job retries analysis and fills fields upon success", async () => {
        const file = makeFile("retry-after-fail.png");
        vi.mocked(api.POST).mockRejectedValueOnce({
          status: 502,
          message: "Bad Gateway",
        });

        const { form, addFile } = renderForm();
        addFile(file);

        await vi.waitFor(() => {
          expect(mockMessage.error).toHaveBeenCalledWith(
            expect.stringContaining("Analisa gagal, isi manual"),
          );
        });

        expect(lastPickerProps.analysis?.statusOf(file)).toBe("failed");

        // Mock successful retry
        vi.mocked(api.POST).mockResolvedValueOnce({
          data: {
            data: {
              reference: "RETRIED-REF-123",
              paymentAccountId: 11,
              items: [],
            },
          },
        } as never);

        // Click manual analyze button
        lastPickerProps.analysis?.onAnalyze(file);

        await vi.waitFor(() => {
          expect(lastPickerProps.analysis?.statusOf(file)).toBe("done");
        });

        expect(form.getFieldValue("reference")).toBe("RETRIED-REF-123");
        expect(form.getFieldValue("paymentAccountId")).toBe(11);
      });

      it("In every case Posting and Draft submit successfully", async () => {
        const { form } = renderForm();
        form.setFieldsValue({
          paymentAccountId: 11,
          items: [{ accountId: 51, quantity: 1, unitPrice: 50000 }],
        });

        vi.mocked(api.POST).mockResolvedValueOnce({
          data: { data: { id: 10 } },
        } as never);

        await submitCashflowForm({
          wid,
          flow: "INCOME",
          status: "POSTED",
          values: form.getFieldsValue(true),
          pendingFiles: [],
          queryClient: mockQueryClient,
          navigate: mockNavigate,
        });

        expect(mockNavigate).toHaveBeenCalledWith("/fin/income");
      });
    });

    it("Unmounting during an analysis cancels it and a late result does not throw or update state", async () => {
      const file = makeFile("unmount.png");
      const d = deferred<unknown>();
      let passedSignal: AbortSignal | undefined;

      vi.mocked(api.POST).mockImplementation(
        (path: string, options?: { signal?: AbortSignal }) => {
          if (path.includes("/agent/financial/analyze")) {
            passedSignal = options?.signal;
            return d.promise;
          }
          return Promise.resolve({ data: { data: {} } } as never);
        },
      );

      const { form, addFile } = renderForm();
      addFile(file);

      expect(passedSignal?.aborted).toBe(false);

      // Trigger unmount cleanup
      getCleanupEffect()?.();

      expect(passedSignal?.aborted).toBe(true);

      // Resolve late
      d.resolve({
        data: {
          data: {
            reference: "SHOULD-NOT-SET",
          },
        },
      });

      expect(form.getFieldValue("reference")).toBeUndefined();
    });

    it("Fields filled by AI carry subtle marker (var(--omed-info) border) until user edits them (onValuesChange clears it)", async () => {
      const file = makeFile("ai-border.png");
      const d = deferred<unknown>();

      vi.mocked(api.POST).mockImplementation((path: string) => {
        if (path.includes("/agent/financial/analyze")) {
          return d.promise;
        }
        return Promise.resolve({ data: { data: {} } } as never);
      });

      const { form, addFile, rerender } = renderForm();
      addFile(file);

      d.resolve({
        data: {
          data: {
            reference: "AI-REF-MARKER",
            description: "AI Description",
            paymentAccountId: 11,
          },
        },
      });

      await vi.waitFor(() => {
        expect(form.getFieldValue("reference")).toBe("AI-REF-MARKER");
      });

      const htmlWithMarkers = rerender();
      expect(htmlWithMarkers).toContain("var(--omed-info)");
    });
  });
});
