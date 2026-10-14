import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import {
  Avatar,
  Button,
  Form,
  Input,
  Popconfirm,
  Table,
  type TableProps,
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
import type { Route } from "./+types/users";

type User = components["schemas"]["model.UserResponse"];

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const name = new URL(request.url).searchParams.get("q") ?? undefined;
  const users = await unwrapList(
    api.POST("/users/search", { body: { name, limit: 100 } }),
  );
  return { users };
}

type FormValues = { name: string; email?: string; avatar?: string };

export default function Users({ loaderData }: Route.ComponentProps) {
  const { users } = loaderData;
  const { t, i18n } = useTranslation();
  const [params, setParams] = useSearchParams();
  const { run, pending } = useAction();
  const [editing, setEditing] = useState<User | "new" | null>(null);

  const save = (values: FormValues) =>
    run(async () => {
      if (editing === "new" || editing === null) {
        await unwrapOne(
          api.POST("/users", {
            body: {
              name: values.name,
              email: values.email ?? "",
              avatar: values.avatar || undefined,
            },
          }),
        );
      } else if (editing.id !== undefined) {
        await unwrapOne(
          api.PUT("/users/{id}", {
            params: { path: { id: editing.id } },
            body: { name: values.name, avatar: values.avatar || undefined },
          }),
        );
      }
    }, "common.saved");

  const remove = (user: User) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/users/{id}", {
            params: { path: { id: user.id ?? 0 } },
          }),
        ),
      "common.deleted",
    );

  const columns: TableProps<User>["columns"] = [
    {
      title: t("common.name"),
      render: (_, u) => (
        <div className="flex items-center gap-2">
          <Avatar size={24} src={u.avatar}>
            {u.name?.slice(0, 1).toUpperCase()}
          </Avatar>
          {u.name}
        </div>
      ),
    },
    { title: t("common.email"), dataIndex: "email" },
    {
      title: t("common.created"),
      align: "right",
      render: (_, u) => formatDate(u.createdAt, i18n.language),
    },
    {
      title: "",
      align: "right",
      render: (_, u) => (
        <div className="flex justify-end gap-1">
          <Tooltip title={t("common.edit")}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={t("common.edit")}
              onClick={() => setEditing(u)}
            />
          </Tooltip>
          <Popconfirm
            title={t("user.deleteConfirm")}
            okText={t("common.delete")}
            cancelText={t("common.cancel")}
            onConfirm={() => remove(u)}
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

  return (
    <Page
      title={t("user.menu.users")}
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
            {t("user.new")}
          </Button>
        </>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={users}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>

      <RecordModal<FormValues>
        open={editing !== null}
        title={isNew ? t("user.new") : t("user.edit")}
        initialValues={
          editing && !isNew
            ? { name: editing.name, avatar: editing.avatar }
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
            {isNew ? (
              <Form.Item
                name="email"
                label={t("common.email")}
                rules={[{ required: true, type: "email" }]}
              >
                <Input />
              </Form.Item>
            ) : null}
            <Form.Item name="avatar" label={t("user.avatar")}>
              <Input placeholder="https://" />
            </Form.Item>
          </>
        )}
      </RecordModal>
    </Page>
  );
}
