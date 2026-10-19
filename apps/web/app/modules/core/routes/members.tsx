import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Avatar,
  Button,
  Empty,
  Popconfirm,
  Table,
  type TableProps,
  Tag,
  Tooltip,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import type { ViewType } from "~/components/ViewSwitcher";
import { api, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { moduleLabel } from "~/lib/roles";
import { useAction } from "~/lib/use-action";
import { getActiveWorkspace } from "~/lib/workspace";
import { useRoles } from "../hooks/use-roles";
import type { Member } from "../lib";
import { membersQueryOptions, rolesQueryOptions } from "../queries";
import type { Route } from "./+types/members";

export async function clientLoader() {
  const workspace = await getActiveWorkspace();
  if (workspace.personal) {
    return { wid: workspace.id, isPersonal: true };
  }
  const wid = workspace.id;
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(membersQueryOptions(wid)),
    queryClient.ensureQueryData(rolesQueryOptions(wid)),
  ]);
  return { wid, isPersonal: false };
}

export default function Members({ loaderData }: Route.ComponentProps) {
  const { t } = useTranslation();
  if (loaderData.isPersonal) {
    return (
      <Page title={t("core.menu.members")}>
        <Panel strong>
          <Empty description={t("core.members.personalNotice")} />
        </Panel>
      </Page>
    );
  }
  return <MembersContent wid={loaderData.wid} />;
}

function MembersContent({ wid }: { wid: number }) {
  const { data: members } = useSuspenseQuery(membersQueryOptions(wid));
  const { label: roleName } = useRoles(wid);
  const { t, i18n } = useTranslation();
  const { run } = useAction();
  const [view, setView] = useState<ViewType>("list");
  const [query, setQuery] = useState("");

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return members;
    return members.filter(
      (m) =>
        m.user?.name?.toLowerCase().includes(needle) ||
        m.user?.email?.toLowerCase().includes(needle),
    );
  }, [members, query]);

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

  const identity = (m: Member) => (
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
  );

  const roleTags = (m: Member) => (
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
  );

  const actions = (m: Member) => (
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
  );

  const columns: TableProps<Member>["columns"] = [
    { title: t("core.members.user"), render: (_, m) => identity(m) },
    { title: t("core.members.roles"), render: (_, m) => roleTags(m) },
    {
      title: t("core.members.joined"),
      align: "right",
      render: (_, m) => formatDate(m.createdAt, i18n.language),
    },
    { title: "", align: "right", render: (_, m) => actions(m) },
  ];

  const rowKey = (m: Member) => m.id ?? m.user?.email ?? "";

  return (
    <Page
      title={t("core.menu.members")}
      count={visible.length}
      search={{
        value: query,
        onChange: setQuery,
        placeholder: t("core.members.search"),
      }}
      views={{ value: view, onChange: setView, views: ["list", "kanban"] }}
      actions={
        <Link to="/core/members/new">
          <Button type="primary" icon={<PlusOutlined />}>
            {t("core.members.invite")}
          </Button>
        </Link>
      }
    >
      {view === "kanban" ? (
        <div className="grid gap-2 [grid-template-columns:repeat(auto-fill,minmax(240px,1fr))]">
          {visible.map((m) => (
            <Panel strong key={rowKey(m)}>
              <div className="flex flex-col gap-2">
                {identity(m)}
                {roleTags(m)}
                <div className="flex items-center justify-between">
                  <span
                    className="text-xs"
                    style={{ color: "var(--omed-text-secondary)" }}
                  >
                    {formatDate(m.createdAt, i18n.language)}
                  </span>
                  {actions(m)}
                </div>
              </div>
            </Panel>
          ))}
        </div>
      ) : (
        <Panel strong>
          <Table
            size="small"
            rowKey={rowKey}
            columns={columns}
            dataSource={visible}
            pagination={false}
            scroll={{ x: "max-content" }}
          />
        </Panel>
      )}
    </Page>
  );
}
