import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import {
  App,
  Button,
  Popconfirm,
  Space,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { requireWorkspaceId } from "~/lib/workspace";
import { mapProviderError } from "../components/provider-form";
import { ProviderTestButton } from "../components/provider-test-button";
import { useAgentPermissions } from "../hooks/use-agent-permissions";
import {
  type Capabilities,
  type ProviderResponse,
  useDeleteProvider,
  useProviders,
  useSetDefaultProvider,
} from "../queries";
import type { Route } from "./+types/providers";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function AgentProvidersPage({
  loaderData,
}: Route.ComponentProps): React.ReactElement {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const wid = loaderData?.wid ?? 0;
  const permissions = useAgentPermissions(wid);
  const { data: providers, isLoading } = useProviders(wid);
  const setDefaultMutation = useSetDefaultProvider(wid);
  const deleteMutation = useDeleteProvider(wid);

  const [settingDefaultId, setSettingDefaultId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const app = App.useApp?.();
  const message = app?.message;

  const handleSetDefault = useCallback(
    async (id: number) => {
      setSettingDefaultId(id);
      try {
        await setDefaultMutation.mutateAsync(id);
        message?.success(t("agent.providers.setDefaultSuccess"));
      } catch (err) {
        const mapped = mapProviderError(err, t);
        message?.error(mapped.message);
      } finally {
        setSettingDefaultId(null);
      }
    },
    [setDefaultMutation, message, t],
  );

  const handleDelete = useCallback(
    async (id: number) => {
      setDeletingId(id);
      try {
        await deleteMutation.mutateAsync(id);
        message?.success(t("agent.providers.deleteSuccess"));
      } catch (err) {
        const mapped = mapProviderError(err, t);
        message?.error(mapped.message);
      } finally {
        setDeletingId(null);
      }
    },
    [deleteMutation, message, t],
  );

  const columns = useMemo<TableProps<ProviderResponse>["columns"]>(() => {
    const cols: TableProps<ProviderResponse>["columns"] = [
      {
        title: t("agent.providers.fields.name"),
        dataIndex: "name",
        key: "name",
        render: (name: string, record: ProviderResponse) => (
          <Space size="small">
            <span style={{ fontWeight: 500, color: "var(--omed-text)" }}>
              {name}
            </span>
            {record.isDefault && (
              <Tag
                style={{
                  background: "var(--omed-tint)",
                  color: "var(--omed-tint-text)",
                  borderColor: "transparent",
                }}
              >
                {t("agent.providers.defaultBadge")}
              </Tag>
            )}
          </Space>
        ),
      },
      {
        title: t("agent.providers.fields.kind"),
        dataIndex: "kind",
        key: "kind",
        render: (kind?: string) => (
          <Tag>
            {kind
              ? t(`agent.providers.kinds.${kind}`, { defaultValue: kind })
              : "-"}
          </Tag>
        ),
      },
      {
        title: t("agent.providers.fields.model"),
        dataIndex: "model",
        key: "model",
        render: (model?: string) => (
          <span style={{ color: "var(--omed-text)" }}>{model ?? "-"}</span>
        ),
      },
      {
        title: t("agent.providers.fields.baseUrl"),
        dataIndex: "baseUrl",
        key: "baseUrl",
        render: (baseUrl?: string) => (
          <span style={{ color: "var(--omed-text-secondary)" }}>
            {baseUrl || "-"}
          </span>
        ),
      },
      {
        title: t("agent.providers.fields.apiKeyHint"),
        dataIndex: "keyHint",
        key: "keyHint",
        render: (keyHint?: string, record?: ProviderResponse) => (
          <span
            style={{
              color: "var(--omed-text-muted)",
              fontFamily: "monospace",
            }}
          >
            {keyHint ? `••••${keyHint}` : record?.hasKey ? "••••" : "-"}
          </span>
        ),
      },
      {
        title: t("agent.providers.fields.capabilities"),
        dataIndex: "capabilities",
        key: "capabilities",
        render: (caps?: Capabilities) => (
          <Space size={[4, 4]} wrap>
            {caps?.streaming && (
              <Tag>{t("agent.providers.capabilities.streaming")}</Tag>
            )}
            {caps?.structuredOutput && (
              <Tag>{t("agent.providers.capabilities.structuredOutput")}</Tag>
            )}
            {caps?.toolCalling && (
              <Tag>{t("agent.providers.capabilities.toolCalling")}</Tag>
            )}
            {caps?.vision && (
              <Tag>{t("agent.providers.capabilities.vision")}</Tag>
            )}
          </Space>
        ),
      },
    ];

    if (permissions.canManage) {
      cols.push({
        title: "",
        key: "actions",
        align: "right",
        render: (_: unknown, record: ProviderResponse) => (
          <Space size="small">
            <ProviderTestButton
              providerId={record.id ?? 0}
              wid={wid}
              size="small"
            />
            {!record.isDefault && (
              <Button
                size="small"
                loading={settingDefaultId === record.id}
                onClick={() => handleSetDefault(record.id ?? 0)}
              >
                {t("agent.providers.actions.setDefault")}
              </Button>
            )}
            <Tooltip title={t("agent.providers.actions.edit")}>
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={() => navigate(`/agent/providers/${record.id}/edit`)}
              >
                {t("agent.providers.actions.edit")}
              </Button>
            </Tooltip>
            <Popconfirm
              title={t("agent.providers.deleteConfirm")}
              onConfirm={() => handleDelete(record.id ?? 0)}
            >
              <Tooltip title={t("agent.providers.actions.delete")}>
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  loading={deletingId === record.id}
                >
                  {t("agent.providers.actions.delete")}
                </Button>
              </Tooltip>
            </Popconfirm>
          </Space>
        ),
      });
    }

    return cols;
  }, [
    t,
    permissions.canManage,
    wid,
    settingDefaultId,
    deletingId,
    handleSetDefault,
    handleDelete,
    navigate,
  ]);

  return (
    <Page
      title={t("agent.providers.title")}
      actions={
        permissions.canManage ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/agent/providers/new")}
          >
            {t("agent.providers.actions.add")}
          </Button>
        ) : undefined
      }
    >
      <Panel>
        <Table
          dataSource={providers}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={false}
        />
      </Panel>
    </Page>
  );
}
