import { DeleteOutlined } from "@ant-design/icons";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { App, Button, Form, Input, Popconfirm, Space } from "antd";
import { useEffect } from "react";
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
import { pageKeys, pageQueryOptions } from "../queries";
import type { Route } from "./+types/page";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(pageQueryOptions(wid, id)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid, id };
}

export default function PageRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canEditPage, canPublishPage, canDeletePage } = useCmsPermissions(wid);

  const { data: page } = useSuspenseQuery(pageQueryOptions(wid, id));

  const [form] = Form.useForm();

  useEffect(() => {
    form.setFieldsValue({
      title: page.title ?? "",
      slug: page.slug ?? "",
      body: page.body ?? "",
    });
  }, [page, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    await run(async () => {
      try {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/cms/pages/{id}", {
            params: { path: { wid, id } },
            body: {
              title: values.title,
              slug: values.slug,
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

  const handleDelete = async () => {
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

  const handlePublish = async () => {
    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/pages/{id}/publish", {
          params: { path: { wid, id } },
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
          params: { path: { wid, id } },
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
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.saved"));
      await queryClient.invalidateQueries({ queryKey: pageKeys.all(wid) });
    });
  };

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
          <Popconfirm
            title={t("cms.pages.deleteConfirm")}
            disabled={!canDeletePage}
            onConfirm={handleDelete}
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
                  placeholder={t("cms.pages.titleField")}
                />
              </Form.Item>

              <Form.Item
                name="slug"
                label={t("cms.pages.slugField")}
                extra={t("cms.pages.slugHint")}
                rules={[{ required: true }]}
              >
                <Input disabled={!canEditPage} placeholder="page-slug" />
              </Form.Item>

              <Form.Item name="body" label={t("cms.pages.bodyField")}>
                <MarkdownEditor rows={16} disabled={!canEditPage} />
              </Form.Item>
            </div>

            <div>
              <PublishPanel
                status={page.status}
                effectiveStatus={page.effectiveStatus}
                publishAt={page.publishAt}
                disabled={!canPublishPage}
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
