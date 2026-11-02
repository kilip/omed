import { useQuery } from "@tanstack/react-query";
import { Alert, App, Button, Form, Select, Switch } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { requireWorkspaceId } from "~/lib/workspace";
import { useAgentPermissions } from "../hooks/use-agent-permissions";
import {
  settingsQueryOptions,
  type UpdateSettingRequest,
  useProviders,
  useUpdateSettings,
} from "../queries";
import type { Route } from "./+types/settings";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export type SettingsFormValues = {
  financialEnabled: boolean;
  financialProviderId?: number | null;
};

export function buildProviderOptions(
  providers: { id?: number; name?: string; isDefault?: boolean }[] | undefined,
  t: (k: string) => string,
): { value: number; label: string }[] {
  const list: { value: number; label: string }[] = [
    {
      value: 0,
      label: t("agent.financial.defaultProvider"),
    },
  ];
  if (providers) {
    for (const p of providers) {
      if (p.id) {
        list.push({
          value: p.id,
          label: p.isDefault
            ? `${p.name ?? ""} (${t("agent.providers.defaultBadge")})`
            : (p.name ?? ""),
        });
      }
    }
  }
  return list;
}

export function buildUpdateSettingsPayload(
  values: SettingsFormValues,
): UpdateSettingRequest {
  return {
    financialEnabled: Boolean(values.financialEnabled),
    financialProviderId:
      typeof values.financialProviderId === "number" &&
      values.financialProviderId > 0
        ? values.financialProviderId
        : null,
  };
}

export function mapSettingsError(
  err: unknown,
  t: (k: string) => string,
): { code?: string; message: string } {
  const errorObj = (typeof err === "object" && err !== null ? err : {}) as {
    status?: number;
    code?: string;
    message?: string;
  };

  const code = errorObj.code;
  const status = errorObj.status;

  if (code === "FINANCIAL_AGENT_NO_PROVIDER" || status === 422) {
    return {
      code: "FINANCIAL_AGENT_NO_PROVIDER",
      message: t("agent.financial.errors.noProvider"),
    };
  }

  return {
    code,
    message:
      errorObj.message ||
      (err instanceof Error
        ? err.message
        : t("agent.financial.errors.generic")),
  };
}

export default function AgentSettingsPage({
  loaderData,
}: Route.ComponentProps): React.ReactElement {
  const { t } = useTranslation();
  const wid = loaderData?.wid ?? 0;
  const permissions = useAgentPermissions(wid);

  const { data: settings } = useQuery({
    ...settingsQueryOptions(wid),
    enabled: Boolean(wid) && permissions.canManageSettings,
  });
  const { data: providers } = useProviders(wid);
  const updateMutation = useUpdateSettings(wid);

  const [form] = Form.useForm<SettingsFormValues>();
  const [formError, setFormError] = useState<string | null>(null);

  const app = App.useApp?.();
  const message = app?.message;

  useEffect(() => {
    if (settings) {
      form.setFieldsValue({
        financialEnabled: settings.financialEnabled ?? false,
        financialProviderId: settings.financialProviderId ?? 0,
      });
    }
  }, [settings, form]);

  const providerOptions = useMemo(
    () => buildProviderOptions(providers, t),
    [providers, t],
  );

  const handleSave = async () => {
    try {
      setFormError(null);
      const values = await form.validateFields();
      const payload = buildUpdateSettingsPayload(values);
      await updateMutation.mutateAsync(payload);
      message?.success(t("agent.financial.saved"));
    } catch (err: unknown) {
      if (err && typeof err === "object" && "errorFields" in err) {
        return;
      }
      const mapped = mapSettingsError(err, t);
      setFormError(mapped.message);
      if (mapped.code === "FINANCIAL_AGENT_NO_PROVIDER") {
        form.setFieldValue("financialEnabled", false);
      }
    }
  };

  return (
    <Page title={t("agent.settings.title")}>
      <Panel>
        <Alert
          type="info"
          showIcon
          message={t("agent.financial.privacyNotice")}
          style={{ marginBottom: 24 }}
        />
        {formError && (
          <Alert
            type="error"
            showIcon
            message={formError}
            style={{ marginBottom: 24 }}
          />
        )}
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            financialEnabled: settings?.financialEnabled ?? false,
            financialProviderId: settings?.financialProviderId ?? 0,
          }}
        >
          <Form.Item
            name="financialEnabled"
            label={t("agent.financial.enabled")}
            valuePropName="checked"
            extra={t("agent.financial.enabledHelp")}
          >
            <Switch disabled={!permissions.canManageSettings} />
          </Form.Item>

          <Form.Item
            name="financialProviderId"
            label={t("agent.financial.provider")}
          >
            <Select
              disabled={!permissions.canManageSettings}
              options={providerOptions}
              placeholder={t("agent.financial.defaultProvider")}
            />
          </Form.Item>

          {permissions.canManageSettings && (
            <Form.Item style={{ marginBottom: 0, marginTop: 16 }}>
              <Button
                type="primary"
                loading={updateMutation.isPending}
                onClick={handleSave}
              >
                {t("agent.financial.save")}
              </Button>
            </Form.Item>
          )}
        </Form>
      </Panel>
    </Page>
  );
}
