import {
  CopyOutlined,
  DeleteOutlined,
  PictureOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  App,
  Button,
  Card,
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
import { MediaPicker } from "../components/media-picker";
import { PublishPanel } from "../components/publish-panel";
import { useCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";
import { cmsSettingQueryOptions, postKeys, postQueryOptions } from "../queries";
import type { Route } from "./+types/post";

type MediaItem = components["schemas"]["model.MediaResponse"];

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  const settings = await queryClient.ensureQueryData(
    cmsSettingQueryOptions(wid),
  );
  const defaultLocale = settings?.defaultLocale ?? "en";
  await Promise.all([
    queryClient.ensureQueryData(postQueryOptions(wid, id, defaultLocale)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id, defaultLocale };
}

export default function PostRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id, defaultLocale: loaderDefaultLocale } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditPost, canPublishPost, canDeletePost } = useCmsPermissions(wid);

  const { data: settings } = useQuery(cmsSettingQueryOptions(wid));
  const defaultLocale = settings?.defaultLocale ?? loaderDefaultLocale;
  const supportedLocales = useMemo(
    () => settings?.supportedLocales ?? ["en", "id"],
    [settings?.supportedLocales],
  );

  const [activeLocale, setActiveLocale] = useState<string>(defaultLocale);
  const { data: post } = useQuery(postQueryOptions(wid, id, activeLocale));

  const [form] = Form.useForm();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [coverMedia, setCoverMedia] = useState<MediaItem | null>(null);

  const translations = post?.translations ?? [];
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
        excerpt: post?.excerpt ?? "",
        body: post?.body ?? "",
      });
    } else {
      form.setFieldsValue({
        title: "",
        slug: "",
        excerpt: "",
        body: "",
      });
    }
    setCoverMedia(post?.coverImage ?? null);
  }, [currentTranslation, post, form]);

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
        excerpt: post?.excerpt ?? "",
        body: post?.body ?? "",
      });
      message.info(t("cms.translations.copiedFromDefault"));
    }
  };

  const handleSave = async () => {
    const values = await form.validateFields();
    await run(async () => {
      try {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/cms/posts/{id}/translations/{locale}", {
            params: { path: { wid, id, locale: activeLocale } },
            body: {
              title: values.title,
              slug: values.slug || slugify(values.title),
              excerpt: values.excerpt || undefined,
              body: values.body || "",
            },
          }),
        );

        if ((coverMedia?.id ?? null) !== (post?.coverMediaId ?? null)) {
          await unwrapOne(
            api.PUT("/workspaces/{wid}/cms/posts/{id}", {
              params: { path: { wid, id } },
              body: {
                coverMediaId: coverMedia?.id,
              },
            }),
          );
        }

        message.success(t("common.saved"));
        await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
      } catch (err: unknown) {
        console.error("handleSave error:", err);
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

  const handleDeletePost = async () => {
    await run(async () => {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/cms/posts/{id}", {
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.deleted"));
      await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
      navigate("/cms/posts");
    });
  };

  const handleDeleteTranslation = async () => {
    await run(async () => {
      try {
        await unwrapVoid(
          api.DELETE("/workspaces/{wid}/cms/posts/{id}/translations/{locale}", {
            params: { path: { wid, id, locale: activeLocale } },
          }),
        );
        message.success(t("common.deleted"));
        await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
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
        api.POST("/workspaces/{wid}/cms/posts/{id}/publish", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
        }),
      );
      message.success(t("cms.publish.publishedAt", { date: "" }));
      await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
    });
  };

  const handleSchedule = async (publishAt: string) => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/posts/{id}/schedule", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
          body: { publishAt },
        }),
      );
      message.success(t("cms.publish.scheduledFor", { date: "" }));
      await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
    });
  };

  const handleUnpublish = async () => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/posts/{id}/unpublish", {
          params: { path: { wid, id }, query: { locale: activeLocale } },
        }),
      );
      message.success(t("common.saved"));
      await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
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

  if (!post) {
    return null;
  }

  return (
    <Page
      title={post.title || t("cms.posts.edit")}
      trail={[
        { title: t("cms.posts.title"), onClick: () => navigate("/cms/posts") },
      ]}
      actions={
        <Space>
          <Button onClick={() => navigate("/cms/posts")}>
            {t("common.cancel")}
          </Button>
          {isCreated && activeLocale !== defaultLocale && (
            <Popconfirm
              title={t("cms.translations.deleteTranslationConfirm")}
              disabled={!canDeletePost}
              onConfirm={handleDeleteTranslation}
            >
              <Button
                danger
                disabled={!canDeletePost}
                icon={<DeleteOutlined />}
              >
                {t("cms.translations.deleteTranslation")}
              </Button>
            </Popconfirm>
          )}
          <Popconfirm
            title={t("cms.posts.deleteConfirm")}
            disabled={!canDeletePost}
            onConfirm={handleDeletePost}
          >
            <Button danger disabled={!canDeletePost} icon={<DeleteOutlined />}>
              {t("common.delete")}
            </Button>
          </Popconfirm>
          <Button
            type="primary"
            loading={pending}
            disabled={!canEditPost}
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
                label={t("cms.posts.titleField")}
                rules={[{ required: true }]}
              >
                <Input
                  size="large"
                  disabled={!canEditPost}
                  onChange={handleTitleChange}
                  placeholder={t("cms.posts.titleField")}
                />
              </Form.Item>

              <Form.Item
                name="slug"
                label={t("cms.posts.slugField")}
                extra={t("cms.posts.slugHint")}
              >
                <Input disabled={!canEditPost} placeholder="post-slug" />
              </Form.Item>

              <Form.Item name="excerpt" label={t("cms.posts.excerptField")}>
                <Input.TextArea
                  rows={3}
                  disabled={!canEditPost}
                  placeholder={t("cms.posts.excerptField")}
                />
              </Form.Item>

              <Form.Item name="body" label={t("cms.posts.bodyField")}>
                <MarkdownEditor rows={16} disabled={!canEditPost} />
              </Form.Item>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <PublishPanel
                status={currentTranslation?.status ?? "DRAFT"}
                effectiveStatus={currentTranslation?.effectiveStatus ?? "DRAFT"}
                publishAt={currentTranslation?.publishAt}
                disabled={!canPublishPost || !isCreated}
                loading={pending}
                onPublish={handlePublish}
                onSchedule={handleSchedule}
                onUnpublish={handleUnpublish}
              />

              <Card title={t("cms.posts.coverImage")} size="small">
                {coverMedia ? (
                  <div style={{ textAlign: "center" }}>
                    <div
                      style={{
                        height: 140,
                        background: "var(--omed-surface-muted)",
                        borderRadius: 4,
                        overflow: "hidden",
                        marginBottom: 12,
                      }}
                    >
                      {coverMedia.url && (
                        <img
                          src={coverMedia.url}
                          alt={coverMedia.altText || coverMedia.title || ""}
                          style={{
                            width: "100%",
                            height: "100%",
                            objectFit: "cover",
                          }}
                        />
                      )}
                    </div>
                    {canEditPost && (
                      <Space>
                        <Button
                          size="small"
                          onClick={() => setPickerOpen(true)}
                        >
                          {t("cms.posts.selectCover")}
                        </Button>
                        <Button
                          size="small"
                          danger
                          onClick={() => setCoverMedia(null)}
                        >
                          {t("cms.posts.removeCover")}
                        </Button>
                      </Space>
                    )}
                  </div>
                ) : (
                  <div style={{ textAlign: "center", padding: "16px 0" }}>
                    {canEditPost ? (
                      <Button
                        icon={<PictureOutlined />}
                        onClick={() => setPickerOpen(true)}
                      >
                        {t("cms.posts.selectCover")}
                      </Button>
                    ) : (
                      <span
                        style={{
                          fontSize: 12,
                          color: "var(--omed-text-muted)",
                        }}
                      >
                        -
                      </span>
                    )}
                  </div>
                )}
              </Card>
            </div>
          </div>
        </Form>

        <MediaPicker
          open={pickerOpen}
          wid={wid}
          selectedId={coverMedia?.id}
          onClose={() => setPickerOpen(false)}
          onSelect={(media) => {
            setCoverMedia(media);
          }}
        />
      </Panel>
    </Page>
  );
}
