import { Button, Form, Input } from "antd";
import { useTranslation } from "react-i18next";
import { redirect, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { getActiveWorkspace } from "~/lib/workspace";
import { RoleSelector } from "../components/role-selector";
import { useRoles } from "../hooks/use-roles";
import { rolesQueryOptions } from "../queries";
import type { Route } from "./+types/member-new";

export async function clientLoader() {
  const workspace = await getActiveWorkspace();
  if (workspace.personal) {
    throw redirect("/core/members");
  }
  const wid = workspace.id;
  const queryClient = getQueryClient();
  await queryClient.ensureQueryData(rolesQueryOptions(wid));
  return { wid };
}

type FormValues = {
  email: string;
  roles: string[];
};

export default function MemberNew({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { roles } = useRoles(wid);
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { run, pending } = useAction();
  const [form] = Form.useForm<FormValues>();

  const submit = (values: FormValues) =>
    run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/members", {
          params: { path: { wid } },
          body: { email: values.email, roles: values.roles },
        }),
      );
      navigate("/core/members");
    }, "common.saved");

  return (
    <Page title={t("core.members.invite")}>
      <Form
        form={form}
        layout="vertical"
        size="small"
        initialValues={{ roles: [] }}
        onFinish={submit}
      >
        <div className="flex flex-col gap-3 max-w-4xl">
          <Panel strong title={t("core.members.invite")}>
            <Form.Item
              name="email"
              label={t("common.email")}
              rules={[{ required: true, type: "email" }]}
            >
              <Input placeholder="user@example.com" />
            </Form.Item>
          </Panel>

          <Form.Item
            name="roles"
            label={t("core.members.roles")}
            rules={[{ required: true, type: "array", min: 1 }]}
          >
            <RoleSelector roles={roles} disabled={pending} />
          </Form.Item>

          <div className="flex items-center gap-2">
            <Button type="primary" htmlType="submit" loading={pending}>
              {t("common.save")}
            </Button>
            <Button onClick={() => navigate("/core/members")}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </Form>
    </Page>
  );
}
