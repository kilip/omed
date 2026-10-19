import type { components } from "@omed/openapi";
import {
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";
import {
  DatePicker,
  Form,
  type FormInstance,
  Input,
  Radio,
  Select,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import {
  calculatePostingsSummary,
  PeriodSelect,
  type PostingLineValue,
  PostingsEditor,
  postingLinesToApi,
} from "../components";
import {
  accountsQueryOptions,
  entryKeys,
  periodsQueryOptions,
  transactionKeys,
  transactionsInfiniteQueryOptions,
} from "../queries";

export type CreateEntryFormValues = {
  transactionMode: "new" | "existing";
  transactionId?: number;
  periodId?: number;
  date: Dayjs;
  description?: string;
  postings: PostingLineValue[];
};

export type CreateEntryDrawerProps = {
  wid: number;
  open: boolean;
  defaultTransactionId?: number;
  onClose: () => void;
  onSuccess?: (entryId: number) => void;
};

type EntryDrawerFormFieldsProps = {
  form: FormInstance<CreateEntryFormValues>;
  accounts: components["schemas"]["model.AccountResponse"][];
  periods: components["schemas"]["model.PeriodResponse"][];
  existingTransactions: components["schemas"]["model.TransactionResponse"][];
};

function EntryDrawerFormFields({
  form,
  accounts,
  periods,
  existingTransactions,
}: EntryDrawerFormFieldsProps) {
  const { t } = useTranslation();
  const mode = Form.useWatch("transactionMode", form) ?? "new";

  return (
    <div className="flex flex-col gap-4">
      <Form.Item
        name="transactionMode"
        label={t("fin.journal.transactionMode", "Transaction mode")}
        rules={[{ required: true }]}
      >
        <Radio.Group buttonStyle="solid" className="w-full grid grid-cols-2">
          <Radio.Button value="new" className="text-center">
            {t("fin.journal.createNewTransaction", "Create new transaction")}
          </Radio.Button>
          <Radio.Button value="existing" className="text-center">
            {t("fin.journal.selectExistingTransaction", "Select existing")}
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

      <div className="pt-2">
        <span className="text-sm font-semibold mb-2 block">
          {t("fin.journal.account", "Postings")}
        </span>
        <PostingsEditor accounts={accounts} currency="IDR" exchangeRate={1} />
      </div>
    </div>
  );
}

export function CreateEntryDrawer({
  wid,
  open,
  defaultTransactionId,
  onClose,
  onSuccess,
}: CreateEntryDrawerProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));

  // Fetch recent transactions for "Select existing" mode
  const { data: txPages } = useSuspenseInfiniteQuery(
    transactionsInfiniteQueryOptions(wid, {}),
  );

  const existingTransactions = useMemo(() => {
    return txPages?.pages.flatMap((page) => page.items) ?? [];
  }, [txPages]);

  const initialValues: Partial<CreateEntryFormValues> = {
    transactionMode: defaultTransactionId ? "existing" : "new",
    transactionId: defaultTransactionId,
    date: dayjs(),
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
  };

  const handleSubmit = async (
    values: CreateEntryFormValues,
  ): Promise<boolean> => {
    const entryDate = values.date.format("YYYY-MM-DD");
    const summary = calculatePostingsSummary(values.postings, "IDR", 1);
    if (!summary.balanced) {
      return false;
    }

    const postings = postingLinesToApi(values.postings, 1, "IDR");
    if (!values.periodId) {
      return false;
    }

    let createdEntryId: number | undefined;

    const ok = await run(async () => {
      if (values.transactionMode === "new") {
        // Create new transaction with inline entry
        const res = await unwrapOne(
          api.POST("/workspaces/{wid}/fin/transactions", {
            params: { path: { wid } },
            body: {
              transactionDate: entryDate,
              description: values.description,
              currency: "IDR",
              exchangeRate: 1,
              entry: {
                periodId: values.periodId as number,
                entryDate,
                description: values.description,
                postings,
              },
            },
          }),
        );
        createdEntryId = res.entries?.[0]?.id;
      } else {
        // Create entry inside existing transaction
        const res = await unwrapOne(
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
        createdEntryId = res.id;
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) }),
        queryClient.invalidateQueries({ queryKey: transactionKeys.all(wid) }),
      ]);
    }, "common.saved");

    if (ok && createdEntryId && onSuccess) {
      onSuccess(createdEntryId);
    }

    return ok;
  };

  return (
    <RecordModal<CreateEntryFormValues>
      variant="drawer"
      open={open}
      title={t("fin.journal.new", "New entry")}
      initialValues={initialValues}
      submitting={pending}
      onClose={onClose}
      onSubmit={handleSubmit}
    >
      {(form) => (
        <EntryDrawerFormFields
          form={form}
          accounts={accounts}
          periods={periods}
          existingTransactions={existingTransactions}
        />
      )}
    </RecordModal>
  );
}
