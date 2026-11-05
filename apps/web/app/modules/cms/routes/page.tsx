import { CopyOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  App,
  Button,
  Form,
  Input,
  Popconfirm,
  Space,
  Tabs,
  Tag,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { ApiError, api, unwrapOne, unwrapVoid } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import { MarkdownEditor } from "../components/markdown-editor";
import { PublishPanel } from "../components/publish-panel";
import { useCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";
import { cmsSettingQueryOptions, pageKeys, pageQueryOptions } from "../queries";
import type { Route } from "./+types/page";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  const settings = await queryClient.ensureQueryData(
    cmsSettingQueryOptions(wid),
  );
  const defaultLocale = settings?.defaultLocale ?? "en";
  await Promise.all([
    queryClient.ensureQueryData(pageQueryOptions(wid, id, defaultLocale)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id, defaultLocale };
}

export default function PageRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id, defaultLocale: loaderDefaultLocale } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditPage, canPublishPage, canDeletePage } = useCmsPermissions(wid);

  const { data: settings } = useQuery(cmsSettingQueryOptions(wid));
  const defaultLocale = settings?.defaultLocale ?? loaderDefaultLocale;
  const supportedLocales = useMemo(
    () => settings?.supportedLocales ?? ["en", "id"],
    [settings?.supportedLocales],
  );

  const [activeLocale, setActiveLocale] = useState<string>(defaultLocale);
  const { data: page } = useQuery(pageQueryOptions(wid, id, activeLocale));

  const [form] = Form.useForm();

  const translations = page?.translations ?? [];
  const currentTranslation = translations.find(
    (tr) => tr.locale === activeLocale,
  );
  const defaultTranslation = translations.find(
    (tr) => tr.locale === defaultLocale,
  );
  const isCreated = Boolean(currentTranslation);

  useEffect(() => {
    form.resetFields();
    if (currentTranslation) {
      form.setFieldsValue({
        title: currentTranslation.title ?? "",
        slug: currentTranslation.slug ?? "",
        body: page?.body ?? "",
      });
    } else {
      form.setFieldsValue({
        title: "",
        slug: "",
        body: "",
      });
    }
  }, [currentTranslation, page, form]);

  const handleTabChange = (key: string) => {
    setActiveLocale(key);
  };

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!isCreated) {
      const slugVal = form.getFieldValue("slug");
      const currentTitle = form.getFieldValue("title") || "";
      if (!slugVal || slugVal === slugify(currentTitle)) {
        form.setFieldValue("slug", slugify(e.target.value));
      }
    }
  };

  const handleCopyFromDefault = () => {
    if (defaultTranslation) {
      form.setFieldsValue({
        title: defaultTranslation.title ?? "",
        slug: `${defaultTranslation.slug ?? ""}-${activeLocale}`,
        body: page?.body ?? "",
      });
      message.info(t("cms.translations.copiedFromDefault"));
    }
  };

  const handleSave = async () => {
    const values = await form.validateFields();
    await run(async () => {
      try {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/cms/pages/{id}/translations/{locale}", {
            params: { path: { wid, id, locale: activeLocale } },
            body: {
              title: values.title,
              slug: values.slug || slugify(values.title),
              body: values.body || "",
            },
          }),
        );
        message.success(t("common.saved"));
        await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
      } catch (err: unknown) {
        if (
          err instanceof ApiError &&
          (err.status === 409 || err.message.includes("SLUG_TAKEN"))
        ) {
          form.setFields([
            {
              name: "slug",
              errors: [t("cms.errors.SLUG_TAKEN")],
            },
          ]);
        } else {
          message.error(t("common.error"));
        }
      }
    });
  };

  const handleDeletePage = async () => {
    await run(async () => {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/cms/pages/{id}", {
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.deleted"));
      await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
      navigate("/cms/pages");
    });
  };

  const handleDeleteTranslation = async () => {
    await run(async () => {
      try {
        await unwrapVoid(
          api.DELETE("/workspaces/{wid}/cms/pages/{id}/translations/{locale}", {
            params: { path: { wid, id, locale: activeLocale } },
          }),
        );
        message.success(t("common.deleted"));
        await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
        setActiveLocale(defaultLocale);
      } catch (err: unknown) {
        const errorObj = err as { code?: string; status?: number };
        if (
          errorObj?.code === "DEFAULT_TRANSLATION_REQUIRED" ||
          errorObj?.status === 422
        ) {
          message.error(t("cms.translations.cannotDeleteDefault"));
        } else {
          message.error(t("common.error"));
        }
      }
    });
  };

  const handlePublish = async () => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/pages/{id}/publish", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
        }),
      );
      message.success(t("cms.publish.publishedAt", { date: "" }));
      await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
    });
  };

  const handleSchedule = async (publishAt: string) => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/pages/{id}/schedule", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
          body: { publishAt },
        }),
      );
      message.success(t("cms.publish.scheduledFor", { date: "" }));
      await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
    });
  };

  const handleUnpublish = async () => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/pages/{id}/unpublish", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
        }),
      );
      message.success(t("common.saved"));
      await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
    });
  };

  const tabItems = supportedLocales.map((loc) => {
    const tr = translations.find((item) => item.locale === loc);
    let badgeColor = "default";
    let badgeText = t("cms.translations.notCreated");

    if (tr) {
      if (tr.effectiveStatus === "PUBLISHED") {
        badgeColor = "success";
        badgeText = t("cms.status.PUBLISHED");
      } else if (tr.effectiveStatus === "SCHEDULED") {
        badgeColor = "processing";
        badgeText = t("cms.status.SCHEDULED");
      } else {
        badgeColor = "warning";
        badgeText = t("cms.status.DRAFT");
      }
    }

    return {
      key: loc,
      label: (
        <span>
          <strong>{loc.toUpperCase()}</strong>{" "}
          <Tag color={badgeColor} style={{ marginLeft: 4, marginRight: 0 }}>
            {badgeText}
          </Tag>
        </span>
      ),
    };
  });

  if (!page) {
    return null;
  }

  return (
    <Page
      title={page.title || t("cms.pages.edit")}
      trail={[
        { title: t("cms.pages.title"), onClick: () => navigate("/cms/pages") },
      ]}
      actions={
        <Space>
          <Button onClick={() => navigate("/cms/pages")}>
            {t("common.cancel")}
          </Button>
          {isCreated && activeLocale !== defaultLocale && (
            <Popconfirm
              title={t("cms.translations.deleteTranslationConfirm")}
              disabled={!canDeletePage}
              onConfirm={handleDeleteTranslation}
            >
              <Button
                danger
                disabled={!canDeletePage}
                icon={<DeleteOutlined />}
              >
                {t("cms.translations.deleteTranslation")}
              </Button>
            </Popconfirm>
          )}
          <Popconfirm
            title={t("cms.pages.deleteConfirm")}
            disabled={!canDeletePage}
            onConfirm={handleDeletePage}
          >
            <Button danger disabled={!canDeletePage} icon={<DeleteOutlined />}>
              {t("common.delete")}
            </Button>
          </Popconfirm>
          <Button
            type="primary"
            loading={pending}
            disabled={!canEditPage}
            onClick={handleSave}
          >
            {t("common.save")}
          </Button>
        </Space>
      }
    >
      <Panel>
        <Tabs
          activeKey={activeLocale}
          onChange={handleTabChange}
          items={tabItems}
          style={{ marginBottom: 16 }}
        />

        {!isCreated && (
          <Alert
            type="info"
            title={
              <Space>
                <span>{t("cms.translations.noTranslationPrompt")}</span>
                {defaultTranslation && (
                  <Button
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={handleCopyFromDefault}
                  >
                    {t("cms.translations.copyFromDefault")}
                  </Button>
                )}
              </Space>
            }
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}

        <Form form={form} layout="vertical">
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 300px",
              gap: 24,
            }}
          >
            <div>
              <Form.Item
                name="title"
                label={t("cms.pages.titleField")}
                rules={[{ required: true }]}
              >
                <Input
                  size="large"
                  disabled={!canEditPage}
                  onChange={handleTitleChange}
                  placeholder={t("cms.pages.titleField")}
                />
              </Form.Item>

              <Form.Item
                name="slug"
                label={t("cms.pages.slugField")}
                extra={t("cms.pages.slugHint")}
              >
                <Input disabled={!canEditPage} placeholder="page-slug" />
              </Form.Item>

              <Form.Item name="body" label={t("cms.pages.bodyField")}>
                <MarkdownEditor rows={16} disabled={!canEditPage} />
              </Form.Item>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <PublishPanel
                status={currentTranslation?.status ?? "DRAFT"}
                effectiveStatus={currentTranslation?.effectiveStatus ?? "DRAFT"}
                publishAt={currentTranslation?.publishAt}
                disabled={!canPublishPage || !isCreated}
                loading={pending}
                onPublish={handlePublish}
                onSchedule={handleSchedule}
                onUnpublish={handleUnpublish}
              />
            </div>
          </div>
        </Form>
      </Panel>
    </Page>
  );
}
