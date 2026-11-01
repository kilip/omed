import { CopyOutlined, PlusOutlined, StopOutlined } from "@ant-design/icons";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
  Alert,
  App,
  Button,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Table,
  type TableProps,
  Tag,
  theme,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { formatDateTime } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import {
  ACCESS_TOKEN_EXPIRY_DAYS,
  type AccessToken,
  type CreatedAccessToken,
  DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS,
} from "../lib";
import { accessTokenKeys, accessTokensQueryOptions } from "../queries";
import type { Route } from "./+types/access-tokens";

export async function clientLoader() {
  const queryClient = getQueryClient();
  await queryClient.ensureQueryData(accessTokensQueryOptions());
  return {};
}

export type CreateTokenFormValues = {
  name: string;
  expiresInDays?: number;
};

export async function submitCreateToken(
  values: CreateTokenFormValues,
): Promise<CreatedAccessToken> {
  const expiresInDays =
    values.expiresInDays ?? DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS;
  return unwrapOne(
    api.POST("/users/me/access-tokens", {
      body: {
        name: values.name.trim(),
        expiresInDays,
      },
    }),
  );
}

export async function revokeAccessToken(
  id: number,
  queryClient?: ReturnType<typeof useQueryClient>,
): Promise<void> {
  await unwrapVoid(
    api.DELETE("/users/me/access-tokens/{id}", {
      params: { path: { id } },
    }),
  );
  if (queryClient) {
    await queryClient.invalidateQueries({ queryKey: accessTokenKeys.all });
  }
}

const statusColor: Record<string, "success" | "warning" | "error" | "default"> =
  {
    active: "success",
    expired: "warning",
    revoked: "error",
  };

export function TokenRevealView({
  token,
  onCopy,
}: {
  token: string;
  onCopy?: () => void;
}) {
  const { t } = useTranslation();
  const { token: antdToken } = theme.useToken();

  return (
    <div className="flex flex-col gap-4">
      <Alert
        type="warning"
        showIcon
        title={t("core.accessTokens.copyWarning")}
      />
      <div className="flex items-center gap-2">
        <Input
          readOnly
          value={token}
          style={{ fontFamily: antdToken.fontFamilyCode || "monospace" }}
        />
        <Button icon={<CopyOutlined />} onClick={onCopy}>
          {t("core.accessTokens.copy")}
        </Button>
      </div>
    </div>
  );
}

export default function AccessTokens(_props: Route.ComponentProps) {
  const { data: tokens } = useSuspenseQuery(accessTokensQueryOptions());
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const { token: antdToken } = theme.useToken();
  const { run, pending } = useAction();
  const queryClient = useQueryClient();
  const [form] = Form.useForm<CreateTokenFormValues>();

  const [modalOpen, setModalOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);

  const handleCloseModal = () => {
    setModalOpen(false);
    setCreatedToken(null);
    form.resetFields();
  };

  const handleCreate = async (values: CreateTokenFormValues) => {
    await run(async () => {
      const res = await submitCreateToken(values);
      setCreatedToken(res.token ?? null);
      await queryClient.invalidateQueries({ queryKey: accessTokenKeys.all });
    });
  };

  const handleRevoke = async (id: number) => {
    await run(
      () => revokeAccessToken(id, queryClient),
      "core.accessTokens.revokedSuccess",
    );
  };

  const handleCopy = async () => {
    if (!createdToken) return;
    try {
      await navigator.clipboard.writeText(createdToken);
      message.success(t("core.accessTokens.copied"));
    } catch {
      message.info(createdToken);
    }
  };

  const expiryOptions = ACCESS_TOKEN_EXPIRY_DAYS.map((days) => ({
    value: days,
    label: t(`core.accessTokens.expires${days}Days`, `${days} days`),
  }));

  const columns: TableProps<AccessToken>["columns"] = [
    {
      title: t("core.accessTokens.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string) => <span className="font-medium">{name}</span>,
    },
    {
      title: t("core.accessTokens.prefix"),
      dataIndex: "prefix",
      key: "prefix",
      render: (prefix: string) => (
        <code style={{ fontFamily: antdToken.fontFamilyCode || "monospace" }}>
          {prefix}
        </code>
      ),
    },
    {
      title: t("core.accessTokens.status"),
      dataIndex: "status",
      key: "status",
      render: (status = "active") => {
        const color = statusColor[status] ?? "default";
        const key =
          status === "expired" || status === "revoked"
            ? (`core.accessTokens.${status}` as const)
            : ("core.accessTokens.active" as const);
        return <Tag color={color}>{t(key)}</Tag>;
      },
    },
    {
      title: t("core.accessTokens.expires"),
      dataIndex: "expiresAt",
      key: "expiresAt",
      render: (val?: string) =>
        val ? formatDateTime(val, i18n.language) : t("core.accessTokens.never"),
    },
    {
      title: t("core.accessTokens.created"),
      dataIndex: "createdAt",
      key: "createdAt",
      render: (val?: string) =>
        val ? formatDateTime(val, i18n.language) : "-",
    },
    {
      title: "",
      key: "actions",
      align: "right",
      render: (_, record) => {
        const isActive = record.status === "active";
        return (
          <Popconfirm
            title={t("core.accessTokens.revokeConfirm")}
            okText={t("core.accessTokens.revoke")}
            cancelText={t("common.cancel")}
            disabled={!isActive}
            onConfirm={() => record.id && handleRevoke(record.id)}
          >
            <Button
              type="text"
              danger
              icon={<StopOutlined />}
              disabled={!isActive}
              aria-label={t("core.accessTokens.revoke")}
            >
              {t("core.accessTokens.revoke")}
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <Page
      title={t("core.menu.accessTokens")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalOpen(true)}
        >
          {t("core.accessTokens.new")}
        </Button>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={tokens}
          pagination={false}
          scroll={{ x: "max-content" }}
          locale={{
            emptyText: <Empty description={t("core.accessTokens.empty")} />,
          }}
        />
      </Panel>

      <Modal
        open={modalOpen}
        title={
          createdToken
            ? t("core.accessTokens.modalCreatedTitle")
            : t("core.accessTokens.new")
        }
        onCancel={handleCloseModal}
        footer={
          createdToken ? (
            <Button type="primary" onClick={handleCloseModal}>
              {t("core.accessTokens.close")}
            </Button>
          ) : (
            <>
              <Button onClick={handleCloseModal}>{t("common.cancel")}</Button>
              <Button
                type="primary"
                loading={pending}
                onClick={() => form.submit()}
              >
                {t("core.accessTokens.create")}
              </Button>
            </>
          )
        }
        destroyOnHidden
      >
        {createdToken ? (
          <TokenRevealView token={createdToken} onCopy={handleCopy} />
        ) : (
          <Form
            form={form}
            layout="vertical"
            initialValues={{
              expiresInDays: DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS,
            }}
            onFinish={handleCreate}
          >
            <Form.Item
              name="name"
              label={t("core.accessTokens.name")}
              rules={[
                {
                  required: true,
                  message: t("core.accessTokens.nameRequired"),
                },
              ]}
            >
              <Input
                placeholder={t("core.accessTokens.namePlaceholder")}
                autoFocus
              />
            </Form.Item>
            <Form.Item
              name="expiresInDays"
              label={t("core.accessTokens.expiresIn")}
            >
              <Select options={expiryOptions} />
            </Form.Item>
          </Form>
        )}
      </Modal>
    </Page>
  );
}
