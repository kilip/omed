import type { components } from "@omed/openapi";
import type { QueryClient } from "@tanstack/react-query";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Button, DatePicker, Form, Input, InputNumber, Select } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { api, unwrapOne } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  CurrencySelect,
  groupPaymentAccounts,
  PendingAttachmentsPicker,
  uploadWorkspaceFile,
} from "../components";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  entryKeys,
  expenseKeys,
  transactionKeys,
} from "../queries";
import type { Route } from "./+types/expense-new";

export type CreateExpenseFormValues = {
  transactionDate: Dayjs;
  paymentAccountId: number;
  expenseAccountId: number;
  amount: number | string;
  currency?: string;
  exchangeRate?: number;
  description?: string;
  reference?: string;
};

export interface SubmitExpenseFormArgs {
  wid: number;
  values: {
    transactionDate?: Dayjs | string;
    paymentAccountId: number;
    expenseAccountId: number;
    amount: number | string;
    currency?: string;
    exchangeRate?: number;
    description?: string;
    reference?: string;
  };
  pendingFiles: File[];
  queryClient: Pick<QueryClient, "invalidateQueries">;
  navigate: (to: string) => void;
}

export async function submitExpenseForm({
  wid,
  values,
  pendingFiles,
  queryClient,
  navigate,
}: SubmitExpenseFormArgs) {
  let attachmentFileIds: number[] = [];
  if (pendingFiles.length > 0) {
    attachmentFileIds = await Promise.all(
      pendingFiles.map((file) => uploadWorkspaceFile(wid, file)),
    );
  }

  const txDate = values.transactionDate
    ? typeof (values.transactionDate as Dayjs).format === "function"
      ? (values.transactionDate as Dayjs).format("YYYY-MM-DD")
      : String(values.transactionDate)
    : dayjs().format("YYYY-MM-DD");

  const amountStr =
    typeof values.amount === "number"
      ? values.amount.toFixed(2)
      : String(values.amount);

  const payload: components["schemas"]["model.CreateExpenseRequest"] = {
    transactionDate: txDate,
    paymentAccountId: values.paymentAccountId,
    expenseAccountId: values.expenseAccountId,
    amount: amountStr,
    currency: values.currency || "IDR",
    exchangeRate: String(values.exchangeRate ?? 1),
    description: values.description || undefined,
    reference: values.reference || undefined,
    attachmentFileIds:
      attachmentFileIds.length > 0 ? attachmentFileIds : undefined,
  };

  await unwrapOne(
    api.POST("/workspaces/{wid}/fin/expense", {
      params: { path: { wid } },
      body: payload,
    }),
  );

  await queryClient.invalidateQueries({ queryKey: expenseKeys.all(wid) });
  await queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) });
  await queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) });

  navigate("/fin/expense");
}

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(currenciesQueryOptions(wid)),
  ]);
  return { wid };
}

export default function ExpenseNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );

  const [form] = Form.useForm<CreateExpenseFormValues>();
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);

  const paymentGroups = useMemo(
    () => groupPaymentAccounts(accounts),
    [accounts],
  );

  const expenseCategoryOptions = useMemo(() => {
    return accounts
      .filter(
        (a) => a.isActive !== false && a.isPostable && a.type === "EXPENSE",
      )
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
        account: a,
      }));
  }, [accounts]);

  const watchedPaymentId = Form.useWatch("paymentAccountId", form);
  const watchedExpenseId = Form.useWatch("expenseAccountId", form);
  const watchedAmount = Form.useWatch("amount", form);
  const watchedCurrency = Form.useWatch("currency", form) ?? "IDR";

  const selectedPaymentAccount = useMemo(
    () => accounts.find((a) => a.id === watchedPaymentId),
    [accounts, watchedPaymentId],
  );

  const selectedExpenseAccount = useMemo(
    () => accounts.find((a) => a.id === watchedExpenseId),
    [accounts, watchedExpenseId],
  );

  const handleFinish = async (values: CreateExpenseFormValues) => {
    await run(async () => {
      await submitExpenseForm({
        wid,
        values,
        pendingFiles,
        queryClient,
        navigate,
      });
    }, "fin.expense.createdSuccess");
  };

  return (
    <Page
      title={t("fin.expense.new", "New Expense")}
      trail={[
        {
          title: t("fin.menu.expense", "Expense"),
          onClick: () => navigate("/fin/expense"),
        },
      ]}
    >
      <FormView
        title={t("fin.expense.new", "New Expense")}
        buttons={
          <div className="flex gap-2">
            <Button
              type="primary"
              loading={pending}
              onClick={() => form.submit()}
            >
              {t("fin.expense.save", "Save Expense")}
            </Button>
            <Button disabled={pending} onClick={() => navigate("/fin/expense")}>
              {t("common.cancel", "Cancel")}
            </Button>
          </div>
        }
      >
        <Form<CreateExpenseFormValues>
          form={form}
          layout="vertical"
          size="small"
          initialValues={{
            transactionDate: dayjs(),
            currency: "IDR",
            exchangeRate: 1,
          }}
          onFinish={(values) => void handleFinish(values)}
          className="max-w-3xl"
        >
          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <Form.Item
              name="paymentAccountId"
              label={t("fin.expense.paymentAccount", "Payment Account")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Select
                showSearch
                placeholder={t(
                  "fin.expense.paymentAccountPlaceholder",
                  "Select payment account",
                )}
                optionFilterProp="label"
                options={paymentGroups}
              />
            </Form.Item>

            <Form.Item
              name="expenseAccountId"
              label={t("fin.expense.expenseCategory", "Expense Category")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Select
                showSearch
                placeholder={t(
                  "fin.expense.expenseCategoryPlaceholder",
                  "Select expense category",
                )}
                optionFilterProp="label"
                options={expenseCategoryOptions}
              />
            </Form.Item>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-x-4">
            <Form.Item
              name="amount"
              label={t("fin.expense.amount", "Amount")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <InputNumber
                className="w-full"
                min={0.01}
                step={1000}
                placeholder={t("fin.expense.amountPlaceholder", "0.00")}
              />
            </Form.Item>

            <Form.Item
              name="transactionDate"
              label={t("fin.expense.date", "Date")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <DatePicker className="w-full" />
            </Form.Item>

            <Form.Item
              name="currency"
              label={t("fin.expense.currency", "Currency")}
            >
              <CurrencySelect currencies={currencies} />
            </Form.Item>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <Form.Item
              name="description"
              label={t("fin.expense.description", "Description")}
            >
              <Input
                placeholder={t(
                  "fin.expense.descriptionPlaceholder",
                  "e.g. Office supplies or software subscription",
                )}
              />
            </Form.Item>

            <Form.Item
              name="reference"
              label={t("fin.expense.reference", "Reference / Receipt #")}
            >
              <Input
                placeholder={t(
                  "fin.expense.referencePlaceholder",
                  "e.g. RCPT-2026-001",
                )}
              />
            </Form.Item>
          </div>

          {/* Live Double-Entry Preview */}
          <div
            className="p-3 rounded border my-4"
            style={{
              borderColor: "var(--omed-border)",
              background: "var(--omed-surface-muted)",
            }}
            data-testid="cashflow-double-entry-preview"
          >
            <div
              className="text-xs font-semibold uppercase tracking-wider mb-2"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {t("fin.expense.entryPreview", "Accounting Preview")}
            </div>
            <div className="flex flex-col gap-1 text-sm font-mono">
              <div className="flex items-center justify-between">
                <span>
                  <strong className="text-emerald-600">(D) Expense:</strong>{" "}
                  {selectedExpenseAccount
                    ? `${selectedExpenseAccount.code} - ${selectedExpenseAccount.name}`
                    : t("fin.types.EXPENSE", "Expense")}
                </span>
                <span>
                  {watchedAmount
                    ? `${watchedCurrency} ${Number(watchedAmount).toLocaleString()}`
                    : "-"}
                </span>
              </div>
              <div className="flex items-center justify-between pl-4">
                <span>
                  <strong className="text-blue-600">(K) Payment:</strong>{" "}
                  {selectedPaymentAccount
                    ? `${selectedPaymentAccount.code} - ${selectedPaymentAccount.name}`
                    : t("fin.cashflow.paymentAccount", "Payment")}
                </span>
                <span>
                  {watchedAmount
                    ? `${watchedCurrency} ${Number(watchedAmount).toLocaleString()}`
                    : "-"}
                </span>
              </div>
            </div>
          </div>

          {/* Receipt / Attachment Picker */}
          <div className="mt-4">
            <PendingAttachmentsPicker
              files={pendingFiles}
              onChange={setPendingFiles}
              disabled={pending}
            />
          </div>
        </Form>
      </FormView>
    </Page>
  );
}
