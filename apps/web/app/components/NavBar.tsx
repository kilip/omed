import { AppstoreOutlined, LogoutOutlined } from "@ant-design/icons";
import { Avatar, Button, Dropdown, Select } from "antd";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router";
import { Logo } from "~/components/Logo";
import { LanguageMenu, ThemeMenu } from "~/components/PreferenceMenus";
import { findModule } from "~/lib/modules";
import type { Workspace } from "~/lib/workspace";

type NavBarProps = {
  user: { name: string; email: string; image?: string | null };
  workspaces: Workspace[];
  activeId?: number;
  onWorkspaceChange: (id: number) => void;
  onSignOut: () => void;
};

/**
 * Odoo-style navbar. Left: the app switcher button, the brand and the current
 * app name. Right (the "systray"): workspace, language, theme and user.
 */
export function NavBar({
  user,
  workspaces,
  activeId,
  onWorkspaceChange,
  onSignOut,
}: NavBarProps) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const meta = findModule(pathname);

  return (
    <header
      className="surface flex flex-wrap items-center gap-x-1 px-3"
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
        className="flex items-center gap-2 px-1 font-medium"
        style={{ color: "var(--omed-text)" }}
      >
        <Logo />
        Omed
      </Link>
      {meta ? (
        <span
          className="ml-2 border-0 border-l border-solid pl-3 font-medium"
          style={{
            borderColor: "var(--omed-border)",
            color: "var(--omed-primary)",
          }}
        >
          {t(meta.labelKey)}
        </span>
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
          onChange={onWorkspaceChange}
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
              onClick: onSignOut,
            },
          ],
        }}
      >
        <Button type="text" aria-label={user.name}>
          <Avatar size={24} src={user.image ?? undefined}>
            {user.name.slice(0, 1).toUpperCase()}
          </Avatar>
        </Button>
      </Dropdown>
    </header>
  );
}
