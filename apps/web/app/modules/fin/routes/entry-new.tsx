import {
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import {
  Alert,
  Button,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useSearchParams } from "react-router";
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
  PeriodSelect,
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
  transactionsInfiniteQueryOptions,
} from "../queries";
import type { Route } from "./+types/entry-new";

export type CreateEntryFormValues = {
  transactionMode: "new" | "existing";
  transactionId?: number;
  periodId?: number;
  date: Dayjs;
  description?: string;
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
    queryClient.ensureInfiniteQueryData(
      transactionsInfiniteQueryOptions(wid, {}),
    ),
  ]);
  return { wid };
}

export default function EntryNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canCreateEntry } = useFinPermissions(wid);

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));
  const { data: txPages } = useSuspenseInfiniteQuery(
    transactionsInfiniteQueryOptions(wid, {}),
  );

  const existingTransactions = useMemo(() => {
    return txPages?.pages.flatMap((page) => page.items) ?? [];
  }, [txPages]);

  const queryTxId = searchParams.get("transactionId")
    ? Number(searchParams.get("transactionId"))
    : undefined;

  const [form] = Form.useForm<CreateEntryFormValues>();
  const [rateNotFound, setRateNotFound] = useState(false);

  const mode = Form.useWatch("transactionMode", form) ?? "new";
  const selectedTxId = Form.useWatch("transactionId", form);
  const selectedCurrency = Form.useWatch("currency", form) ?? "IDR";
  const selectedRate = Form.useWatch("exchangeRate", form) ?? 1;
  const selectedDate = Form.useWatch("date", form);

  // When an existing transaction is selected, inherit its currency & exchange rate
  useEffect(() => {
    if (mode === "existing" && selectedTxId) {
      const tx = existingTransactions.find((item) => item.id === selectedTxId);
      if (tx) {
        if (tx.currency) {
          form.setFieldValue("currency", tx.currency);
        }
        if (tx.exchangeRate) {
          form.setFieldValue("exchangeRate", tx.exchangeRate);
        }
      }
    }
  }, [mode, selectedTxId, existingTransactions, form]);

  // Lookup exchange rate when in "new" transaction mode
  useEffect(() => {
    if (mode === "existing") return;

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
  }, [mode, selectedCurrency, selectedDate, wid, form]);

  // Auto-set matching open period
  useEffect(() => {
    if (!form.getFieldValue("periodId")) {
      const selected = form.getFieldValue("date") ?? dayjs();
      const dateStr =
        typeof selected.format === "function"
          ? selected.format("YYYY-MM-DD")
          : String(selected).slice(0, 10);
      const matched = periods.find((p) => {
        if (p.status !== "OPEN") return false;
        const start = (p.startDate ?? "").slice(0, 10);
        const end = (p.endDate ?? "9999-12-31").slice(0, 10);
        return dateStr >= start && dateStr <= end;
      });
      if (matched?.id) {
        form.setFieldValue("periodId", matched.id);
      } else if (periods.length > 0) {
        const firstOpen =
          periods.find((p) => p.status === "OPEN") ?? periods[0];
        if (firstOpen?.id) {
          form.setFieldValue("periodId", firstOpen.id);
        }
      }
    }
  }, [periods, form]);

  const handleSubmit = async (values: CreateEntryFormValues) => {
    const entryDate = values.date
      ? typeof values.date.format === "function"
        ? values.date.format("YYYY-MM-DD")
        : String(values.date)
      : dayjs().format("YYYY-MM-DD");

    const currency = values.currency || "IDR";
    const exchangeRate = values.exchangeRate || 1;

    const summary = calculatePostingsSummary(
      values.postings,
      currency,
      exchangeRate,
    );
    if (!summary.balanced) {
      return;
    }

    const postings = postingLinesToApi(
      values.postings,
      exchangeRate,
      currency,
      values.description,
    );
    if (!values.periodId) {
      return;
    }

    const ok = await run(async () => {
      if (values.transactionMode === "new") {
        // Create new transaction with inline entry
        await unwrapOne(
          api.POST("/workspaces/{wid}/fin/transactions", {
            params: { path: { wid } },
            body: {
              transactionDate: entryDate,
              description: values.description,
              currency,
              exchangeRate: exchangeRate.toString(),
              entry: {
                periodId: values.periodId as number,
                entryDate,
                description: values.description,
                postings,
              },
            },
          }),
        );
      } else {
        // Create entry inside existing transaction
        await unwrapOne(
          api.POST("/workspaces/{wid}/fin/entries", {
            params: { path: { wid } },
            body: {
              transactionId: values.transactionId,
              periodId: values.periodId as number,
              entryDate,
              description: values.description,
              postings,
            },
          }),
        );
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) }),
        queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) }),
      ]);
    }, "common.saved");

    if (ok) {
      if (queryTxId) {
        navigate(`/fin/transactions/${queryTxId}`);
      } else {
        navigate("/fin/journal");
      }
    }
  };

  const actionButtons = (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        type="primary"
        loading={pending}
        disabled={!canCreateEntry}
        onClick={() => form.submit()}
      >
        {t("common.save", "Save")}
      </Button>
      <Button onClick={() => navigate("/fin/journal")}>
        {t("common.cancel", "Cancel")}
      </Button>
    </div>
  );

  return (
    <Page
      title={t("fin.journal.new", "New entry")}
      trail={[
        {
          title: t("fin.menu.journal", "Journal Entries"),
          onClick: () => navigate("/fin/journal"),
        },
      ]}
    >
      <FormView
        title={t("fin.journal.new", "New entry")}
        buttons={actionButtons}
      >
        <Form
          form={form}
          layout="vertical"
          size="small"
          initialValues={{
            transactionMode: queryTxId ? "existing" : "new",
            transactionId: queryTxId,
            periodId: undefined,
            date: dayjs(),
            currency: "IDR",
            exchangeRate: 1,
            postings: [
              {
                accountId: undefined,
                debit: undefined,
                credit: undefined,
                description: "",
              },
              {
                accountId: undefined,
                debit: undefined,
                credit: undefined,
                description: "",
              },
            ],
          }}
          onFinish={handleSubmit}
        >
          <div className="flex flex-col gap-4">
            <Form.Item
              name="transactionMode"
              label={t("fin.journal.transactionMode", "Transaction mode")}
              rules={[{ required: true }]}
            >
              <Radio.Group
                buttonStyle="solid"
                className="w-full grid grid-cols-2"
              >
                <Radio.Button value="new" className="text-center">
                  {t(
                    "fin.journal.createNewTransaction",
                    "Create new transaction",
                  )}
                </Radio.Button>
                <Radio.Button value="existing" className="text-center">
                  {t(
                    "fin.journal.selectExistingTransaction",
                    "Select existing",
                  )}
                </Radio.Button>
              </Radio.Group>
            </Form.Item>

            {mode === "existing" && (
              <Form.Item
                name="transactionId"
                label={t("fin.journal.transaction", "Transaction")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <Select
                  showSearch
                  placeholder={t(
                    "fin.journal.selectExistingTransaction",
                    "Select transaction",
                  )}
                  optionFilterProp="label"
                  options={existingTransactions.map((tx) => ({
                    value: tx.id as number,
                    label: `${tx.number} - ${tx.description || tx.reference || "No description"}`,
                  }))}
                />
              </Form.Item>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <Form.Item
                name="date"
                label={t("common.date", "Date")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <DatePicker
                  className="w-full"
                  onChange={(d) => {
                    if (d) {
                      const dateStr = d.format("YYYY-MM-DD");
                      const matched = periods.find((p) => {
                        if (p.status !== "OPEN") return false;
                        const start = p.startDate ?? "";
                        const end = p.endDate ?? "9999-12-31";
                        return dateStr >= start && dateStr <= end;
                      });
                      if (matched?.id) {
                        form.setFieldValue("periodId", matched.id);
                      }
                    }
                  }}
                />
              </Form.Item>

              <Form.Item
                name="periodId"
                label={t("fin.journal.period", "Period")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <PeriodSelect periods={periods} />
              </Form.Item>
            </div>

            <Form.Item
              name="description"
              label={t("fin.journal.label", "Description")}
            >
              <Input placeholder={t("fin.journal.label", "Description")} />
            </Form.Item>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <Form.Item
                name="currency"
                label={t("fin.transactions.currency", "Currency")}
                rules={[
                  { required: true, message: t("common.required", "Required") },
                ]}
              >
                <CurrencySelect
                  currencies={currencies}
                  disabled={mode === "existing"}
                />
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
                  disabled={mode === "existing" || selectedCurrency === "IDR"}
                />
              </Form.Item>
            </div>

            {mode === "new" && rateNotFound && selectedCurrency !== "IDR" && (
              <Alert
                type="warning"
                showIcon
                title={t(
                  "fin.transactions.rateNotFound",
                  "No exchange rate found for this date. Enter rate manually.",
                )}
                className="mb-2"
              />
            )}

            <div className="pt-2">
              <span className="text-sm font-semibold mb-2 block">
                {t("fin.journal.account", "Postings")}
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
