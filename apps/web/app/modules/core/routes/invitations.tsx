import {
  CopyOutlined,
  MailOutlined,
  PlusOutlined,
  StopOutlined,
} from "@ant-design/icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  App,
  Button,
  DatePicker,
  Empty,
  Form,
  Input,
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
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { env } from "~/lib/env";
import { formatDateTime } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { getActiveWorkspace } from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import { type Invitation, invitationStatus } from "../lib";
import { invitationsQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/invitations";

export async function clientLoader() {
  const workspace = await getActiveWorkspace();
  if (workspace.personal) {
    return { wid: workspace.id, isPersonal: true };
  }
  const wid = workspace.id;
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(invitationsQueryOptions(wid)),
    queryClient.ensureQueryData(rolesQueryOptions(wid)),
  ]);
  return { wid, isPersonal: false };
}

type FormValues = {
  roles: string[];
  maxUses?: number;
  expiresAt?: { toISOString: () => string };
  email?: string;
};

const statusColor = {
  active: "success",
  revoked: "error",
  expired: "default",
  exhausted: "warning",
} as const;

export default function Invitations({ loaderData }: Route.ComponentProps) {
  const { t } = useTranslation();
  if (loaderData.isPersonal) {
    return (
      <Page title={t("core.menu.invitations")}>
        <Panel strong>
          <Empty description={t("core.invitations.personalNotice")} />
        </Panel>
      </Page>
    );
  }
  return <InvitationsContent wid={loaderData.wid} />;
}

function InvitationsContent({ wid }: { wid: number }) {
  const { data: invitations } = useSuspenseQuery(invitationsQueryOptions(wid));
  const { options: roleOptions, label: roleName } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const { run, pending } = useAction();
  const [creating, setCreating] = useState(false);
  const [sendingTo, setSendingTo] = useState<Invitation | null>(null);

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
              email: values.email?.trim() || undefined,
            },
          }),
        ),
      "common.saved",
    );

  const sendEmail = (values: { email: string }) =>
    run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/invitations/{id}/send", {
            params: { path: { wid, id: sendingTo?.id ?? 0 } },
            body: { email: values.email.trim() },
          }),
        ),
      "core.invitations.emailSent",
    );

  const revoke = (invitation: Invitation) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/invitations/{id}", {
            params: { path: { wid, id: invitation.id ?? 0 } },
          }),
        ),
      "core.invitations.revoked",
    );

  const copyLink = async (invitation: Invitation) => {
    const link = `${env.publicUrl}/invite/${invitation.token}`;
    try {
      await navigator.clipboard.writeText(link);
      message.success(t("core.invitations.copied"));
    } catch {
      message.info(link);
    }
  };

  const columns: TableProps<Invitation>["columns"] = [
    {
      title: t("core.invitations.link"),
      render: (_, i) => (
        <div className="flex items-center gap-1">
          <code>{`…${(i.token ?? "").slice(-8)}`}</code>
          <Tooltip title={t("core.invitations.copy")}>
            <Button
              type="text"
              icon={<CopyOutlined />}
              aria-label={t("core.invitations.copy")}
              onClick={() => copyLink(i)}
            />
          </Tooltip>
        </div>
      ),
    },
    {
      title: t("core.members.roles"),
      render: (_, i) => (
        <div className="flex flex-wrap gap-1">
          {(i.roles ?? []).map((role) => (
            <Tag key={role}>{roleName(role)}</Tag>
          ))}
        </div>
      ),
    },
    {
      title: t("core.invitations.status"),
      render: (_, i) => {
        const status = invitationStatus(i);
        return (
          <Tag color={statusColor[status]}>
            {t(`core.invitations.${status}`)}
          </Tag>
        );
      },
    },
    {
      title: t("core.invitations.uses"),
      align: "right",
      render: (_, i) =>
        i.maxUses
          ? `${i.usesCount ?? 0} / ${i.maxUses}`
          : `${i.usesCount ?? 0} / ∞`,
    },
    {
      title: t("core.invitations.expires"),
      align: "right",
      render: (_, i) =>
        i.expiresAt
          ? formatDateTime(i.expiresAt, i18n.language)
          : t("core.invitations.never"),
    },
    {
      title: "",
      align: "right",
      render: (_, i) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip title={t("core.invitations.sendEmail")}>
            <Button
              type="text"
              icon={<MailOutlined />}
              disabled={invitationStatus(i) !== "active"}
              aria-label={t("core.invitations.sendEmail")}
              onClick={() => setSendingTo(i)}
            />
          </Tooltip>
          <Popconfirm
            title={t("core.invitations.revokeConfirm")}
            okText={t("core.invitations.revoke")}
            cancelText={t("common.cancel")}
            disabled={invitationStatus(i) !== "active"}
            onConfirm={() => revoke(i)}
          >
            <Button
              type="text"
              danger
              icon={<StopOutlined />}
              disabled={invitationStatus(i) !== "active"}
              aria-label={t("core.invitations.revoke")}
            />
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <Page
      title={t("core.menu.invitations")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreating(true)}
        >
          {t("core.invitations.new")}
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
        title={t("core.invitations.new")}
        initialValues={{ roles: [] }}
        submitting={pending}
        onClose={() => setCreating(false)}
        onSubmit={create}
      >
        {() => (
          <>
            <Form.Item
              name="roles"
              label={t("core.members.roles")}
              rules={[{ required: true, type: "array", min: 1 }]}
            >
              <Select
                mode="multiple"
                options={roleOptions}
                optionFilterProp="label"
                placeholder={t("core.members.rolesPlaceholder")}
              />
            </Form.Item>
            <Form.Item
              name="email"
              label={t("core.invitations.email")}
              extra={t("core.invitations.emailHint")}
              rules={[{ type: "email" }]}
            >
              <Input placeholder="user@example.com" />
            </Form.Item>
            <Form.Item
              name="maxUses"
              label={t("core.invitations.maxUses")}
              extra={t("core.invitations.maxUsesHint")}
            >
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="expiresAt" label={t("core.invitations.expires")}>
              <DatePicker showTime style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}
      </RecordModal>

      <RecordModal<{ email: string }>
        open={Boolean(sendingTo)}
        title={t("core.invitations.sendEmail")}
        initialValues={{ email: "" }}
        submitting={pending}
        onClose={() => setSendingTo(null)}
        onSubmit={sendEmail}
      >
        {() => (
          <Form.Item
            name="email"
            label={t("core.invitations.email")}
            rules={[{ required: true, type: "email" }]}
          >
            <Input placeholder="user@example.com" autoFocus />
          </Form.Item>
        )}
      </RecordModal>
    </Page>
  );
}
