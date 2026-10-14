import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Button,
  Form,
  Input,
  Popconfirm,
  Segmented,
  Select,
  Switch,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { accountsQueryOptions } from "../queries";
import type { Route } from "./+types/accounts";

type Account = components["schemas"]["model.AccountResponse"];
type AccountType = components["schemas"]["model.CreateAccountRequest"]["type"];
type NormalBalance = "DEBIT" | "CREDIT";
type Row = Account & { children?: Row[] };

const types: AccountType[] = [
  "ASSET",
  "LIABILITY",
  "EQUITY",
  "REVENUE",
  "EXPENSE",
];

const defaultNormal = (type: AccountType): NormalBalance =>
  type === "ASSET" || type === "EXPENSE" ? "DEBIT" : "CREDIT";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  await getQueryClient().ensureQueryData(accountsQueryOptions(wid));
  return { wid };
}

function buildTree(accounts: Account[]): Row[] {
  const byId = new Map<number, Row>();
  for (const account of accounts) {
    if (account.id !== undefined) byId.set(account.id, { ...account });
  }

  const roots: Row[] = [];
  for (const row of byId.values()) {
    const parent = row.parentId ? byId.get(row.parentId) : undefined;
    if (parent) {
      parent.children = [...(parent.children ?? []), row];
    } else {
      roots.push(row);
    }
  }

  const byCode = (a: Row, b: Row) => (a.code ?? "").localeCompare(b.code ?? "");
  const sort = (rows: Row[]) => {
    rows.sort(byCode);
    for (const row of rows) if (row.children) sort(row.children);
  };
  sort(roots);
  return roots;
}

type FormValues = {
  code: string;
  name: string;
  type: AccountType;
  parentId?: number;
  normalBalance?: NormalBalance;
  description?: string;
  isPostable: boolean;
  isActive: boolean;
};

export default function Accounts({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { data: accounts } = useSuspenseQuery(accountsQueryOptions(wid));
  const { t } = useTranslation();
  const { run, pending } = useAction();
  const [editing, setEditing] = useState<Account | "new" | null>(null);
  const [filter, setFilter] = useState<AccountType | "ALL">("ALL");
  const [query, setQuery] = useState("");

  const rows = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return buildTree(
      accounts.filter(
        (a) =>
          (filter === "ALL" || a.type === filter) &&
          (!needle ||
            a.code?.toLowerCase().includes(needle) ||
            a.name?.toLowerCase().includes(needle)),
      ),
    );
  }, [accounts, filter, query]);

  const save = (values: FormValues) =>
    run(async () => {
      if (editing === "new" || editing === null) {
        await unwrapOne(
          api.POST("/workspaces/{wid}/fin/accounts", {
            params: { path: { wid } },
            body: { ...values },
          }),
        );
      } else if (editing.id !== undefined) {
        const { type: _type, ...rest } = values;
        await unwrapOne(
          api.PUT("/workspaces/{wid}/fin/accounts/{id}", {
            params: { path: { wid, id: editing.id } },
            body: rest,
          }),
        );
      }
    }, "common.saved");

  const remove = (account: Account) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/fin/accounts/{id}", {
            params: { path: { wid, id: account.id ?? 0 } },
          }),
        ),
      "common.deleted",
    );

  const columns: TableProps<Row>["columns"] = [
    {
      title: t("fin.accounts.code"),
      dataIndex: "code",
      width: 140,
      render: (code: string) => <code>{code}</code>,
    },
    { title: t("common.name"), dataIndex: "name" },
    {
      title: t("common.type"),
      dataIndex: "type",
      render: (type: AccountType) => <Tag>{t(`fin.types.${type}`)}</Tag>,
    },
    {
      title: t("fin.accounts.normalBalance"),
      dataIndex: "normalBalance",
      render: (value?: NormalBalance) =>
        value ? t(`fin.normal.${value}`) : "-",
    },
    {
      title: t("fin.accounts.postable"),
      dataIndex: "isPostable",
      render: (value?: boolean) => (value ? t("common.yes") : t("common.no")),
    },
    {
      title: t("common.status"),
      dataIndex: "isActive",
      render: (value?: boolean) => (
        <span className="inline-flex items-center gap-1">
          <span
            aria-hidden="true"
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{
              background: value
                ? "var(--omed-success)"
                : "var(--omed-text-muted)",
            }}
          />
          {value ? t("common.active") : t("common.inactive")}
        </span>
      ),
    },
    {
      title: "",
      align: "right",
      render: (_, account) => (
        <div className="flex justify-end gap-1">
          <Tooltip title={t("common.edit")}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={t("common.edit")}
              onClick={() => setEditing(account)}
            />
          </Tooltip>
          <Popconfirm
            title={t("fin.accounts.deleteConfirm")}
            okText={t("common.delete")}
            cancelText={t("common.cancel")}
            onConfirm={() => remove(account)}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              aria-label={t("common.delete")}
            />
          </Popconfirm>
        </div>
      ),
    },
  ];

  const isNew = editing === "new";
  const current = editing && !isNew ? editing : undefined;

  return (
    <Page
      title={t("fin.menu.accounts")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setEditing("new")}
        >
          {t("fin.accounts.new")}
        </Button>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Input.Search
          allowClear
          style={{ width: 260 }}
          placeholder={t("fin.accounts.search")}
          onSearch={setQuery}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Segmented<AccountType | "ALL">
          value={filter}
          onChange={setFilter}
          options={[
            { value: "ALL", label: t("fin.accounts.allTypes") },
            ...types.map((type) => ({
              value: type,
              label: t(`fin.types.${type}`),
            })),
          ]}
        />
      </div>

      <Panel strong>
        <Table<Row>
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={rows}
          pagination={false}
          expandable={{ defaultExpandAllRows: true }}
          scroll={{ x: "max-content" }}
        />
      </Panel>

      <RecordModal<FormValues>
        variant="drawer"
        open={editing !== null}
        title={isNew ? t("fin.accounts.new") : t("fin.accounts.edit")}
        initialValues={
          current
            ? {
                code: current.code,
                name: current.name,
                type: current.type as AccountType,
                parentId: current.parentId || undefined,
                normalBalance: current.normalBalance as NormalBalance,
                description: current.description,
                isPostable: current.isPostable ?? true,
                isActive: current.isActive ?? true,
              }
            : {
                type: "ASSET",
                normalBalance: "DEBIT",
                isPostable: true,
                isActive: true,
              }
        }
        submitting={pending}
        onClose={() => setEditing(null)}
        onSubmit={save}
      >
        {(form) => (
          <>
            <Form.Item
              name="code"
              label={t("fin.accounts.code")}
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="name"
              label={t("common.name")}
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="type"
              label={t("common.type")}
              rules={[{ required: true }]}
            >
              <Select
                disabled={!isNew}
                options={types.map((type) => ({
                  value: type,
                  label: t(`fin.types.${type}`),
                }))}
                onChange={(type: AccountType) =>
                  form.setFieldValue("normalBalance", defaultNormal(type))
                }
              />
            </Form.Item>
            <Form.Item name="parentId" label={t("fin.accounts.parent")}>
              <Select
                allowClear
                showSearch
                optionFilterProp="label"
                options={accounts
                  .filter((a) => a.id !== current?.id)
                  .map((a) => ({
                    value: a.id,
                    label: `${a.code} ${a.name}`,
                  }))}
              />
            </Form.Item>
            <Form.Item
              name="normalBalance"
              label={t("fin.accounts.normalBalance")}
            >
              <Select
                options={(["DEBIT", "CREDIT"] as const).map((value) => ({
                  value,
                  label: t(`fin.normal.${value}`),
                }))}
              />
            </Form.Item>
            <Form.Item name="description" label={t("common.description")}>
              <Input.TextArea rows={2} />
            </Form.Item>
            <div className="flex gap-6">
              <Form.Item
                name="isPostable"
                label={t("fin.accounts.postable")}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item
                name="isActive"
                label={t("common.active")}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </div>
          </>
        )}
      </RecordModal>
    </Page>
  );
}
