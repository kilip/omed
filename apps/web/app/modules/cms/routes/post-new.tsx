import { PictureOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { App, Button, Card, Form, Input, Space } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { ApiError, api, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { MarkdownEditor } from "../components/markdown-editor";
import { MediaPicker } from "../components/media-picker";
import { useCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";
import type { Route } from "./+types/post-new";

type MediaItem = components["schemas"]["model.MediaResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function PostNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { run, pending } = useAction();
  const { canCreatePost } = useCmsPermissions(wid);

  const [form] = Form.useForm();
  const [slugTouched, setSlugTouched] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [coverMedia, setCoverMedia] = useState<MediaItem | null>(null);

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    if (!slugTouched) {
      form.setFieldValue("slug", slugify(val));
    }
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    await run(async () => {
      try {
        const created = await unwrapOne(
          api.POST("/workspaces/{wid}/cms/posts", {
            params: { path: { wid } },
            body: {
              title: values.title,
              slug: values.slug || slugify(values.title),
              excerpt: values.excerpt || undefined,
              body: values.body || "",
              coverMediaId: coverMedia?.id,
            },
          }),
        );
        message.success(t("common.saved"));
        navigate(`/cms/posts/${created.id}`);
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

  return (
    <Page
      title={t("cms.posts.new")}
      trail={[
        { title: t("cms.posts.title"), onClick: () => navigate("/cms/posts") },
      ]}
      actions={
        <Space>
          <Button onClick={() => navigate("/cms/posts")}>
            {t("common.cancel")}
          </Button>
          <Button
            type="primary"
            loading={pending}
            disabled={!canCreatePost}
            onClick={handleSubmit}
          >
            {t("common.save")}
          </Button>
        </Space>
      }
    >
      <Panel>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            title: "",
            slug: "",
            excerpt: "",
            body: "",
          }}
        >
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 280px",
              gap: 24,
            }}
          >
            <div>
              <Form.Item
                name="title"
                label={t("cms.posts.titleField")}
                rules={[{ required: true, message: t("common.error") }]}
              >
                <Input
                  size="large"
                  placeholder={t("cms.posts.titleField")}
                  onChange={handleTitleChange}
                />
              </Form.Item>

              <Form.Item
                name="slug"
                label={t("cms.posts.slugField")}
                extra={t("cms.posts.slugHint")}
              >
                <Input
                  placeholder="post-slug"
                  onChange={() => setSlugTouched(true)}
                />
              </Form.Item>

              <Form.Item name="excerpt" label={t("cms.posts.excerptField")}>
                <Input.TextArea
                  rows={3}
                  placeholder={t("cms.posts.excerptField")}
                />
              </Form.Item>

              <Form.Item name="body" label={t("cms.posts.bodyField")}>
                <MarkdownEditor rows={16} />
              </Form.Item>
            </div>

            <div>
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
                    <Space>
                      <Button size="small" onClick={() => setPickerOpen(true)}>
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
                  </div>
                ) : (
                  <div style={{ textAlign: "center", padding: "16px 0" }}>
                    <Button
                      icon={<PictureOutlined />}
                      onClick={() => setPickerOpen(true)}
                    >
                      {t("cms.posts.selectCover")}
                    </Button>
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
