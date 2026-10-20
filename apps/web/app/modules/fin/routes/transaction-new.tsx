import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Alert, Button, DatePicker, Form, Input, InputNumber } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { api, unwrapOne } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import {
  CurrencySelect,
  calculatePostingsSummary,
  type PostingLineValue,
  PostingsEditor,
  postingLinesToApi,
} from "../components";
import { useFinPermissions } from "../hooks";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  entryKeys,
  periodsQueryOptions,
  transactionKeys,
} from "../queries";
import type { Route } from "./+types/transaction-new";

export type CreateTransactionFormValues = {
  date: Dayjs;
  description?: string;
  reference?: string;
  currency: string;
  exchangeRate: number;
  postings: PostingLineValue[];
};

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(currenciesQueryOptions(wid)),
    queryClient.ensureQueryData(periodsQueryOptions(wid)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function TransactionNewRoute({
  loaderData,
}: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canCreateTransaction } = useFinPermissions(wid);

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));

  const [form] = Form.useForm<CreateTransactionFormValues>();
  const [rateNotFound, setRateNotFound] = useState(false);

  const selectedCurrency = Form.useWatch("currency", form) ?? "IDR";
  const selectedRate = Form.useWatch("exchangeRate", form) ?? 1;
  const selectedDate = Form.useWatch("date", form);

  useEffect(() => {
    if (!selectedCurrency || selectedCurrency === "IDR") {
      form.setFieldValue("exchangeRate", 1);
      setRateNotFound(false);
      return;
    }

    const dateStr = selectedDate
      ? selectedDate.format("YYYY-MM-DD")
      : dayjs().format("YYYY-MM-DD");

    let isMounted = true;
    api
      .GET("/workspaces/{wid}/fin/exchange-rates/lookup", {
        params: {
          path: { wid },
          query: { from: selectedCurrency, to: "IDR", date: dateStr },
        },
      })
      .then((res) => {
        if (!isMounted) return;
        if (res.data?.data?.rate) {
          form.setFieldValue("exchangeRate", res.data.data.rate);
          setRateNotFound(false);
        } else {
          setRateNotFound(true);
        }
      })
      .catch(() => {
        if (!isMounted) return;
        setRateNotFound(true);
      });

    return () => {
      isMounted = false;
    };
  }, [selectedCurrency, selectedDate, wid, form]);

  const handleSubmit = async (values: CreateTransactionFormValues) => {
    const txDate = values.date
      ? typeof values.date.format === "function"
        ? values.date.format("YYYY-MM-DD")
        : String(values.date)
      : dayjs().format("YYYY-MM-DD");
    const currency = values.currency || "IDR";
    const exchangeRate = values.exchangeRate || 1;

    // Check if initial entry is provided
    const validLines = (values.postings ?? []).filter(
      (p) => p.accountId && ((p.debit ?? 0) > 0 || (p.credit ?? 0) > 0),
    );

    let inlineEntry:
      | components["schemas"]["model.CreateEntryInlineRequest"]
      | undefined;

    if (validLines.length > 0) {
      const summary = calculatePostingsSummary(
        values.postings,
        currency,
        exchangeRate,
      );
      if (!summary.balanced) {
        return;
      }

      // Auto-find period for the transaction date
      const matched = periods.find((p) => {
        if (p.status !== "OPEN") return false;
        const start = p.startDate ?? "";
        const end = p.endDate ?? "9999-12-31";
        return txDate >= start && txDate <= end;
      });
      const periodId =
        matched?.id ?? periods.find((p) => p.status === "OPEN")?.id;

      if (!periodId) {
        return;
      }

      inlineEntry = {
        periodId,
        entryDate: txDate,
        description: values.description,
        postings: postingLinesToApi(values.postings, exchangeRate, currency),
      };
    }

    let createdId: number | undefined;

    const ok = await run(async () => {
      const res = await unwrapOne(
        api.POST("/workspaces/{wid}/fin/transactions", {
          params: { path: { wid } },
          body: {
            transactionDate: txDate,
            description: values.description,
            reference: values.reference,
            currency,
            exchangeRate,
            entry: inlineEntry,
          },
        }),
      );
      createdId = res.id;

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) }),
        queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) }),
      ]);
    }, "common.saved");

    if (ok && createdId) {
      navigate(`/fin/transactions/${createdId}`);
    }
  };

  const actionButtons = (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        type="primary"
        loading={pending}
        disabled={!canCreateTransaction}
        onClick={() => form.submit()}
      >
        {t("common.save", "Save")}
      </Button>
      <Button onClick={() => navigate("/fin/transactions")}>
        {t("common.cancel", "Cancel")}
      </Button>
    </div>
  );

  return (
    <Page
      title={t("fin.transactions.new", "New transaction")}
      trail={[
        {
          title: t("fin.menu.transactions", "Transactions"),
          onClick: () => navigate("/fin/transactions"),
        },
      ]}
    >
      <FormView
        title={t("fin.transactions.new", "New transaction")}
        buttons={actionButtons}
      >
        <Form
          form={form}
          layout="vertical"
          size="small"
          initialValues={{
            date: dayjs(),
            currency: "IDR",
            exchangeRate: 1,
            postings: [],
          }}
          onFinish={handleSubmit}
        >
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <Form.Item
                name="date"
                label={t("fin.transactions.date", "Date")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <DatePicker className="w-full" />
              </Form.Item>

              <Form.Item
                name="reference"
                label={t("fin.transactions.reference", "Reference")}
              >
                <Input
                  placeholder={t("fin.transactions.reference", "Reference")}
                />
              </Form.Item>
            </div>

            <Form.Item
              name="description"
              label={t("fin.transactions.description", "Description")}
            >
              <Input
                placeholder={t("fin.transactions.description", "Description")}
              />
            </Form.Item>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <Form.Item
                name="currency"
                label={t("fin.transactions.currency", "Currency")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <CurrencySelect currencies={currencies} />
              </Form.Item>

              <Form.Item
                name="exchangeRate"
                label={t("fin.transactions.exchangeRate", "Exchange rate")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <InputNumber
                  className="w-full"
                  min={0.000001}
                  step={0.01}
                  disabled={selectedCurrency === "IDR"}
                />
              </Form.Item>
            </div>

            {rateNotFound && selectedCurrency !== "IDR" && (
              <Alert
                type="warning"
                showIcon
                message={t(
                  "fin.transactions.rateNotFound",
                  "No exchange rate found for this date. Enter rate manually.",
                )}
                className="mb-2"
              />
            )}

            <div className="pt-2">
              <span className="text-sm font-semibold mb-2 block">
                {t("fin.transactions.initialEntry", "Initial entry (optional)")}
              </span>
              <PostingsEditor
                accounts={accounts}
                currency={selectedCurrency}
                exchangeRate={selectedRate}
              />
            </div>
          </div>
        </Form>
      </FormView>
    </Page>
  );
}
