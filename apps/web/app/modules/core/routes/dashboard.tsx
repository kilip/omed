import { PlusOutlined } from "@ant-design/icons";
import { Button, Statistic, Table, type TableProps, Tag } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate, useRevalidator } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, orForbidden, unwrapList, unwrapOne } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import {
  requireWorkspaceId,
  setActiveWorkspaceId,
  type Workspace,
} from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import { invitationStatus } from "../lib";
import { rolesQueryOptions } from "../queries";
import type { Route } from "./+types/dashboard";

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

export default function CoreDashboard({ loaderData }: Route.ComponentProps) {
  const { workspaces, members, invitations, permissions, wid } = loaderData;
  const { label: roleName } = useRoles(wid);
  const { t } = useTranslation();
  const navigate = useNavigate();
  const revalidator = useRevalidator();

  const activeInvitations = invitations.filter(
    (i) => invitationStatus(i) === "active",
  );

  const columns: TableProps<Workspace>["columns"] = [
    { title: t("common.name"), dataIndex: "name" },
    { title: t("core.workspaces.owner"), render: (_, w) => w.owner?.name },
    {
      title: t("common.type"),
      render: (_, w) =>
        w.personal ? (
          <Tag>{t("core.workspaces.personal")}</Tag>
        ) : (
          <Tag variant="filled">{t("core.workspaces.shared")}</Tag>
        ),
    },
    {
      title: "",
      align: "right",
      render: (_, w) =>
        w.id === wid ? (
          <Tag color="success">{t("common.active")}</Tag>
        ) : (
          <Button
            size="small"
            onClick={(e) => {
              e.stopPropagation();
              if (w.id) {
                setActiveWorkspaceId(w.id);
                void revalidator.revalidate();
              }
            }}
          >
            {t("core.workspaces.switch")}
          </Button>
        ),
    },
  ];

  return (
    <Page title={t("core.menu.dashboard")}>
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        <Panel>
          <Statistic
            title={t("core.dashboard.workspaces")}
            value={workspaces.length}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("core.dashboard.members")}
            value={members.length}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("core.dashboard.invitations")}
            value={activeInvitations.length}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("core.dashboard.roles")}
            value={permissions.roles?.length ?? 0}
          />
        </Panel>
      </div>

      <div className="flex flex-wrap items-start gap-3">
        <Panel
          strong
          title={t("core.dashboard.yourWorkspaces")}
          extra={
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => navigate("/core/workspaces/new")}
            >
              {t("core.workspaces.new")}
            </Button>
          }
          className="flex-[2_1_520px]"
        >
          <Table
            size="small"
            rowKey="id"
            columns={columns}
            dataSource={workspaces}
            pagination={false}
            scroll={{ x: "max-content" }}
            onRow={(record) => ({
              onClick: () => navigate(`/core/workspaces/${record.id}`),
              className: "cursor-pointer",
            })}
          />
        </Panel>

        <div className="flex min-w-0 flex-[1_1_300px] flex-col gap-3">
          <Panel strong title={t("core.dashboard.access")}>
            <div className="mb-2 flex flex-wrap gap-1">
              {permissions.owner ? (
                <Tag color="success">{t("core.dashboard.owner")}</Tag>
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
