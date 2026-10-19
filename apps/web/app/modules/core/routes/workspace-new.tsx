import { Button, Form, Input } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { useAction } from "~/lib/use-action";
import { refreshWorkspaces, setActiveWorkspaceId } from "~/lib/workspace";

type FormValues = {
  name: string;
  description?: string;
};

export default function WorkspaceNew() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { run, pending } = useAction();
  const [form] = Form.useForm<FormValues>();

  const submit = (values: FormValues) =>
    run(async () => {
      const created = await unwrapOne(
        api.POST("/workspaces", {
          body: {
            name: values.name.trim(),
            description: values.description?.trim() || undefined,
          },
        }),
      );
      if (created?.id) {
        setActiveWorkspaceId(created.id);
        await refreshWorkspaces();
      }
      navigate("/core");
    }, "common.saved");

  return (
    <Page
      title={t("core.workspaces.new")}
      trail={[
        {
          title: t("core.menu.dashboard"),
          onClick: () => navigate("/core"),
        },
      ]}
    >
      <Form form={form} layout="vertical" size="small" onFinish={submit}>
        <div className="flex max-w-4xl flex-col gap-3">
          <Panel strong title={t("core.workspaces.new")}>
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

          <div className="flex items-center gap-2">
            <Button type="primary" htmlType="submit" loading={pending}>
              {t("common.save")}
            </Button>
            <Button onClick={() => navigate("/core")}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </Form>
    </Page>
  );
}
