import { App, Button, Form, Input, Space } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { ApiError, api, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { MarkdownEditor } from "../components/markdown-editor";
import { useCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";
import type { Route } from "./+types/page-new";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function PageNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { run, pending } = useAction();
  const { canCreatePage } = useCmsPermissions(wid);

  const [form] = Form.useForm();
  const [slugTouched, setSlugTouched] = useState(false);

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
          api.POST("/workspaces/{wid}/cms/pages", {
            params: { path: { wid } },
            body: {
              title: values.title,
              slug: values.slug || slugify(values.title),
              body: values.body || "",
            },
          }),
        );
        message.success(t("common.saved"));
        navigate(`/cms/pages/${created.id}`);
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
      title={t("cms.pages.new")}
      trail={[
        { title: t("cms.pages.title"), onClick: () => navigate("/cms/pages") },
      ]}
      actions={
        <Space>
          <Button onClick={() => navigate("/cms/pages")}>
            {t("common.cancel")}
          </Button>
          <Button
            type="primary"
            loading={pending}
            disabled={!canCreatePage}
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
            body: "",
          }}
        >
          <Form.Item
            name="title"
            label={t("cms.pages.titleField")}
            rules={[{ required: true, message: t("common.error") }]}
          >
            <Input
              size="large"
              placeholder={t("cms.pages.titleField")}
              onChange={handleTitleChange}
            />
          </Form.Item>

          <Form.Item
            name="slug"
            label={t("cms.pages.slugField")}
            extra={t("cms.pages.slugHint")}
          >
            <Input
              placeholder="page-slug"
              onChange={() => setSlugTouched(true)}
            />
          </Form.Item>

          <Form.Item name="body" label={t("cms.pages.bodyField")}>
            <MarkdownEditor rows={16} />
          </Form.Item>
        </Form>
      </Panel>
    </Page>
  );
}
