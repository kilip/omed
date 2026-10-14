import { CopyOutlined, PlusOutlined, StopOutlined } from "@ant-design/icons";
import {
  App,
  Button,
  DatePicker,
  Form,
  InputNumber,
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
import { api, unwrapList, unwrapOne, unwrapVoid } from "~/lib/api";
import { env } from "~/lib/env";
import { formatDateTime } from "~/lib/format";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { type Invitation, invitationStatus } from "../lib";
import type { Route } from "./+types/invitations";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const invitations = await unwrapList(
    api.POST("/workspaces/{wid}/invitations/search", {
      params: { path: { wid } },
      body: { limit: 100 },
    }),
  );
  return { wid, invitations };
}

type FormValues = {
  roles: string[];
  maxUses?: number;
  expiresAt?: { toISOString: () => string };
};

const statusColor = {
  active: "success",
  revoked: "error",
  expired: "default",
  exhausted: "warning",
} as const;

export default function Invitations({ loaderData }: Route.ComponentProps) {
  const { wid, invitations } = loaderData;
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const { run, pending } = useAction();
  const [creating, setCreating] = useState(false);

  const create = (values: FormValues) =>
    run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/invitations", {
            params: { path: { wid } },
            body: {
              roles: values.roles,
              maxUses: values.maxUses || undefined,
              expiresAt: values.expiresAt?.toISOString(),
            },
          }),
        ),
      "common.saved",
    );

  const revoke = (invitation: Invitation) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/invitations/{id}", {
            params: { path: { wid, id: invitation.id ?? 0 } },
          }),
        ),
      "sys.invitations.revoked",
    );

  const copyLink = async (invitation: Invitation) => {
    const link = `${env.publicUrl}/invite/${invitation.token}`;
    try {
      await navigator.clipboard.writeText(link);
      message.success(t("sys.invitations.copied"));
    } catch {
      message.info(link);
    }
  };

  const columns: TableProps<Invitation>["columns"] = [
    {
      title: t("sys.invitations.link"),
      render: (_, i) => (
        <div className="flex items-center gap-1">
          <code>{`…${(i.token ?? "").slice(-8)}`}</code>
          <Tooltip title={t("sys.invitations.copy")}>
            <Button
              type="text"
              icon={<CopyOutlined />}
              aria-label={t("sys.invitations.copy")}
              onClick={() => copyLink(i)}
            />
          </Tooltip>
        </div>
      ),
    },
    {
      title: t("sys.members.roles"),
      render: (_, i) => (
        <div className="flex flex-wrap gap-1">
          {(i.roles ?? []).map((role) => (
            <Tag key={role}>{role}</Tag>
          ))}
        </div>
      ),
    },
    {
      title: t("sys.invitations.status"),
      render: (_, i) => {
        const status = invitationStatus(i);
        return (
          <Tag color={statusColor[status]}>
            {t(`sys.invitations.${status}`)}
          </Tag>
        );
      },
    },
    {
      title: t("sys.invitations.uses"),
      align: "right",
      render: (_, i) =>
        i.maxUses
          ? `${i.usesCount ?? 0} / ${i.maxUses}`
          : `${i.usesCount ?? 0} / ∞`,
    },
    {
      title: t("sys.invitations.expires"),
      align: "right",
      render: (_, i) =>
        i.expiresAt
          ? formatDateTime(i.expiresAt, i18n.language)
          : t("sys.invitations.never"),
    },
    {
      title: "",
      align: "right",
      render: (_, i) => (
        <Popconfirm
          title={t("sys.invitations.revokeConfirm")}
          okText={t("sys.invitations.revoke")}
          cancelText={t("common.cancel")}
          disabled={invitationStatus(i) !== "active"}
          onConfirm={() => revoke(i)}
        >
          <Button
            type="text"
            danger
            icon={<StopOutlined />}
            disabled={invitationStatus(i) !== "active"}
            aria-label={t("sys.invitations.revoke")}
          />
        </Popconfirm>
      ),
    },
  ];

  return (
    <Page
      title={t("sys.menu.invitations")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreating(true)}
        >
          {t("sys.invitations.new")}
        </Button>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={invitations}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>

      <RecordModal<FormValues>
        open={creating}
        title={t("sys.invitations.new")}
        initialValues={{ roles: [] }}
        submitting={pending}
        onClose={() => setCreating(false)}
        onSubmit={create}
      >
        {() => (
          <>
            <Form.Item
              name="roles"
              label={t("sys.members.roles")}
              extra={t("sys.members.rolesHint")}
              rules={[{ required: true, type: "array", min: 1 }]}
            >
              <Select
                mode="tags"
                tokenSeparators={[","]}
                placeholder="fin:accountant"
                open={false}
              />
            </Form.Item>
            <Form.Item
              name="maxUses"
              label={t("sys.invitations.maxUses")}
              extra={t("sys.invitations.maxUsesHint")}
            >
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="expiresAt" label={t("sys.invitations.expires")}>
              <DatePicker showTime style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}
      </RecordModal>
    </Page>
  );
}
