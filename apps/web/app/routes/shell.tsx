import { AppstoreOutlined, LogoutOutlined } from "@ant-design/icons";
import { Avatar, Button, Dropdown, Select } from "antd";
import { useTranslation } from "react-i18next";
import {
  Link,
  Outlet,
  redirect,
  useLocation,
  useNavigate,
  useRevalidator,
} from "react-router";
import { Logo } from "~/components/Logo";
import { LanguageMenu, ThemeMenu } from "~/components/PreferenceMenus";
import { authClient } from "~/lib/auth-client";
import { findModule } from "~/lib/modules";
import {
  refreshWorkspaces,
  requireWorkspaceId,
  setActiveWorkspaceId,
  type Workspace,
} from "~/lib/workspace";
import type { Route } from "./+types/shell";

export async function clientLoader() {
  const { data } = await authClient.getSession();
  if (!data) throw redirect("/login");

  let workspaces: Workspace[] = [];
  let activeId: number | undefined;
  let apiError: string | undefined;
  try {
    workspaces = await refreshWorkspaces();
    activeId = await requireWorkspaceId();
  } catch (error) {
    apiError = error instanceof Error ? error.message : String(error);
  }

  return { user: data.user, workspaces, activeId, apiError };
}

export default function Shell({ loaderData }: Route.ComponentProps) {
  const { user, workspaces, activeId } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const revalidator = useRevalidator();
  const { pathname } = useLocation();
  const meta = findModule(pathname);

  const signOut = async () => {
    await authClient.signOut();
    navigate("/login");
  };

  return (
    <div className="flex min-h-screen flex-col">
      <header
        className="glass-strong flex flex-wrap items-center gap-x-2 px-3"
        style={{
          minHeight: "var(--omed-header-height)",
          borderRadius: 0,
          borderInline: 0,
          borderTop: 0,
        }}
      >
        <Link to="/" aria-label={t("shell.apps")}>
          <Button type="text" icon={<AppstoreOutlined />} />
        </Link>
        <Link
          to="/"
          className="flex items-center gap-2 font-medium"
          style={{ color: "var(--omed-text)" }}
        >
          <Logo />
          Omed
        </Link>
        {meta ? (
          <span className="px-2 font-medium">{t(meta.labelKey)}</span>
        ) : null}
        <span className="flex-1" />
        {workspaces.length > 0 ? (
          <Select
            aria-label={t("shell.workspace")}
            value={activeId}
            style={{ minWidth: 180 }}
            options={workspaces.map((w) => ({
              value: w.id,
              label: w.personal ? t("shell.personalWorkspace") : w.name,
            }))}
            onChange={(id) => {
              setActiveWorkspaceId(id);
              void revalidator.revalidate();
            }}
          />
        ) : null}
        <LanguageMenu />
        <ThemeMenu />
        <Dropdown
          trigger={["click"]}
          menu={{
            items: [
              {
                key: "me",
                disabled: true,
                label: (
                  <div className="leading-tight">
                    <div>{user.name}</div>
                    <div className="text-xs">{user.email}</div>
                  </div>
                ),
              },
              { type: "divider" },
              {
                key: "out",
                icon: <LogoutOutlined />,
                label: t("shell.signOut"),
                onClick: signOut,
              },
            ],
          }}
        >
          <Button type="text" aria-label={user.name}>
            <Avatar size={22} src={user.image ?? undefined}>
              {user.name.slice(0, 1).toUpperCase()}
            </Avatar>
          </Button>
        </Dropdown>
      </header>
      <div className="flex flex-1 flex-col">
        <Outlet />
      </div>
    </div>
  );
}
