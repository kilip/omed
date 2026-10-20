import {
  ArrowLeftOutlined,
  CheckOutlined,
  DeleteOutlined,
  EditOutlined,
  RollbackOutlined,
  StopOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import {
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Popconfirm,
  Table,
  type TableProps,
} from "antd";
import dayjs from "dayjs";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { StatusBar } from "~/components/StatusBar";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { formatDate, formatDateTime } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import {
  apiToPostingLines,
  calculatePostingsSummary,
  EntryStatusTag,
  MoneyCell,
  PostingsEditor,
  postingLinesToApi,
} from "../components";
import { useFinPermissions } from "../hooks";
import {
  accountsQueryOptions,
  entryQueryOptions,
  periodsQueryOptions,
  transactionQueryOptions,
} from "../queries";
import type { Route } from "./+types/entry";

type Posting = components["schemas"]["model.PostingResponse"];

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(entryQueryOptions(wid, id)),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(periodsQueryOptions(wid)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id };
}

export default function EntryRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { run, pending } = useAction();
  const {
    canEditEntry,
    canDeleteEntry,
    canPostEntry,
    canVoidEntry,
    canReverseEntry,
  } = useFinPermissions(wid);

  const { data: entry } = useSuspenseQuery(entryQueryOptions(wid, id));
  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));

  // If entry has a transaction, fetch transaction to know currency & exchange rate
  const { data: transaction } = useQuery({
    ...transactionQueryOptions(wid, entry.transactionId ?? 0),
    enabled: !!entry.transactionId,
  });

  const currency = transaction?.currency || "IDR";
  const exchangeRate = transaction?.exchangeRate || 1;
  const isMultiCurrency = currency.toUpperCase() !== "IDR";

  const period = periods.find((p) => p.id === entry.periodId);

  // Editing state for DRAFT
  const [isEditing, setIsEditing] = useState(false);
  const [editForm] = Form.useForm();

  // Reverse modal state
  const [reverseModalOpen, setReverseModalOpen] = useState(false);
  const [reverseForm] = Form.useForm();

  // Summary for read-only postings
  const summary = calculatePostingsSummary(
    apiToPostingLines(entry.postings, exchangeRate),
    currency,
    exchangeRate,
  );

  const startEdit = () => {
    editForm.setFieldsValue({
      description: entry.description ?? "",
      postings: apiToPostingLines(entry.postings, exchangeRate),
    });
    setIsEditing(true);
  };

  const cancelEdit = () => {
    setIsEditing(false);
    editForm.resetFields();
  };

  const handleSaveEdit = async () => {
    try {
      const values = await editForm.validateFields();
      const postings = postingLinesToApi(
        values.postings,
        exchangeRate,
        currency,
        values.description || entry.description,
      );
      const summaryCheck = calculatePostingsSummary(
        values.postings,
        currency,
        exchangeRate,
      );
      if (!summaryCheck.balanced) {
        return;
      }

      const ok = await run(
        () =>
          unwrapOne(
            api.PUT("/workspaces/{wid}/fin/entries/{id}", {
              params: { path: { wid, id } },
              body: {
                description: values.description,
                postings,
              },
            }),
          ),
        "common.saved",
      );

      if (ok) {
        setIsEditing(false);
      }
    } catch {
      // Form validation failed
    }
  };

  const handlePost = async () => {
    await run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/fin/entries/{id}/post", {
            params: { path: { wid, id } },
          }),
        ),
      "fin.journal.trackPosted",
    );
  };

  const handleDelete = async () => {
    const ok = await run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/fin/entries/{id}", {
            params: { path: { wid, id } },
          }),
        ),
      "common.deleted",
    );
    if (ok) {
      navigate("/fin/journal");
    }
  };

  const handleVoid = async () => {
    await run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/fin/entries/{id}/void", {
            params: { path: { wid, id } },
          }),
        ),
      "common.saved",
    );
  };

  const handleReverseSubmit = async () => {
    try {
      const values = await reverseForm.validateFields();
      const entryDate = values.entryDate
        ? values.entryDate.format("YYYY-MM-DD")
        : undefined;

      let newReversingId: number | undefined;

      const ok = await run(async () => {
        const res = await unwrapOne(
          api.POST("/workspaces/{wid}/fin/entries/{id}/reverse", {
            params: { path: { wid, id } },
            body: {
              entryDate,
              description: values.description,
            },
          }),
        );
        newReversingId = res.id;
      }, "common.saved");

      if (ok) {
        setReverseModalOpen(false);
        reverseForm.resetFields();
        if (newReversingId) {
          navigate(`/fin/journal/${newReversingId}`);
        }
      }
    } catch {
      // validation error
    }
  };

  const status = entry.status ?? "DRAFT";

  // Action buttons depending on entry status and permissions
  const actionButtons = (
    <div className="flex flex-wrap items-center gap-2">
      {status === "DRAFT" && !isEditing && (
        <>
          {canPostEntry && (
            <Button
              type="primary"
              icon={<CheckOutlined />}
              loading={pending}
              onClick={handlePost}
            >
              {t("fin.journal.post", "Post")}
            </Button>
          )}
          {canEditEntry && (
            <Button icon={<EditOutlined />} onClick={startEdit}>
              {t("fin.journal.editEntry", "Edit entry")}
            </Button>
          )}
          {canDeleteEntry && (
            <Popconfirm
              title={t(
                "fin.journal.deleteConfirm",
                "Delete this draft journal entry?",
              )}
              onConfirm={handleDelete}
            >
              <Button danger icon={<DeleteOutlined />} loading={pending}>
                {t("common.delete", "Delete")}
              </Button>
            </Popconfirm>
          )}
          {canVoidEntry && (
            <Popconfirm
              title={t(
                "fin.journal.voidConfirm",
                "Are you sure you want to void this entry?",
              )}
              okButtonProps={{ className: "popconfirm-ok-btn" }}
              onConfirm={handleVoid}
            >
              <Button icon={<StopOutlined />} loading={pending}>
                {t("fin.journal.void", "Void")}
              </Button>
            </Popconfirm>
          )}
        </>
      )}

      {status === "DRAFT" && isEditing && canEditEntry && (
        <>
          <Button type="primary" loading={pending} onClick={handleSaveEdit}>
            {t("fin.journal.saveChanges", "Save changes")}
          </Button>
          <Button onClick={cancelEdit}>{t("common.cancel", "Cancel")}</Button>
        </>
      )}

      {status === "POSTED" && (
        <>
          {canReverseEntry && (
            <Button
              type="primary"
              icon={<RollbackOutlined />}
              onClick={() => {
                reverseForm.setFieldsValue({
                  entryDate: dayjs(),
                  description: `Reversal of ${entry.number}`,
                });
                setReverseModalOpen(true);
              }}
            >
              {t("fin.journal.reverse", "Reverse")}
            </Button>
          )}
          {canVoidEntry && (
            <Popconfirm
              title={t(
                "fin.journal.voidConfirmPosted",
                "Voiding a posted entry will create a reversing entry and mark this entry as void. Proceed?",
              )}
              okButtonProps={{ className: "popconfirm-ok-btn" }}
              onConfirm={handleVoid}
            >
              <Button danger icon={<StopOutlined />} loading={pending}>
                {t("fin.journal.void", "Void")}
              </Button>
            </Popconfirm>
          )}
        </>
      )}
    </div>
  );

  const postingColumns: TableProps<Posting>["columns"] = [
    {
      title: t("fin.journal.account", "Account"),
      key: "account",
      render: (_, record) => {
        const acc = accounts.find((a) => a.id === record.accountId);
        return acc ? (
          <Link
            to={`/fin/ledger?account=${acc.id}`}
            className="text-primary hover:underline"
          >
            {acc.code} - {acc.name}
          </Link>
        ) : (
          `#${record.accountId}`
        );
      },
    },
    {
      title: t("fin.journal.label", "Description"),
      dataIndex: "description",
      key: "description",
      render: (val) => val || "-",
    },
    {
      title: t("fin.journal.debit", "Debit"),
      key: "debit",
      align: "right",
      render: (_, record) => {
        if (record.direction !== "DEBIT") return "";
        return <MoneyCell amount={record.amount} currency={currency} />;
      },
    },
    {
      title: t("fin.journal.credit", "Credit"),
      key: "credit",
      align: "right",
      render: (_, record) => {
        if (record.direction !== "CREDIT") return "";
        return <MoneyCell amount={record.amount} currency={currency} />;
      },
    },
    ...(isMultiCurrency
      ? [
          {
            title: "Base (IDR)",
            key: "baseAmount",
            align: "right" as const,
            render: (_: unknown, record: Posting) => (
              <MoneyCell amount={record.baseAmount} currency="IDR" />
            ),
          },
        ]
      : []),
  ];

  return (
    <Page
      title={entry.number || `#${entry.id}`}
      trail={[
        {
          title: t("fin.menu.journal", "Journal Entries"),
          onClick: () => navigate("/fin/journal"),
        },
      ]}
      actions={
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/fin/journal")}
        >
          {t("fin.menu.journal", "Journal Entries")}
        </Button>
      }
    >
      <FormView
        title={
          <div className="flex items-center gap-2">
            <span className="font-semibold text-lg">
              {entry.number || `#${entry.id}`}
            </span>
            <EntryStatusTag status={status} />
          </div>
        }
        buttons={actionButtons}
        statusbar={
          <StatusBar
            label={t("common.status", "Status")}
            current={status}
            steps={
              status === "VOID"
                ? [
                    { value: "DRAFT", label: t("fin.journal.DRAFT", "Draft") },
                    { value: "VOID", label: t("fin.journal.VOID", "Void") },
                  ]
                : [
                    { value: "DRAFT", label: t("fin.journal.DRAFT", "Draft") },
                    {
                      value: "POSTED",
                      label: t("fin.journal.POSTED", "Posted"),
                    },
                  ]
            }
          />
        }
      >
        {isEditing ? (
          <Form form={editForm} layout="vertical" className="mt-2">
            <Form.Item
              name="description"
              label={t("fin.journal.label", "Description")}
            >
              <Input placeholder={t("fin.journal.label", "Description")} />
            </Form.Item>

            <PostingsEditor
              accounts={accounts}
              currency={currency}
              exchangeRate={exchangeRate}
            />
          </Form>
        ) : (
          <div className="flex flex-col gap-6">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 rounded border border-border bg-card/50 text-sm">
              <div>
                <span className="text-xs text-muted-foreground block">
                  {t("common.date", "Date")}
                </span>
                <span className="font-medium">
                  {formatDate(entry.entryDate, i18n.language)}
                </span>
              </div>
              <div>
                <span className="text-xs text-muted-foreground block">
                  {t("fin.journal.period", "Period")}
                </span>
                <span className="font-medium">
                  {period?.name ?? `#${entry.periodId}`}
                </span>
              </div>
              <div>
                <span className="text-xs text-muted-foreground block">
                  {t("fin.journal.transaction", "Transaction")}
                </span>
                {entry.transactionId ? (
                  <Link
                    to={`/fin/transactions/${entry.transactionId}`}
                    className="text-primary hover:underline font-medium"
                  >
                    #{entry.transactionId}
                  </Link>
                ) : (
                  "-"
                )}
              </div>
              <div>
                <span className="text-xs text-muted-foreground block">
                  {t("fin.journal.postedAt", "Posted at")}
                </span>
                <span className="font-medium">
                  {entry.postedAt
                    ? formatDateTime(entry.postedAt, i18n.language)
                    : "-"}
                </span>
              </div>

              {entry.reversalOfId && (
                <div className="col-span-2">
                  <span className="text-xs text-muted-foreground block">
                    {t("fin.journal.reversalOf", "Reversal of entry")}
                  </span>
                  <Link
                    to={`/fin/journal/${entry.reversalOfId}`}
                    className="text-primary hover:underline font-medium"
                  >
                    #{entry.reversalOfId}
                  </Link>
                </div>
              )}

              {entry.description && (
                <div className="col-span-2">
                  <span className="text-xs text-muted-foreground block">
                    {t("fin.journal.label", "Description")}
                  </span>
                  <span className="font-medium">{entry.description}</span>
                </div>
              )}
            </div>

            <div>
              <h3 className="text-sm font-semibold mb-2">
                {t("fin.journal.account", "Postings")}
              </h3>
              <Table<Posting>
                size="small"
                rowKey="id"
                columns={postingColumns}
                dataSource={entry.postings ?? []}
                pagination={false}
                scroll={{ x: "max-content" }}
                summary={() => (
                  <Table.Summary.Row>
                    <Table.Summary.Cell index={0} colSpan={2}>
                      <strong>{t("common.total", "Total")}</strong>
                    </Table.Summary.Cell>
                    <Table.Summary.Cell index={2} align="right">
                      <MoneyCell
                        amount={summary.totalDebit}
                        currency={currency}
                        className="font-bold"
                      />
                    </Table.Summary.Cell>
                    <Table.Summary.Cell index={3} align="right">
                      <MoneyCell
                        amount={summary.totalCredit}
                        currency={currency}
                        className="font-bold"
                      />
                    </Table.Summary.Cell>
                    {isMultiCurrency && (
                      <Table.Summary.Cell index={4} align="right">
                        <MoneyCell
                          amount={summary.totalBaseDebit}
                          currency="IDR"
                          className="font-bold"
                        />
                      </Table.Summary.Cell>
                    )}
                  </Table.Summary.Row>
                )}
              />
            </div>
          </div>
        )}
      </FormView>

      {canReverseEntry && (
        <Modal
          open={reverseModalOpen}
          title={t("fin.journal.reverseModalTitle", "Reverse journal entry")}
          onCancel={() => setReverseModalOpen(false)}
          onOk={handleReverseSubmit}
          confirmLoading={pending}
        >
          <Form form={reverseForm} layout="vertical" className="mt-4">
            <Form.Item
              name="entryDate"
              label={t("fin.journal.reverseDate", "Reversal date")}
              rules={[{ required: true }]}
            >
              <DatePicker className="w-full" />
            </Form.Item>
            <Form.Item
              name="description"
              label={t(
                "fin.journal.reverseDescription",
                "Reversal description",
              )}
            >
              <Input
                placeholder={t("fin.journal.reverseDescription", "Description")}
              />
            </Form.Item>
          </Form>
        </Modal>
      )}
    </Page>
  );
}
