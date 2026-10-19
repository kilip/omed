import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
  Alert,
  DatePicker,
  Form,
  type FormInstance,
  Input,
  InputNumber,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import {
  CurrencySelect,
  calculatePostingsSummary,
  type PostingLineValue,
  PostingsEditor,
  postingLinesToApi,
} from "../components";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  entryKeys,
  periodsQueryOptions,
  transactionKeys,
} from "../queries";

export type CreateTransactionFormValues = {
  date: Dayjs;
  description?: string;
  reference?: string;
  currency: string;
  exchangeRate: number;
  postings: PostingLineValue[];
};

export type CreateTransactionDrawerProps = {
  wid: number;
  open: boolean;
  onClose: () => void;
  onSuccess?: (transactionId: number) => void;
};

type DrawerFormFieldsProps = {
  form: FormInstance<CreateTransactionFormValues>;
  accounts: components["schemas"]["model.AccountResponse"][];
  currencies: components["schemas"]["model.CurrencyResponse"][];
  rateNotFound: boolean;
};

function CreateTransactionFormFields({
  form,
  accounts,
  currencies,
  rateNotFound,
  onRateNotFoundChange,
  wid,
}: DrawerFormFieldsProps & {
  onRateNotFoundChange: (val: boolean) => void;
  wid: number;
}) {
  const { t } = useTranslation();
  const selectedCurrency = Form.useWatch("currency", form) ?? "IDR";
  const selectedRate = Form.useWatch("exchangeRate", form) ?? 1;
  const selectedDate = Form.useWatch("date", form);

  useEffect(() => {
    if (!selectedCurrency || selectedCurrency === "IDR") {
      form.setFieldValue("exchangeRate", 1);
      onRateNotFoundChange(false);
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
          onRateNotFoundChange(false);
        } else {
          onRateNotFoundChange(true);
        }
      })
      .catch(() => {
        if (!isMounted) return;
        onRateNotFoundChange(true);
      });

    return () => {
      isMounted = false;
    };
  }, [selectedCurrency, selectedDate, wid, form, onRateNotFoundChange]);

  return (
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
          <Input placeholder={t("fin.transactions.reference", "Reference")} />
        </Form.Item>
      </div>

      <Form.Item
        name="description"
        label={t("fin.transactions.description", "Description")}
      >
        <Input placeholder={t("fin.transactions.description", "Description")} />
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
  );
}

export function CreateTransactionDrawer({
  wid,
  open,
  onClose,
  onSuccess,
}: CreateTransactionDrawerProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const [rateNotFound, setRateNotFound] = useState(false);

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: currencies = [] } = useSuspenseQuery(
    currenciesQueryOptions(wid),
  );
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));

  const initialValues: Partial<CreateTransactionFormValues> = {
    date: dayjs(),
    currency: "IDR",
    exchangeRate: 1,
    postings: [],
  };

  const handleSubmit = async (
    values: CreateTransactionFormValues,
  ): Promise<boolean> => {
    const txDate = values.date.format("YYYY-MM-DD");
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
        return false;
      }

      // Auto-find period for the transaction date
      const matched = periods.find((p) => {
        if (p.status !== "OPEN") return false;
        const start = p.startDate ?? "";
        const end = p.endDate ?? "9999-12-31";
        return txDate >= start && txDate <= end;
      });

      if (!matched?.id) {
        return false;
      }

      inlineEntry = {
        periodId: matched.id,
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

    if (ok && createdId && onSuccess) {
      onSuccess(createdId);
    }

    return ok;
  };

  return (
    <RecordModal<CreateTransactionFormValues>
      variant="drawer"
      open={open}
      title={t("fin.transactions.new", "New transaction")}
      initialValues={initialValues}
      submitting={pending}
      onClose={() => {
        setRateNotFound(false);
        onClose();
      }}
      onSubmit={handleSubmit}
    >
      {(form) => (
        <CreateTransactionFormFields
          form={form}
          accounts={accounts}
          currencies={currencies}
          rateNotFound={rateNotFound}
          onRateNotFoundChange={setRateNotFound}
          wid={wid}
        />
      )}
    </RecordModal>
  );
}
