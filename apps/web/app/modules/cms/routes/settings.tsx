import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, App, Button, Form, Select, Typography } from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { requireWorkspaceId } from "~/lib/workspace";
import { useCmsPermissions } from "../hooks/use-cms-permissions";
import { cmsSettingQueryOptions, settingKeys } from "../queries";
import type { Route } from "./+types/settings";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export type CmsSettingsFormValues = {
  defaultLocale: string;
  supportedLocales: string[];
};

export const COMMON_LOCALES = [
  { value: "en", label: "English (en)" },
  { value: "id", label: "Bahasa Indonesia (id)" },
  { value: "ja", label: "Japanese (ja)" },
  { value: "es", label: "Spanish (es)" },
  { value: "fr", label: "French (fr)" },
  { value: "de", label: "German (de)" },
  { value: "zh", label: "Chinese (zh)" },
];

export default function CmsSettingsPage({
  loaderData,
}: Route.ComponentProps): React.ReactElement {
  const { t } = useTranslation();
  const wid = loaderData?.wid ?? 0;
  const permissions = useCmsPermissions(wid);
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: settings, isLoading } = useQuery(cmsSettingQueryOptions(wid));
  const [form] = Form.useForm<CmsSettingsFormValues>();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const supportedLocalesWatch =
    Form.useWatch("supportedLocales", form) || settings?.supportedLocales || [];

  useEffect(() => {
    if (settings) {
      form.setFieldsValue({
        defaultLocale: settings.defaultLocale,
        supportedLocales: settings.supportedLocales,
      });
    }
  }, [settings, form]);

  const mutation = useMutation({
    mutationFn: async (values: CmsSettingsFormValues) => {
      return unwrapOne(
        api.PUT("/workspaces/{wid}/cms/settings", {
          params: { path: { wid } },
          body: {
            defaultLocale: values.defaultLocale,
            supportedLocales: values.supportedLocales,
          },
        }),
      );
    },
    onSuccess: () => {
      message.success(t("cms.settings.savedSuccess"));
      setErrorMessage(null);
      queryClient.invalidateQueries({ queryKey: settingKeys.all(wid) });
    },
    onError: (err: unknown) => {
      const errorObj = err as { code?: string; message?: string };
      const code = errorObj?.code;
      if (code === "DEFAULT_LOCALE_CHANGE_BLOCKED") {
        setErrorMessage(t("cms.settings.errors.defaultLocaleChangeBlocked"));
      } else if (code === "LOCALE_IN_USE") {
        setErrorMessage(t("cms.settings.errors.localeInUse"));
      } else {
        setErrorMessage(
          errorObj?.message || t("cms.settings.errors.saveFailed"),
        );
      }
    },
  });

  const onFinish = (values: CmsSettingsFormValues) => {
    setErrorMessage(null);
    mutation.mutate(values);
  };

  const defaultLocaleOptions = supportedLocalesWatch.map((loc: string) => {
    const found = COMMON_LOCALES.find((c) => c.value === loc);
    return {
      value: loc,
      label: found ? found.label : loc,
    };
  });

  const canEdit = permissions.isAdmin || permissions.isOwner;

  return (
    <Page title={t("cms.settings.title")}>
      <div style={{ maxWidth: 640 }}>
        {errorMessage && (
          <Alert
            type="error"
            title={errorMessage}
            showIcon
            closable
            onClose={() => setErrorMessage(null)}
            style={{ marginBottom: 16 }}
          />
        )}

        <Panel title={t("cms.settings.panelTitle")}>
          <Typography.Paragraph type="secondary">
            {t("cms.settings.description")}
          </Typography.Paragraph>

          <Form
            form={form}
            layout="vertical"
            onFinish={onFinish}
            disabled={!canEdit || isLoading}
          >
            <Form.Item
              name="supportedLocales"
              label={t("cms.settings.supportedLocalesLabel")}
              rules={[
                {
                  required: true,
                  message: t("cms.settings.supportedLocalesRequired"),
                },
              ]}
              extra={t("cms.settings.supportedLocalesHelp")}
            >
              <Select
                mode="tags"
                placeholder={t("cms.settings.supportedLocalesPlaceholder")}
                options={COMMON_LOCALES}
              />
            </Form.Item>

            <Form.Item
              name="defaultLocale"
              label={t("cms.settings.defaultLocaleLabel")}
              rules={[
                {
                  required: true,
                  message: t("cms.settings.defaultLocaleRequired"),
                },
              ]}
              extra={t("cms.settings.defaultLocaleHelp")}
            >
              <Select
                options={defaultLocaleOptions}
                placeholder={t("cms.settings.defaultLocalePlaceholder")}
              />
            </Form.Item>

            {canEdit && (
              <Form.Item style={{ marginBottom: 0 }}>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={mutation.isPending}
                >
                  {t("common.save")}
                </Button>
              </Form.Item>
            )}
          </Form>
        </Panel>
      </div>
    </Page>
  );
}
