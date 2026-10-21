import { DeleteOutlined, PictureOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { App, Button, Card, Form, Input, Popconfirm, Space } from "antd";
import { useEffect, useState } from "react";
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
import { postKeys, postQueryOptions } from "../queries";
import type { Route } from "./+types/post";

type MediaItem = components["schemas"]["model.MediaResponse"];

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(postQueryOptions(wid, id)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id };
}

export default function PostRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditPost, canPublishPost, canDeletePost } = useCmsPermissions(wid);

  const { data: post } = useSuspenseQuery(postQueryOptions(wid, id));

  const [form] = Form.useForm();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [coverMedia, setCoverMedia] = useState<MediaItem | null>(
    post.coverImage ?? null,
  );

  useEffect(() => {
    form.setFieldsValue({
      title: post.title ?? "",
      slug: post.slug ?? "",
      excerpt: post.excerpt ?? "",
      body: post.body ?? "",
    });
    setCoverMedia(post.coverImage ?? null);
  }, [post, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    await run(async () => {
      try {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/cms/posts/{id}", {
            params: { path: { wid, id } },
            body: {
              title: values.title,
              slug: values.slug,
              excerpt: values.excerpt || undefined,
              body: values.body || "",
              coverMediaId: coverMedia?.id,
            },
          }),
        );
        message.success(t("common.saved"));
        await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
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

  const handleDelete = async () => {
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

  const handlePublish = async () => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/posts/{id}/publish", {
          params: { path: { wid, id } },
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
          params: { path: { wid, id } },
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
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.saved"));
      await queryClient.invalidateQueries({ queryKey: postKeys.all(wid) });
    });
  };

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
          <Popconfirm
            title={t("cms.posts.deleteConfirm")}
            disabled={!canDeletePost}
            onConfirm={handleDelete}
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
                  placeholder={t("cms.posts.titleField")}
                />
              </Form.Item>

              <Form.Item
                name="slug"
                label={t("cms.posts.slugField")}
                extra={t("cms.posts.slugHint")}
                rules={[{ required: true }]}
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
                status={post.status}
                effectiveStatus={post.effectiveStatus}
                publishAt={post.publishAt}
                disabled={!canPublishPost}
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
