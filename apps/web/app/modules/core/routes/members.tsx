import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Avatar,
  Button,
  Popconfirm,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { moduleLabel } from "~/lib/roles";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import type { Member } from "../lib";
import { membersQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/members";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(membersQueryOptions(wid)),
    queryClient.ensureQueryData(rolesQueryOptions(wid)),
  ]);
  return { wid };
}

export default function Members({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { data: members } = useSuspenseQuery(membersQueryOptions(wid));
  const { label: roleName } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const { run } = useAction();

  const remove = (member: Member) =>
    run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/members/{id}", {
            params: { path: { wid, id: member.user?.id ?? 0 } },
          }),
        ),
      "common.deleted",
    );

  const columns: TableProps<Member>["columns"] = [
    {
      title: t("core.members.user"),
      render: (_, m) => (
        <div className="flex items-center gap-2">
          <Avatar size={24} src={m.user?.avatar}>
            {m.user?.name?.slice(0, 1).toUpperCase()}
          </Avatar>
          <div className="leading-tight">
            <div>{m.user?.name}</div>
            <div
              className="text-xs"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {m.user?.email}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: t("core.members.roles"),
      render: (_, m) => (
        <div className="flex flex-wrap gap-1">
          {(m.roles ?? []).map((role) => {
            const mod = role.split(":")[0];
            const modName = moduleLabel(t, mod);
            return (
              <Tag key={role}>
                <span style={{ color: "var(--omed-text-secondary)" }}>
                  {modName}:
                </span>{" "}
                {roleName(role)}
              </Tag>
            );
          })}
        </div>
      ),
    },
    {
      title: t("core.members.joined"),
      align: "right",
      render: (_, m) => formatDate(m.createdAt, i18n.language),
    },
    {
      title: "",
      align: "right",
      render: (_, m) => (
        <div className="flex justify-end gap-1">
          <Tooltip title={t("core.members.editRoles")}>
            <Link to={`/core/members/${m.user?.id}`}>
              <Button
                type="text"
                icon={<EditOutlined />}
                aria-label={t("core.members.editRoles")}
              />
            </Link>
          </Tooltip>
          <Popconfirm
            title={t("core.members.removeConfirm")}
            okText={t("core.members.remove")}
            cancelText={t("common.cancel")}
            onConfirm={() => remove(m)}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              aria-label={t("core.members.remove")}
            />
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <Page
      title={t("core.menu.members")}
      actions={
        <Link to="/core/members/new">
          <Button type="primary" icon={<PlusOutlined />}>
            {t("core.members.invite")}
          </Button>
        </Link>
      }
    >
      <Panel strong>
        <Table
          size="small"
          rowKey={(m) => m.id ?? m.user?.email ?? ""}
          columns={columns}
          dataSource={members}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>
    </Page>
  );
}
