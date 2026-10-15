import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Avatar,
  Button,
  Form,
  Input,
  Popconfirm,
  Select,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import type { Member } from "../lib";
import { membersQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/members";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(membersQueryOptions(wid)),
    queryClient.ensureQueryData(rolesQueryOptions(wid)),
  ]);
  return { wid };
}

type FormValues = { email?: string; roles: string[] };

export default function Members({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { data: members } = useSuspenseQuery(membersQueryOptions(wid));
  const { options: roleOptions, label: roleName } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const { run, pending } = useAction();
  const [editing, setEditing] = useState<Member | "new" | null>(null);

  const save = (values: FormValues) =>
    run(async () => {
      if (editing === "new" || editing === null) {
        await unwrapOne(
          api.POST("/workspaces/{wid}/members", {
            params: { path: { wid } },
            body: { email: values.email ?? "", roles: values.roles },
          }),
        );
      } else if (editing.user?.id !== undefined) {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/members/{id}", {
            params: { path: { wid, id: editing.user.id } },
            body: { roles: values.roles },
          }),
        );
      }
    }, "common.saved");

  const remove = (member: Member) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/members/{id}", {
            params: { path: { wid, id: member.user?.id ?? 0 } },
          }),
        ),
      "common.deleted",
    );

  const columns: TableProps<Member>["columns"] = [
    {
      title: t("sys.members.user"),
      render: (_, m) => (
        <div className="flex items-center gap-2">
          <Avatar size={24} src={m.user?.avatar}>
            {m.user?.name?.slice(0, 1).toUpperCase()}
          </Avatar>
          <div className="leading-tight">
            <div>{m.user?.name}</div>
            <div
              className="text-xs"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {m.user?.email}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: t("sys.members.roles"),
      render: (_, m) => (
        <div className="flex flex-wrap gap-1">
          {(m.roles ?? []).map((role) => (
            <Tag key={role}>{roleName(role)}</Tag>
          ))}
        </div>
      ),
    },
    {
      title: t("sys.members.joined"),
      align: "right",
      render: (_, m) => formatDate(m.createdAt, i18n.language),
    },
    {
      title: "",
      align: "right",
      render: (_, m) => (
        <div className="flex justify-end gap-1">
          <Tooltip title={t("sys.members.editRoles")}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={t("sys.members.editRoles")}
              onClick={() => setEditing(m)}
            />
          </Tooltip>
          <Popconfirm
            title={t("sys.members.removeConfirm")}
            okText={t("sys.members.remove")}
            cancelText={t("common.cancel")}
            onConfirm={() => remove(m)}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              aria-label={t("sys.members.remove")}
            />
          </Popconfirm>
        </div>
      ),
    },
  ];

  const isNew = editing === "new";

  return (
    <Page
      title={t("sys.menu.members")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setEditing("new")}
        >
          {t("sys.members.invite")}
        </Button>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey={(m) => m.id ?? m.user?.email ?? ""}
          columns={columns}
          dataSource={members}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>

      <RecordModal<FormValues>
        open={editing !== null}
        title={isNew ? t("sys.members.invite") : t("sys.members.editRoles")}
        initialValues={
          editing && !isNew ? { roles: editing.roles ?? [] } : { roles: [] }
        }
        submitting={pending}
        onClose={() => setEditing(null)}
        onSubmit={save}
      >
        {() => (
          <>
            {isNew ? (
              <Form.Item
                name="email"
                label={t("common.email")}
                rules={[{ required: true, type: "email" }]}
              >
                <Input />
              </Form.Item>
            ) : null}
            <Form.Item
              name="roles"
              label={t("sys.members.roles")}
              rules={[{ required: true, type: "array", min: 1 }]}
            >
              <Select
                mode="multiple"
                options={roleOptions}
                optionFilterProp="label"
                placeholder={t("sys.members.rolesPlaceholder")}
              />
            </Form.Item>
          </>
        )}
      </RecordModal>
    </Page>
  );
}
