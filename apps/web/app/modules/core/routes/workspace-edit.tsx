import { Button, Form, Input, Popconfirm } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, orForbidden, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import {
  getActiveWorkspace,
  getStoredWorkspaceId,
  refreshWorkspaces,
  setActiveWorkspaceId,
} from "~/lib/workspace";
import type { Route } from "./+types/workspace-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = Number(params.id);
  const path = { wid };
  const [workspace, permissions] = await Promise.all([
    unwrapOne(api.GET("/workspaces/{wid}", { params: { path } })),
    unwrapOne(
      api.GET("/workspaces/{wid}/permissions", { params: { path } }),
    ).catch(orForbidden({ roles: [], permissions: [], owner: false })),
  ]);

  return { workspace, permissions, wid };
}

type FormValues = {
  name: string;
  description?: string;
};

export default function WorkspaceEdit({ loaderData }: Route.ComponentProps) {
  const { workspace, permissions, wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { run, pending } = useAction();
  const [form] = Form.useForm<FormValues>();

  if (!workspace) {
    return (
      <Page title={t("core.workspaces.edit")}>
        <Panel strong>
          <div className="py-4 text-center">
            <p className="mb-3" style={{ color: "var(--omed-text-secondary)" }}>
              {t("common.error")}
            </p>
            <Button onClick={() => navigate("/core")}>
              {t("common.cancel")}
            </Button>
          </div>
        </Panel>
      </Page>
    );
  }

  const canDelete = !workspace.personal && Boolean(permissions?.owner);

  const submit = (values: FormValues) =>
    run(async () => {
      await unwrapOne(
        api.PUT("/workspaces/{wid}", {
          params: { path: { wid } },
          body: {
            name: values.name.trim(),
            description: values.description?.trim() || undefined,
          },
        }),
      );
      await refreshWorkspaces();
      navigate("/core");
    }, "common.saved");

  const remove = () =>
    run(async () => {
      await unwrapOne(
        api.DELETE("/workspaces/{wid}", {
          params: { path: { wid } },
        }),
      );
      await refreshWorkspaces();
      if (getStoredWorkspaceId() === wid) {
        const active = await getActiveWorkspace();
        setActiveWorkspaceId(active.id);
      }
      navigate("/core");
    }, "common.deleted");

  return (
    <Page
      title={t("core.workspaces.edit")}
      trail={[
        {
          title: t("core.menu.dashboard"),
          onClick: () => navigate("/core"),
        },
      ]}
    >
      <Form
        form={form}
        layout="vertical"
        size="small"
        initialValues={{
          name: workspace.name,
          description: workspace.description ?? undefined,
        }}
        onFinish={submit}
      >
        <div className="flex max-w-4xl flex-col gap-3">
          <Panel strong title={t("core.workspaces.edit")}>
            <Form.Item
              name="name"
              label={t("common.name")}
              rules={[{ required: true, whitespace: true }]}
            >
              <Input placeholder={t("core.workspaces.namePlaceholder")} />
            </Form.Item>
            <Form.Item name="description" label={t("common.description")}>
              <Input.TextArea
                rows={4}
                placeholder={t("core.workspaces.descriptionPlaceholder")}
              />
            </Form.Item>
          </Panel>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Button type="primary" htmlType="submit" loading={pending}>
                {t("common.save")}
              </Button>
              <Button onClick={() => navigate("/core")}>
                {t("common.cancel")}
              </Button>
            </div>

            {canDelete ? (
              <Popconfirm
                title={t("core.workspaces.deleteConfirm")}
                okText={t("common.yes")}
                cancelText={t("common.no")}
                okButtonProps={{ danger: true, loading: pending }}
                onConfirm={remove}
              >
                <Button danger loading={pending}>
                  {t("common.delete")}
                </Button>
              </Popconfirm>
            ) : null}
          </div>
        </div>
      </Form>
    </Page>
  );
}
