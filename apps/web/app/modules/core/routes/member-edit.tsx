import { useSuspenseQuery } from "@tanstack/react-query";
import { Avatar, Button, Form } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { RoleSelector } from "../components/role-selector";
import { useRoles } from "../hooks/use-roles";
import { membersQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/member-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(membersQueryOptions(wid)),
    queryClient.ensureQueryData(rolesQueryOptions(wid)),
  ]);
  return { wid, id: Number(params.id) };
}

type FormValues = {
  roles: string[];
};

export default function MemberEdit({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { data: members } = useSuspenseQuery(membersQueryOptions(wid));
  const { roles } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { run, pending } = useAction();
  const [form] = Form.useForm<FormValues>();

  const member = members.find((m) => m.user?.id === id);

  if (!member) {
    return (
      <Page title={t("core.members.editRoles")}>
        <Panel strong>
          <div className="py-4 text-center">
            <p className="mb-3" style={{ color: "var(--omed-text-secondary)" }}>
              {t("common.error")}
            </p>
            <Button onClick={() => navigate("/core/members")}>
              {t("common.cancel")}
            </Button>
          </div>
        </Panel>
      </Page>
    );
  }

  const submit = (values: FormValues) =>
    run(async () => {
      await unwrapOne(
        api.PUT("/workspaces/{wid}/members/{id}", {
          params: { path: { wid, id } },
          body: { roles: values.roles },
        }),
      );
      navigate("/core/members");
    }, "common.saved");

  return (
    <Page title={t("core.members.editRoles")}>
      <Form
        form={form}
        layout="vertical"
        size="small"
        initialValues={{ roles: member.roles ?? [] }}
        onFinish={submit}
      >
        <div className="flex flex-col gap-3 max-w-4xl">
          <Panel strong title={t("core.members.user")}>
            <div className="flex items-center gap-3 py-1">
              <Avatar size={40} src={member.user?.avatar}>
                {member.user?.name?.slice(0, 1).toUpperCase()}
              </Avatar>
              <div className="leading-tight">
                <div className="text-base font-medium">{member.user?.name}</div>
                <div
                  className="text-xs"
                  style={{ color: "var(--omed-text-secondary)" }}
                >
                  {member.user?.email}
                </div>
                {member.createdAt ? (
                  <div
                    className="text-xs mt-1"
                    style={{ color: "var(--omed-text-muted)" }}
                  >
                    {t("core.members.joined")}:{" "}
                    {formatDate(member.createdAt, i18n.language)}
                  </div>
                ) : null}
              </div>
            </div>
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
