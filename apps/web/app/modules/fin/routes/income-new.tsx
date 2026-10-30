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
  incomeKeys,
  transactionKeys,
} from "../queries";
import type { Route } from "./+types/income-new";

export type CreateIncomeFormValues = {
  transactionDate: Dayjs;
  paymentAccountId: number;
  revenueAccountId: number;
  amount: number | string;
  currency?: string;
  exchangeRate?: number;
  description?: string;
  reference?: string;
};

export interface SubmitIncomeFormArgs {
  wid: number;
  values: {
    transactionDate?: Dayjs | string;
    paymentAccountId: number;
    revenueAccountId: number;
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

export async function submitIncomeForm({
  wid,
  values,
  pendingFiles,
  queryClient,
  navigate,
}: SubmitIncomeFormArgs) {
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

  const payload: components["schemas"]["model.CreateIncomeRequest"] = {
    transactionDate: txDate,
    paymentAccountId: values.paymentAccountId,
    revenueAccountId: values.revenueAccountId,
    amount: amountStr,
    currency: values.currency || "IDR",
    exchangeRate: String(values.exchangeRate ?? 1),
    description: values.description || undefined,
    reference: values.reference || undefined,
    attachmentFileIds:
      attachmentFileIds.length > 0 ? attachmentFileIds : undefined,
  };

  await unwrapOne(
    api.POST("/workspaces/{wid}/fin/income", {
      params: { path: { wid } },
      body: payload,
    }),
  );

  await queryClient.invalidateQueries({ queryKey: incomeKeys.all(wid) });
  await queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) });
  await queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) });

  navigate("/fin/income");
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

export default function IncomeNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );

  const [form] = Form.useForm<CreateIncomeFormValues>();
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);

  const paymentGroups = useMemo(
    () => groupPaymentAccounts(accounts),
    [accounts],
  );

  const revenueCategoryOptions = useMemo(() => {
    return accounts
      .filter(
        (a) => a.isActive !== false && a.isPostable && a.type === "REVENUE",
      )
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
        account: a,
      }));
  }, [accounts]);

  const watchedPaymentId = Form.useWatch("paymentAccountId", form);
  const watchedRevenueId = Form.useWatch("revenueAccountId", form);
  const watchedAmount = Form.useWatch("amount", form);
  const watchedCurrency = Form.useWatch("currency", form) ?? "IDR";

  const selectedPaymentAccount = useMemo(
    () => accounts.find((a) => a.id === watchedPaymentId),
    [accounts, watchedPaymentId],
  );

  const selectedRevenueAccount = useMemo(
    () => accounts.find((a) => a.id === watchedRevenueId),
    [accounts, watchedRevenueId],
  );

  const handleFinish = async (values: CreateIncomeFormValues) => {
    await run(async () => {
      await submitIncomeForm({
        wid,
        values,
        pendingFiles,
        queryClient,
        navigate,
      });
    }, "fin.income.createdSuccess");
  };

  return (
    <Page
      title={t("fin.income.new", "New Income")}
      trail={[
        {
          title: t("fin.menu.income", "Income"),
          onClick: () => navigate("/fin/income"),
        },
      ]}
    >
      <FormView
        title={t("fin.income.new", "New Income")}
        buttons={
          <div className="flex gap-2">
            <Button
              type="primary"
              loading={pending}
              onClick={() => form.submit()}
            >
              {t("fin.income.save", "Save Income")}
            </Button>
            <Button disabled={pending} onClick={() => navigate("/fin/income")}>
              {t("common.cancel", "Cancel")}
            </Button>
          </div>
        }
      >
        <Form<CreateIncomeFormValues>
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
              label={t("fin.income.depositAccount", "Deposit Account")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Select
                showSearch
                placeholder={t(
                  "fin.income.depositAccountPlaceholder",
                  "Select deposit account",
                )}
                optionFilterProp="label"
                options={paymentGroups}
              />
            </Form.Item>

            <Form.Item
              name="revenueAccountId"
              label={t("fin.income.revenueCategory", "Revenue Category")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Select
                showSearch
                placeholder={t(
                  "fin.income.revenueCategoryPlaceholder",
                  "Select revenue category",
                )}
                optionFilterProp="label"
                options={revenueCategoryOptions}
              />
            </Form.Item>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-x-4">
            <Form.Item
              name="amount"
              label={t("fin.income.amount", "Amount")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <InputNumber
                className="w-full"
                min={0.01}
                step={1000}
                placeholder={t("fin.income.amountPlaceholder", "0.00")}
              />
            </Form.Item>

            <Form.Item
              name="transactionDate"
              label={t("fin.income.date", "Date")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <DatePicker className="w-full" />
            </Form.Item>

            <Form.Item
              name="currency"
              label={t("fin.income.currency", "Currency")}
            >
              <CurrencySelect currencies={currencies} />
            </Form.Item>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <Form.Item
              name="description"
              label={t("fin.income.description", "Description")}
            >
              <Input
                placeholder={t(
                  "fin.income.descriptionPlaceholder",
                  "e.g. Client consulting revenue",
                )}
              />
            </Form.Item>

            <Form.Item
              name="reference"
              label={t("fin.income.reference", "Reference")}
            >
              <Input
                placeholder={t(
                  "fin.income.referencePlaceholder",
                  "e.g. INV-2026-001",
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
              {t("fin.income.entryPreview", "Accounting Preview")}
            </div>
            <div className="flex flex-col gap-1 text-sm font-mono">
              <div className="flex items-center justify-between">
                <span>
                  <strong className="text-emerald-600">(D) Payment:</strong>{" "}
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
              <div className="flex items-center justify-between pl-4">
                <span>
                  <strong className="text-blue-600">(K) Revenue:</strong>{" "}
                  {selectedRevenueAccount
                    ? `${selectedRevenueAccount.code} - ${selectedRevenueAccount.name}`
                    : t("fin.types.REVENUE", "Revenue")}
                </span>
                <span>
                  {watchedAmount
                    ? `${watchedCurrency} ${Number(watchedAmount).toLocaleString()}`
                    : "-"}
                </span>
              </div>
            </div>
          </div>

          {/* Attachment Picker */}
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
