import {
  ArrowLeftOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Button, Popconfirm, Table, type TableProps } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { api, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import {
  EntryStatusTag,
  MoneyCell,
  TransactionAttachmentsSection,
} from "../components";
import { useFinPermissions } from "../hooks";
import { transactionKeys, transactionQueryOptions } from "../queries";
import type { Route } from "./+types/transaction";

type Entry = components["schemas"]["model.EntryResponse"];

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

export default function TransactionRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditTransaction, canDeleteTransaction, canCreateEntry } =
    useFinPermissions(wid);

  const { data: transaction } = useSuspenseQuery(
    transactionQueryOptions(wid, id),
  );

  const entries = transaction.entries ?? [];
  const hasPostedEntries = entries.some((e) => e.status === "POSTED");

  const handleDelete = async () => {
    const ok = await run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/fin/transactions/{id}", {
            params: { path: { wid, id } },
          }),
        ),
      "common.deleted",
    );
    if (ok) {
      await queryClient.invalidateQueries({
        queryKey: transactionKeys.all(wid),
      });
      navigate("/fin/transactions");
    }
  };

  const actionButtons = (
    <div className="flex flex-wrap items-center gap-2">
      {!hasPostedEntries && (
        <>
          {canEditTransaction && (
            <Button
              icon={<EditOutlined />}
              onClick={() => navigate(`/fin/transactions/${id}/edit`)}
            >
              {t("fin.transactions.edit", "Edit transaction")}
            </Button>
          )}
          {canDeleteTransaction && (
            <Popconfirm
              title={t(
                "fin.transactions.deleteConfirm",
                "Are you sure you want to delete this transaction?",
              )}
              onConfirm={handleDelete}
            >
              <Button danger icon={<DeleteOutlined />} loading={pending}>
                {t("fin.transactions.delete", "Delete transaction")}
              </Button>
            </Popconfirm>
          )}
        </>
      )}

      {canCreateEntry && (
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() =>
            navigate(`/fin/journal/new?transactionId=${transaction.id}`)
          }
        >
          {t("fin.transactions.addEntry", "Add entry")}
        </Button>
      )}
    </div>
  );

  const entryColumns: TableProps<Entry>["columns"] = [
    {
      title: t("fin.journal.number", "Number"),
      dataIndex: "number",
      key: "number",
      render: (num, record) => (
        <span className="font-semibold text-primary">
          {num || `#${record.id}`}
        </span>
      ),
    },
    {
      title: t("common.date", "Date"),
      dataIndex: "entryDate",
      key: "entryDate",
      render: (val) => formatDate(val, i18n.language),
    },
    {
      title: t("fin.journal.label", "Description"),
      dataIndex: "description",
      key: "description",
      render: (text) => text || "-",
    },
    {
      title: t("fin.journal.amount", "Amount"),
      key: "amount",
      align: "right",
      render: (_, record) => {
        const totalDebit = (record.postings ?? []).reduce(
          (sum, p) =>
            p.direction === "DEBIT"
              ? sum +
                (typeof p.amount === "string"
                  ? Number.parseFloat(p.amount) || 0
                  : (p.amount ?? 0))
              : sum,
          0,
        );
        return (
          <MoneyCell
            amount={totalDebit}
            currency={transaction.currency || "IDR"}
          />
        );
      },
    },
    {
      title: t("common.status", "Status"),
      dataIndex: "status",
      key: "status",
      render: (st) => <EntryStatusTag status={st} />,
    },
  ];

  return (
    <Page
      title={transaction.number || `#${transaction.id}`}
      trail={[
        {
          title: t("fin.menu.transactions", "Transactions"),
          onClick: () => navigate("/fin/transactions"),
        },
      ]}
      actions={
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/fin/transactions")}
        >
          {t("fin.menu.transactions", "Transactions")}
        </Button>
      }
    >
      <FormView
        title={
          <span className="font-semibold text-lg">
            {transaction.number || `#${transaction.id}`}
          </span>
        }
        buttons={actionButtons}
      >
        <div className="p-6 flex flex-col gap-6">
          {/* Header Info */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.number", "Number")}
              </span>
              <span className="font-semibold text-sm">
                {transaction.number || `#${transaction.id}`}
              </span>
            </div>
            <div>
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.date", "Date")}
              </span>
              <span className="text-sm">
                {formatDate(transaction.transactionDate, i18n.language)}
              </span>
            </div>
            <div>
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.currency", "Currency")}
              </span>
              <span className="text-sm font-mono">
                {transaction.currency || "IDR"}{" "}
                {transaction.currency !== "IDR" &&
                  `(Rate: ${transaction.exchangeRate || 1})`}
              </span>
            </div>
            <div>
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.source", "Source")}
              </span>
              <span className="text-sm">
                {transaction.source
                  ? t(
                      `fin.transactions.sources.${transaction.source}`,
                      transaction.source,
                    )
                  : "-"}
              </span>
            </div>
            <div>
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.reference", "Reference")}
              </span>
              <span className="text-sm">{transaction.reference || "-"}</span>
            </div>
            <div className="md:col-span-3">
              <span className="text-xs text-muted block mb-1">
                {t("fin.transactions.description", "Description")}
              </span>
              <span className="text-sm">{transaction.description || "-"}</span>
            </div>
          </div>

          {/* Embedded Entries Table */}
          <div className="pt-4 border-t">
            <h3 className="text-base font-semibold mb-3">
              {t("fin.transactions.entries", "Journal entries")}
            </h3>
            <Table<Entry>
              rowKey="id"
              columns={entryColumns}
              dataSource={entries}
              pagination={false}
              onRow={(record) => ({
                onClick: () => {
                  if (record.id) {
                    navigate(`/fin/journal/${record.id}`);
                  }
                },
                className: "cursor-pointer hover:bg-muted/50",
              })}
            />
          </div>

          {/* Attachments Section */}
          <TransactionAttachmentsSection
            wid={wid}
            transactionId={id}
            attachments={transaction.attachments ?? []}
            canModify={!hasPostedEntries && canEditTransaction}
          />
        </div>
      </FormView>
    </Page>
  );
}
