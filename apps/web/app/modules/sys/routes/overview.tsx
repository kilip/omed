import { PlusOutlined } from "@ant-design/icons";
import { Button, Statistic, Table, type TableProps, Tag } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, orForbidden, unwrapList, unwrapOne } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId, type Workspace } from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import { invitationStatus } from "../lib";
import { rolesQueryOptions } from "../queries";
import type { Route } from "./+types/overview";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const path = { wid };

  const [workspaces, members, invitations, permissions] = await Promise.all([
    unwrapList(api.POST("/workspaces/search", { body: { limit: 100 } })),
    unwrapList(
      api.POST("/workspaces/{wid}/members/search", {
        params: { path },
        body: { limit: 100 },
      }),
    ).catch(orForbidden([])),
    unwrapList(
      api.POST("/workspaces/{wid}/invitations/search", {
        params: { path },
        body: { limit: 100 },
      }),
    ).catch(orForbidden([])),
    unwrapOne(api.GET("/workspaces/{wid}/permissions", { params: { path } })),
    // Warms the roles query that useRoles reads for the role labels.
    getQueryClient().ensureQueryData(rolesQueryOptions(wid)),
  ]);

  return { workspaces, members, invitations, permissions, wid };
}

export default function Overview({ loaderData }: Route.ComponentProps) {
  const { workspaces, members, invitations, permissions, wid } = loaderData;
  const { label: roleName } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();

  const activeInvitations = invitations.filter(
    (i) => invitationStatus(i) === "active",
  );

  const columns: TableProps<Workspace>["columns"] = [
    { title: t("common.name"), dataIndex: "name" },
    { title: t("sys.workspaces.owner"), render: (_, w) => w.owner?.name },
    {
      title: t("common.type"),
      render: (_, w) =>
        w.personal ? (
          <Tag>{t("sys.workspaces.personal")}</Tag>
        ) : (
          <Tag variant="filled">{t("sys.workspaces.shared")}</Tag>
        ),
    },
    {
      title: t("common.created"),
      align: "right",
      render: (_, w) => formatDate(w.createdAt, i18n.language),
    },
  ];

  return (
    <Page
      title={t("sys.menu.overview")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate("/sys/workspaces")}
        >
          {t("sys.workspaces.new")}
        </Button>
      }
    >
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        <Panel>
          <Statistic
            title={t("sys.overview.workspaces")}
            value={workspaces.length}
          />
        </Panel>
        <Panel>
          <Statistic title={t("sys.overview.members")} value={members.length} />
        </Panel>
        <Panel>
          <Statistic
            title={t("sys.overview.invitations")}
            value={activeInvitations.length}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("sys.overview.roles")}
            value={permissions.roles?.length ?? 0}
          />
        </Panel>
      </div>

      <div className="flex flex-wrap items-start gap-3">
        <Panel
          strong
          title={t("sys.overview.yourWorkspaces")}
          className="flex-[2_1_520px]"
        >
          <Table
            size="small"
            rowKey="id"
            columns={columns}
            dataSource={workspaces}
            pagination={false}
            scroll={{ x: "max-content" }}
          />
        </Panel>

        <div className="flex min-w-0 flex-[1_1_300px] flex-col gap-3">
          <Panel strong title={t("sys.overview.access")}>
            <div className="mb-2 flex flex-wrap gap-1">
              {permissions.owner ? (
                <Tag color="success">{t("sys.overview.owner")}</Tag>
              ) : null}
              {(permissions.roles ?? []).map((role) => (
                <Tag key={role}>{roleName(role)}</Tag>
              ))}
            </div>
            <div className="flex flex-col gap-1 text-[13px]">
              {(permissions.permissions ?? []).map((p) => (
                <div
                  key={`${p.object}:${p.action}`}
                  className="flex justify-between gap-2"
                >
                  <code>{p.object}</code>
                  <span style={{ color: "var(--omed-text-secondary)" }}>
                    {p.action}
                  </span>
                </div>
              ))}
            </div>
          </Panel>
        </div>
      </div>
    </Page>
  );
}
