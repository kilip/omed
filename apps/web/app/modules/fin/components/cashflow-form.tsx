import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import type { QueryClient } from "@tanstack/react-query";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
  App,
  Button,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Select,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { api, unwrapOne } from "~/lib/api";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  entryKeys,
  expenseKeys,
  incomeKeys,
  transactionKeys,
} from "../queries";
import {
  type CashflowItemValue,
  groupPaymentAccounts,
  uploadWorkspaceFile,
} from "./cashflow-helpers";
import { CurrencySelect } from "./currency-select";
import { MoneyInput } from "./money-input";
import { PendingAttachmentsPicker } from "./transaction-attachments";

export type CashflowFlow = "INCOME" | "EXPENSE";
export type CashflowType = CashflowFlow;
export type CashflowStatus = "DRAFT" | "POSTED";

export interface CashflowFormValues {
  transactionDate: Dayjs;
  paymentAccountId: number;
  currency?: string;
  exchangeRate?: number;
  reference?: string;
  description?: string;
  items: CashflowItemValue[];
}

export interface CashflowActionsProps {
  pendingStatus: CashflowStatus | null;
  onPosting: () => void;
  onDraft: () => void;
  onCancel: () => void;
}

export function CashflowActions({
  pendingStatus,
  onPosting,
  onDraft,
  onCancel,
}: CashflowActionsProps) {
  const { t } = useTranslation();
  const isPending = pendingStatus !== null;

  return (
    <div className="flex gap-2">
      <Button
        type="primary"
        loading={pendingStatus === "POSTED"}
        disabled={isPending}
        onClick={onPosting}
      >
        {t("fin.cashflow.posting", "Posting")}
      </Button>
      <Button
        loading={pendingStatus === "DRAFT"}
        disabled={isPending}
        onClick={onDraft}
      >
        {t("fin.cashflow.draft", "Draft")}
      </Button>
      <Button disabled={isPending} onClick={onCancel}>
        {t("common.cancel", "Cancel")}
      </Button>
    </div>
  );
}

export function buildCashflowPayload(
  values: CashflowFormValues,
  status: CashflowStatus,
  attachmentFileIds: number[] = [],
): components["schemas"]["model.SaveCashFlowRequest"] {
  const txDate = values.transactionDate
    ? typeof (values.transactionDate as unknown as Dayjs).format === "function"
      ? (values.transactionDate as unknown as Dayjs).format("YYYY-MM-DD")
      : String(values.transactionDate)
    : dayjs().format("YYYY-MM-DD");

  const items = (values.items || []).map((item) => ({
    accountId: item.accountId as number,
    description: item.description || undefined,
    quantity: String(item.quantity ?? 1),
    unitPrice: String(item.unitPrice ?? 0),
  }));

  return {
    transactionDate: txDate,
    paymentAccountId: values.paymentAccountId as number,
    currency: values.currency || "IDR",
    exchangeRate: String(values.exchangeRate ?? 1),
    reference: values.reference || undefined,
    description: values.description || undefined,
    status,
    attachmentFileIds:
      attachmentFileIds.length > 0 ? attachmentFileIds : undefined,
    items,
  };
}

export interface SubmitCashflowFormArgs {
  wid: number;
  flow: CashflowFlow;
  documentId?: number;
  status: CashflowStatus;
  values: CashflowFormValues;
  pendingFiles: File[];
  queryClient: Pick<QueryClient, "invalidateQueries">;
  navigate: (to: string) => void;
}

export async function submitCashflowForm({
  wid,
  flow,
  documentId,
  status,
  values,
  pendingFiles,
  queryClient,
  navigate,
}: SubmitCashflowFormArgs) {
  let attachmentFileIds: number[] = [];
  if (pendingFiles.length > 0) {
    attachmentFileIds = await Promise.all(
      pendingFiles.map((file) => uploadWorkspaceFile(wid, file)),
    );
  }

  const payload = buildCashflowPayload(values, status, attachmentFileIds);
  const basePath =
    flow === "INCOME"
      ? "/workspaces/{wid}/fin/income"
      : "/workspaces/{wid}/fin/expense";

  if (documentId) {
    await unwrapOne(
      api.PUT(
        `${basePath}/{id}` as never,
        {
          params: { path: { wid, id: documentId as number } },
          body: payload,
        } as never,
      ),
    );
  } else {
    await unwrapOne(
      api.POST(
        basePath as never,
        {
          params: { path: { wid } },
          body: payload,
        } as never,
      ),
    );
  }

  if (flow === "INCOME") {
    await queryClient.invalidateQueries({ queryKey: incomeKeys.all(wid) });
  } else {
    await queryClient.invalidateQueries({ queryKey: expenseKeys.all(wid) });
  }
  await queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) });
  await queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) });

  navigate(flow === "INCOME" ? "/fin/income" : "/fin/expense");
}

export interface CashflowFormProps {
  wid: number;
  flow: "INCOME" | "EXPENSE";
  documentId?: number;
  transactionId?: number;
  initialValues?:
    | Partial<CashflowFormValues>
    | components["schemas"]["model.CashFlowItemResponse"];
}

export function CashflowForm({
  wid,
  flow,
  documentId,
  transactionId,
  initialValues,
}: CashflowFormProps) {
  const activeDocumentId = documentId ?? transactionId;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [form] = Form.useForm<CashflowFormValues>();
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);
  const [pendingStatus, setPendingStatus] = useState<CashflowStatus | null>(
    null,
  );

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );

  const isPending = pendingStatus !== null;

  const paymentGroups = useMemo(
    () => groupPaymentAccounts(accounts),
    [accounts],
  );

  const targetCategoryType = flow === "INCOME" ? "REVENUE" : "EXPENSE";
  const itemCategoryOptions = useMemo(() => {
    return accounts
      .filter(
        (a) =>
          a.isActive !== false && a.isPostable && a.type === targetCategoryType,
      )
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
        account: a,
      }));
  }, [accounts, targetCategoryType]);

  const watchedPaymentId = Form.useWatch("paymentAccountId", form);
  const watchedCurrency = Form.useWatch("currency", form) ?? "IDR";
  const watchedItems = Form.useWatch("items", form);

  const effectiveItems = useMemo(() => {
    if (watchedItems && watchedItems.length > 0) return watchedItems;
    if (initialValues?.items && initialValues.items.length > 0) {
      return initialValues.items;
    }
    return [{ quantity: 1, unitPrice: 0 }];
  }, [watchedItems, initialValues]);

  const selectedPaymentAccount = useMemo(
    () => accounts.find((a) => a.id === watchedPaymentId),
    [accounts, watchedPaymentId],
  );

  const itemLines = useMemo(() => {
    return effectiveItems.map((item, index) => {
      const accountId =
        "accountId" in item
          ? item.accountId
          : "account" in item
            ? item.account?.id
            : undefined;
      const acc = accounts.find((a) => a.id === accountId);
      const qty = Number(item?.quantity ?? 0);
      const unitPrice = Number(item?.unitPrice ?? 0);
      const total = qty * unitPrice;
      return {
        id: `line-${index}-${accountId ?? "none"}`,
        account: acc,
        description: item?.description,
        qty,
        unitPrice,
        total,
      };
    });
  }, [effectiveItems, accounts]);

  const grandTotal = useMemo(() => {
    return itemLines.reduce((acc, curr) => acc + curr.total, 0);
  }, [itemLines]);

  const handleSubmit = async (status: CashflowStatus) => {
    try {
      const values = await form.validateFields();
      setPendingStatus(status);
      try {
        await submitCashflowForm({
          wid,
          flow,
          documentId: activeDocumentId,
          status,
          values,
          pendingFiles,
          queryClient,
          navigate,
        });
        message.success(
          status === "POSTED"
            ? flow === "INCOME"
              ? t("fin.income.createSuccess", "Income recorded successfully")
              : t("fin.expense.createSuccess", "Expense recorded successfully")
            : flow === "INCOME"
              ? t("fin.income.draftSuccess", "Income draft saved successfully")
              : t(
                  "fin.expense.draftSuccess",
                  "Expense draft saved successfully",
                ),
        );
      } finally {
        setPendingStatus(null);
      }
    } catch (err: unknown) {
      if (err && typeof err === "object" && "errorFields" in err) {
        return;
      }
      const msg = err instanceof Error ? err.message : t("common.error");
      message.error(msg);
    }
  };

  const pageTitle =
    flow === "INCOME"
      ? documentId
        ? t("fin.income.edit", "Edit Income")
        : t("fin.income.new", "New Income")
      : documentId
        ? t("fin.expense.edit", "Edit Expense")
        : t("fin.expense.new", "New Expense");

  const normalizedInitialValues = useMemo(() => {
    if (!initialValues) {
      return {
        transactionDate: dayjs(),
        currency: "IDR",
        exchangeRate: 1,
        items: [{ quantity: 1, unitPrice: 0 }],
      };
    }
    const raw = initialValues as Record<string, unknown>;
    const txDate = raw.transactionDate
      ? typeof (raw.transactionDate as Dayjs).isValid === "function"
        ? (raw.transactionDate as Dayjs)
        : dayjs(raw.transactionDate as string)
      : dayjs();

    const paymentAccountId =
      (raw.paymentAccountId as number | undefined) ??
      (raw.paymentAccount as { id?: number } | undefined)?.id;

    const rawItems =
      (raw.items as Array<Record<string, unknown>> | undefined) ?? [];
    const items =
      rawItems.length > 0
        ? rawItems.map((it) => ({
            accountId:
              (it.accountId as number | undefined) ??
              (it.account as { id?: number } | undefined)?.id,
            description: it.description as string | undefined,
            quantity: it.quantity !== undefined ? Number(it.quantity) : 1,
            unitPrice: it.unitPrice !== undefined ? Number(it.unitPrice) : 0,
          }))
        : [{ quantity: 1, unitPrice: 0 }];

    return {
      transactionDate: txDate,
      paymentAccountId,
      currency: (raw.currency as string | undefined) || "IDR",
      exchangeRate:
        raw.exchangeRate !== undefined ? Number(raw.exchangeRate) : 1,
      reference: (raw.reference as string | undefined) ?? undefined,
      description: (raw.description as string | undefined) ?? undefined,
      items,
    };
  }, [initialValues]);

  return (
    <Page
      title={pageTitle}
      trail={[
        {
          title:
            flow === "INCOME"
              ? t("fin.menu.income", "Income")
              : t("fin.menu.expense", "Expense"),
          onClick: () =>
            navigate(flow === "INCOME" ? "/fin/income" : "/fin/expense"),
        },
      ]}
    >
      <FormView
        title={pageTitle}
        buttons={
          <CashflowActions
            pendingStatus={pendingStatus}
            onPosting={() => void handleSubmit("POSTED")}
            onDraft={() => void handleSubmit("DRAFT")}
            onCancel={() =>
              navigate(flow === "INCOME" ? "/fin/income" : "/fin/expense")
            }
          />
        }
      >
        <Form<CashflowFormValues>
          form={form}
          layout="vertical"
          size="small"
          initialValues={normalizedInitialValues}
          className="w-full max-w-5xl"
        >
          <div className="grid grid-cols-1 md:grid-cols-3 gap-x-4">
            <Form.Item
              name="paymentAccountId"
              label={
                flow === "INCOME"
                  ? t("fin.income.depositAccount", "Deposit Account")
                  : t("fin.expense.paymentAccount", "Payment Account")
              }
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Select
                showSearch
                placeholder={
                  flow === "INCOME"
                    ? t(
                        "fin.income.depositAccountPlaceholder",
                        "Select deposit account",
                      )
                    : t(
                        "fin.expense.paymentAccountPlaceholder",
                        "Select payment account",
                      )
                }
                optionFilterProp="label"
                options={paymentGroups}
              />
            </Form.Item>

            <Form.Item
              name="transactionDate"
              label={
                flow === "INCOME"
                  ? t("fin.income.date", "Date")
                  : t("fin.expense.date", "Date")
              }
              rules={[{ required: true, message: t("common.required") }]}
            >
              <DatePicker className="w-full" />
            </Form.Item>

            <Form.Item
              name="currency"
              label={t("fin.cashflow.currency", "Currency")}
            >
              <CurrencySelect currencies={currencies} />
            </Form.Item>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <Form.Item
              name="reference"
              label={
                flow === "INCOME"
                  ? t("fin.income.invoice", "Invoice #")
                  : t("fin.expense.reference", "Reference / Receipt #")
              }
            >
              <Input
                placeholder={
                  flow === "INCOME"
                    ? t("fin.income.invoicePlaceholder", "e.g. INV-2026-001")
                    : t(
                        "fin.expense.referencePlaceholder",
                        "e.g. RCPT-2026-001",
                      )
                }
              />
            </Form.Item>

            <Form.Item
              name="description"
              label={
                flow === "INCOME"
                  ? t("fin.income.description", "Description")
                  : t("fin.expense.description", "Description")
              }
            >
              <Input
                placeholder={t(
                  "fin.cashflow.descriptionPlaceholder",
                  "e.g. Services, supplies, etc.",
                )}
              />
            </Form.Item>
          </div>

          {/* Items Section */}
          <div className="mt-4 mb-2">
            <div className="text-sm font-semibold mb-2">
              {t("fin.cashflow.items", "Items")}
            </div>
            <Form.List name="items">
              {(fields, { add, remove }) => (
                <div className="flex flex-col gap-2">
                  <div className="border rounded overflow-hidden">
                    <table className="w-full text-left text-sm">
                      <thead
                        className="border-b text-xs uppercase"
                        style={{
                          backgroundColor: "var(--omed-surface-muted)",
                          color: "var(--omed-text-secondary)",
                        }}
                      >
                        <tr>
                          <th className="p-2 w-1/3">
                            {flow === "INCOME"
                              ? t(
                                  "fin.income.revenueCategory",
                                  "Revenue Category",
                                )
                              : t(
                                  "fin.expense.expenseCategory",
                                  "Expense Category",
                                )}
                          </th>
                          <th className="p-2">
                            {t("fin.cashflow.itemDescription", "Description")}
                          </th>
                          <th className="p-2 w-24">
                            {t("fin.cashflow.quantity", "Qty")}
                          </th>
                          <th className="p-2 w-36">
                            {t("fin.cashflow.unitPrice", "Unit Price")}
                          </th>
                          <th className="p-2 w-36 text-right">
                            {t("fin.cashflow.lineTotal", "Total")}
                          </th>
                          <th className="p-2 w-12 text-center"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {fields.map(({ key, name, ...restField }, idx) => {
                          const item = effectiveItems[idx];
                          const qty = Number(item?.quantity ?? 0);
                          const unitPrice = Number(item?.unitPrice ?? 0);
                          const lineTotal = qty * unitPrice;

                          return (
                            <tr key={key} className="border-b last:border-b-0">
                              <td className="p-2 align-top">
                                <Form.Item
                                  {...restField}
                                  name={[name, "accountId"]}
                                  rules={[
                                    {
                                      required: true,
                                      message: t("common.required"),
                                    },
                                  ]}
                                  className="mb-0"
                                >
                                  <Select
                                    showSearch
                                    placeholder={t(
                                      "fin.cashflow.itemAccountPlaceholder",
                                      "Select account",
                                    )}
                                    optionFilterProp="label"
                                    options={itemCategoryOptions}
                                  />
                                </Form.Item>
                              </td>
                              <td className="p-2 align-top">
                                <Form.Item
                                  {...restField}
                                  name={[name, "description"]}
                                  className="mb-0"
                                >
                                  <Input
                                    placeholder={t(
                                      "fin.cashflow.itemDescription",
                                      "Description",
                                    )}
                                  />
                                </Form.Item>
                              </td>
                              <td className="p-2 align-top">
                                <Form.Item
                                  {...restField}
                                  name={[name, "quantity"]}
                                  rules={[
                                    {
                                      required: true,
                                      message: t("common.required"),
                                    },
                                  ]}
                                  className="mb-0"
                                >
                                  <InputNumber min={1} className="w-full" />
                                </Form.Item>
                              </td>
                              <td className="p-2 align-top">
                                <Form.Item
                                  {...restField}
                                  name={[name, "unitPrice"]}
                                  rules={[
                                    {
                                      required: true,
                                      message: t("common.required"),
                                    },
                                  ]}
                                  className="mb-0"
                                >
                                  <MoneyInput
                                    currency={watchedCurrency}
                                    className="w-full"
                                  />
                                </Form.Item>
                              </td>
                              <td className="p-2 align-middle text-right font-mono text-sm">
                                {watchedCurrency} {lineTotal.toLocaleString()}
                              </td>
                              <td className="p-2 align-middle text-center">
                                <Button
                                  type="text"
                                  danger
                                  disabled={fields.length <= 1 || isPending}
                                  icon={<DeleteOutlined />}
                                  onClick={() => remove(name)}
                                />
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                  <div>
                    <Button
                      type="dashed"
                      icon={<PlusOutlined />}
                      disabled={isPending}
                      onClick={() => add({ quantity: 1, unitPrice: 0 })}
                    >
                      {t("fin.cashflow.addItem", "Add item")}
                    </Button>
                  </div>
                </div>
              )}
            </Form.List>
          </div>

          {/* Document Grand Total */}
          <div className="flex justify-end items-center gap-2 p-3 my-2 text-base font-semibold border-t">
            <span>{t("fin.cashflow.total", "Total")}:</span>
            <span className="font-mono">
              {watchedCurrency} {grandTotal.toLocaleString()}
            </span>
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
              {flow === "INCOME" ? (
                <>
                  {/* Debit payment account */}
                  <div className="flex items-center justify-between">
                    <span>
                      <strong className="text-emerald-600">(D) Payment:</strong>{" "}
                      {selectedPaymentAccount
                        ? `${selectedPaymentAccount.code} - ${selectedPaymentAccount.name}`
                        : t("fin.cashflow.paymentAccount", "Payment")}
                    </span>
                    <span>
                      {grandTotal > 0
                        ? `${watchedCurrency} ${grandTotal.toLocaleString()}`
                        : "-"}
                    </span>
                  </div>
                  {/* Credit revenue items */}
                  {itemLines.map((line) => (
                    <div
                      key={`rev-${line.id}`}
                      className="flex items-center justify-between pl-4"
                    >
                      <span>
                        <strong className="text-blue-600">(K) Revenue:</strong>{" "}
                        {line.account
                          ? `${line.account.code} - ${line.account.name}`
                          : t("fin.types.REVENUE", "Revenue")}
                        {line.description ? ` (${line.description})` : ""}
                      </span>
                      <span>
                        {line.total > 0
                          ? `${watchedCurrency} ${line.total.toLocaleString()}`
                          : "-"}
                      </span>
                    </div>
                  ))}
                </>
              ) : (
                <>
                  {/* Debit expense items */}
                  {itemLines.map((line) => (
                    <div
                      key={`exp-${line.id}`}
                      className="flex items-center justify-between"
                    >
                      <span>
                        <strong className="text-emerald-600">
                          (D) Expense:
                        </strong>{" "}
                        {line.account
                          ? `${line.account.code} - ${line.account.name}`
                          : t("fin.types.EXPENSE", "Expense")}
                        {line.description ? ` (${line.description})` : ""}
                      </span>
                      <span>
                        {line.total > 0
                          ? `${watchedCurrency} ${line.total.toLocaleString()}`
                          : "-"}
                      </span>
                    </div>
                  ))}
                  {/* Credit payment account */}
                  <div className="flex items-center justify-between pl-4">
                    <span>
                      <strong className="text-blue-600">(K) Payment:</strong>{" "}
                      {selectedPaymentAccount
                        ? `${selectedPaymentAccount.code} - ${selectedPaymentAccount.name}`
                        : t("fin.cashflow.paymentAccount", "Payment")}
                    </span>
                    <span>
                      {grandTotal > 0
                        ? `${watchedCurrency} ${grandTotal.toLocaleString()}`
                        : "-"}
                    </span>
                  </div>
                </>
              )}
            </div>
          </div>

          {/* Receipt / Attachment Picker */}
          <div className="mt-4">
            <PendingAttachmentsPicker
              files={pendingFiles}
              onChange={setPendingFiles}
              disabled={isPending}
            />
          </div>
        </Form>
      </FormView>
    </Page>
  );
}
