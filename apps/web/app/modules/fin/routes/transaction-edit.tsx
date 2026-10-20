import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Alert, Button, Descriptions, Form, Input, Tag } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { api, unwrapOne } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import { useFinPermissions } from "../hooks";
import { transactionKeys, transactionQueryOptions } from "../queries";
import type { Route } from "./+types/transaction-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(transactionQueryOptions(wid, id)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id };
}

type EditFormValues = {
  description?: string;
  reference?: string;
};

export default function TransactionEditRoute({
  loaderData,
}: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditTransaction } = useFinPermissions(wid);

  const { data: transaction } = useSuspenseQuery(
    transactionQueryOptions(wid, id),
  );

  const [form] = Form.useForm<EditFormValues>();

  const entries = transaction.entries ?? [];
  const hasPostedEntries = entries.some((e) => e.status === "POSTED");

  const handleSubmit = async (values: EditFormValues) => {
    const ok = await run(
      () =>
        unwrapOne(
          api.PUT("/workspaces/{wid}/fin/transactions/{id}", {
            params: { path: { wid, id } },
            body: {
              description: values.description,
              reference: values.reference,
            },
          }),
        ),
      "common.saved",
    );

    if (ok) {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: transactionKeys.detail(wid, id),
        }),
        queryClient.invalidateQueries({
          queryKey: transactionKeys.all(wid),
        }),
      ]);
      navigate(`/fin/transactions/${id}`);
    }
  };

  const actionButtons = (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        type="primary"
        loading={pending}
        disabled={hasPostedEntries || !canEditTransaction}
        onClick={() => form.submit()}
      >
        {t("fin.transactions.saveChanges", "Save changes")}
      </Button>
      <Button onClick={() => navigate(`/fin/transactions/${id}`)}>
        {t("common.cancel", "Cancel")}
      </Button>
    </div>
  );

  return (
    <Page
      title={`${t("fin.transactions.edit", "Edit transaction")} - ${
        transaction.number || `#${transaction.id}`
      }`}
      trail={[
        {
          title: t("fin.menu.transactions", "Transactions"),
          onClick: () => navigate("/fin/transactions"),
        },
        {
          title: transaction.number || `#${transaction.id}`,
          onClick: () => navigate(`/fin/transactions/${id}`),
        },
      ]}
    >
      <FormView
        title={transaction.number || `#${transaction.id}`}
        buttons={actionButtons}
      >
        {hasPostedEntries && (
          <Alert
            type="warning"
            showIcon
            message={t(
              "fin.transactions.hasPostedEntries",
              "Cannot edit transaction because one or more entries are posted.",
            )}
            className="mb-4"
          />
        )}

        <div className="flex flex-col gap-4">
          <Descriptions size="small" bordered column={{ xs: 1, sm: 2, md: 3 }}>
            <Descriptions.Item label={t("common.date", "Date")}>
              {formatDate(transaction.transactionDate, i18n.language)}
            </Descriptions.Item>
            <Descriptions.Item label={t("fin.transactions.source", "Source")}>
              <Tag>{transaction.source}</Tag>
            </Descriptions.Item>
            <Descriptions.Item
              label={t("fin.transactions.currency", "Currency")}
            >
              {transaction.currency}
              {transaction.exchangeRate && transaction.currency !== "IDR"
                ? ` (@ ${transaction.exchangeRate})`
                : ""}
            </Descriptions.Item>
          </Descriptions>

          <Form
            form={form}
            layout="vertical"
            size="small"
            initialValues={{
              description: transaction.description,
              reference: transaction.reference,
            }}
            onFinish={handleSubmit}
            disabled={hasPostedEntries || !canEditTransaction}
          >
            <Form.Item
              name="reference"
              label={t("fin.transactions.reference", "Reference")}
            >
              <Input
                placeholder={t("fin.transactions.reference", "Reference")}
              />
            </Form.Item>

            <Form.Item
              name="description"
              label={t("fin.transactions.description", "Description")}
            >
              <Input
                placeholder={t("fin.transactions.description", "Description")}
              />
            </Form.Item>
          </Form>
        </div>
      </FormView>
    </Page>
  );
}
