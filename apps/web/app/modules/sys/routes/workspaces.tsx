import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import {
  Button,
  Form,
  Input,
  Popconfirm,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapList, unwrapOne, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId, type Workspace } from "~/lib/workspace";
import type { Route } from "./+types/workspaces";

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  await requireWorkspaceId();
  const name = new URL(request.url).searchParams.get("q") ?? undefined;
  const workspaces = await unwrapList(
    api.POST("/workspaces/search", { body: { name, limit: 100 } }),
  );
  return { workspaces };
}

type FormValues = { name: string; description?: string };

export default function Workspaces({ loaderData }: Route.ComponentProps) {
  const { workspaces } = loaderData;
  const { t, i18n } = useTranslation();
  const [params, setParams] = useSearchParams();
  const { run, pending } = useAction();
  const [editing, setEditing] = useState<Workspace | "new" | null>(null);

  const save = (values: FormValues) =>
    run(async () => {
      if (editing === "new" || editing === null) {
        await unwrapOne(api.POST("/workspaces", { body: values }));
      } else if (editing.id !== undefined) {
        await unwrapOne(
          api.PUT("/workspaces/{wid}", {
            params: { path: { wid: editing.id } },
            body: values,
          }),
        );
      }
    }, "common.saved");

  const remove = (workspace: Workspace) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}", {
            params: { path: { wid: workspace.id ?? 0 } },
          }),
        ),
      "common.deleted",
    );

  const columns: TableProps<Workspace>["columns"] = [
    { title: t("common.name"), dataIndex: "name" },
    { title: t("common.description"), dataIndex: "description" },
    { title: t("sys.workspaces.owner"), render: (_, w) => w.owner?.name },
    {
      title: t("common.type"),
      render: (_, w) =>
        w.personal ? (
          <Tag>{t("sys.workspaces.personal")}</Tag>
        ) : (
          <Tag variant="filled">{t("sys.workspaces.shared")}</Tag>
        ),
    },
    {
      title: t("common.created"),
      align: "right",
      render: (_, w) => formatDate(w.createdAt, i18n.language),
    },
    {
      title: "",
      align: "right",
      render: (_, w) => (
        <div className="flex justify-end gap-1">
          <Tooltip title={t("common.edit")}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={t("common.edit")}
              onClick={() => setEditing(w)}
            />
          </Tooltip>
          <Popconfirm
            title={t("sys.workspaces.deleteConfirm")}
            okText={t("common.delete")}
            cancelText={t("common.cancel")}
            disabled={w.personal}
            onConfirm={() => remove(w)}
          >
            <Button
              type="text"
              danger
              disabled={w.personal}
              icon={<DeleteOutlined />}
              aria-label={t("common.delete")}
            />
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <Page
      title={t("sys.menu.workspaces")}
      actions={
        <>
          <Input.Search
            allowClear
            style={{ width: 240 }}
            placeholder={t("common.search")}
            defaultValue={params.get("q") ?? ""}
            onSearch={(q) => setParams(q ? { q } : {})}
          />
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setEditing("new")}
          >
            {t("sys.workspaces.new")}
          </Button>
        </>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={workspaces}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>

      <RecordModal<FormValues>
        open={editing !== null}
        title={
          editing === "new" || editing === null
            ? t("sys.workspaces.new")
            : t("sys.workspaces.edit")
        }
        initialValues={
          editing && editing !== "new"
            ? { name: editing.name, description: editing.description }
            : undefined
        }
        submitting={pending}
        onClose={() => setEditing(null)}
        onSubmit={save}
      >
        {() => (
          <>
            <Form.Item
              name="name"
              label={t("common.name")}
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item name="description" label={t("common.description")}>
              <Input.TextArea rows={3} />
            </Form.Item>
          </>
        )}
      </RecordModal>
    </Page>
  );
}
