import {
  ArrowLeftOutlined,
  DeleteOutlined,
  EditOutlined,
  SendOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient } from "@tanstack/react-query";
import {
  App,
  Button,
  Descriptions,
  Popconfirm,
  Table,
  type TableProps,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { useFinPermissions } from "../hooks";
import {
  entryKeys,
  expenseKeys,
  incomeKeys,
  transactionKeys,
} from "../queries";
import { EntryStatusTag } from "./entry-status-tag";
import { MoneyCell } from "./money-cell";
import { TransactionAttachmentsSection } from "./transaction-attachments";

type CashFlowItem = components["schemas"]["model.CashFlowItemResponse"];
type CashFlowLine = components["schemas"]["model.CashFlowLineResponse"];

export function summarizeCategories(items: CashFlowLine[]): {
  first?: string;
  more: number;
} {
  if (!items || items.length === 0) {
    return { first: undefined, more: 0 };
  }
  const firstItem = items[0];
  const first = firstItem?.account
    ? `${firstItem.account.code} - ${firstItem.account.name}`
    : undefined;
  const more = Math.max(0, items.length - 1);
  return { first, more };
}

export interface CashflowDetailProps {
  wid: number;
  flow: "INCOME" | "EXPENSE" | "income" | "expense";
  item: CashFlowItem;
}

export function CashflowDetail({ wid, flow, item }: CashflowDetailProps) {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [postingPending, setPostingPending] = useState(false);
  const [deletePending, setDeletePending] = useState(false);

  const { canEditTransaction, canDeleteTransaction, canPostEntry } =
    useFinPermissions(wid);

  const isIncome = flow.toUpperCase() === "INCOME";
  const flowPath = isIncome ? "income" : "expense";
  const isDraft = item.entryStatus === "DRAFT";

  const handlePosting = async () => {
    if (!item.entryId) return;
    setPostingPending(true);
    try {
      await unwrapVoid(
        api.POST("/workspaces/{wid}/fin/entries/{id}/post", {
          params: { path: { wid, id: item.entryId } },
        }),
      );
      message?.success?.(
        t("fin.cashflow.postSuccess", "Document posted successfully"),
      );
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: isIncome ? incomeKeys.all(wid) : expenseKeys.all(wid),
        }),
        queryClient.invalidateQueries({
          queryKey: transactionKeys.all(wid),
        }),
        queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) }),
      ]);
    } catch (err) {
      message?.error?.(
        err instanceof Error ? err.message : t("common.error", "Error"),
      );
    } finally {
      setPostingPending(false);
    }
  };

  const handleDelete = async () => {
    if (!item.id) return;
    setDeletePending(true);
    try {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/fin/transactions/{id}", {
          params: { path: { wid, id: item.id } },
        }),
      );
      message?.success?.(
        t("fin.cashflow.deleteSuccess", "Document deleted successfully"),
      );
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: isIncome ? incomeKeys.all(wid) : expenseKeys.all(wid),
        }),
        queryClient.invalidateQueries({
          queryKey: transactionKeys.all(wid),
        }),
        queryClient.invalidateQueries({ queryKey: entryKeys.all(wid) }),
      ]);
      navigate(`/fin/${flowPath}`);
    } catch (err) {
      message?.error?.(
        err instanceof Error ? err.message : t("common.error", "Error"),
      );
    } finally {
      setDeletePending(false);
    }
  };

  const actionButtons = isDraft ? (
    <div className="flex flex-wrap items-center gap-2">
      {canEditTransaction && (
        <Button
          icon={<EditOutlined />}
          onClick={() => navigate(`/fin/${flowPath}/${item.id}/edit`)}
        >
          {t("common.edit", "Edit")}
        </Button>
      )}
      {canPostEntry && (
        <Button
          type="primary"
          icon={<SendOutlined />}
          loading={postingPending}
          onClick={handlePosting}
        >
          {t("fin.cashflow.posting", "Posting")}
        </Button>
      )}
      {canDeleteTransaction && (
        <Popconfirm
          title={t(
            "fin.cashflow.deleteConfirm",
            "Are you sure you want to delete this draft document?",
          )}
          onConfirm={handleDelete}
        >
          <Button danger icon={<DeleteOutlined />} loading={deletePending}>
            {t("common.delete", "Delete")}
          </Button>
        </Popconfirm>
      )}
    </div>
  ) : undefined;

  const itemColumns: TableProps<CashFlowLine>["columns"] = [
    {
      title: t("fin.cashflow.itemAccount", "Account"),
      key: "account",
      render: (_, record) =>
        record.account
          ? `${record.account.code} - ${record.account.name}`
          : "-",
    },
    {
      title: t("fin.cashflow.itemDescription", "Description"),
      dataIndex: "description",
      key: "description",
      render: (val) => val || "-",
    },
    {
      title: t("fin.cashflow.quantity", "Qty"),
      dataIndex: "quantity",
      key: "quantity",
      align: "right",
      render: (val) => val ?? 1,
    },
    {
      title: t("fin.cashflow.unitPrice", "Unit Price"),
      dataIndex: "unitPrice",
      key: "unitPrice",
      align: "right",
      render: (val) => (
        <MoneyCell amount={val ?? 0} currency={item.currency ?? "IDR"} />
      ),
    },
    {
      title: t("fin.cashflow.lineTotal", "Total"),
      dataIndex: "amount",
      key: "amount",
      align: "right",
      render: (val) => (
        <MoneyCell amount={val ?? 0} currency={item.currency ?? "IDR"} />
      ),
    },
  ];

  const moduleTitle = isIncome
    ? t("fin.menu.income", "Income")
    : t("fin.menu.expense", "Expense");
  const pageTitle =
    `${moduleTitle} ${item.number ? `#${item.number}` : ""}`.trim();

  return (
    <Page
      title={pageTitle}
      trail={[
        {
          title: moduleTitle,
          onClick: () => navigate(`/fin/${flowPath}`),
        },
      ]}
      actions={
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/fin/${flowPath}`)}
        >
          {moduleTitle}
        </Button>
      }
    >
      <FormView
        title={
          <div className="flex items-center gap-3">
            <span>{pageTitle}</span>
            <EntryStatusTag status={item.entryStatus} />
          </div>
        }
        buttons={actionButtons}
      >
        <div className="space-y-6 pt-2">
          <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="small">
            <Descriptions.Item label={t("common.date", "Date")}>
              {item.transactionDate
                ? formatDate(item.transactionDate, i18n.language)
                : "-"}
            </Descriptions.Item>
            <Descriptions.Item
              label={
                isIncome
                  ? t("fin.income.depositAccount", "Deposit Account")
                  : t("fin.expense.paymentAccount", "Payment Account")
              }
            >
              {item.paymentAccount
                ? `${item.paymentAccount.code} - ${item.paymentAccount.name}`
                : "-"}
            </Descriptions.Item>
            <Descriptions.Item
              label={
                isIncome
                  ? t("fin.income.reference", "Invoice Number")
                  : t("fin.expense.reference", "Reference")
              }
            >
              {item.reference || "-"}
            </Descriptions.Item>
            <Descriptions.Item
              label={t("fin.cashflow.description", "Description")}
            >
              {item.description || "-"}
            </Descriptions.Item>
            <Descriptions.Item label={t("fin.cashflow.currency", "Currency")}>
              {item.currency || "IDR"}
            </Descriptions.Item>
          </Descriptions>

          <Panel strong title={t("fin.cashflow.items", "Line Items")}>
            <Table<CashFlowLine>
              size="small"
              rowKey="id"
              columns={itemColumns}
              dataSource={item.items ?? []}
              pagination={false}
            />
            <div className="flex justify-end pt-4 border-t mt-4 border-border font-semibold text-base">
              <span className="mr-4">{t("fin.cashflow.total", "Total")}:</span>
              <MoneyCell
                amount={item.amount ?? 0}
                currency={item.currency ?? "IDR"}
              />
            </div>
          </Panel>

          {item.id != null && (
            <TransactionAttachmentsSection
              wid={wid}
              transactionId={item.id}
              attachments={item.attachments ?? []}
              canModify={isDraft && canEditTransaction}
            />
          )}
        </div>
      </FormView>
    </Page>
  );
}
