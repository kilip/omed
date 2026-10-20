import { AppstoreOutlined, LogoutOutlined } from "@ant-design/icons";
import { Avatar, Button, Dropdown, Menu, type MenuProps, Select } from "antd";
import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router";
import { Logo } from "~/components/Logo";
import { LanguageMenu, ThemeMenu } from "~/components/PreferenceMenus";
import { findModule, type ModuleMenuItem } from "~/lib/modules";
import type { Workspace } from "~/lib/workspace";

type MenuItemType = Required<MenuProps>["items"][number];

function filterMenuItems(
  items: ModuleMenuItem[],
  isPersonal?: boolean,
): ModuleMenuItem[] {
  return items
    .filter((item) => !(item.hideInPersonalWorkspace && isPersonal))
    .map((item) => {
      if (item.children) {
        return {
          ...item,
          children: filterMenuItems(item.children, isPersonal),
        };
      }
      return item;
    });
}

function getSelectedKeys(
  items: ModuleMenuItem[],
  base: string,
  pathname: string,
): string[] {
  const selected: string[] = [];
  for (const item of items) {
    if (item.children && item.children.length > 0) {
      selected.push(...getSelectedKeys(item.children, base, pathname));
    } else if (item.path !== undefined) {
      const matches = item.path
        ? pathname === `${base}/${item.path}` ||
          pathname.startsWith(`${base}/${item.path}/`)
        : pathname === base;
      if (matches) {
        selected.push(item.key);
      }
    }
  }
  return selected;
}

function mapToMenuItems(
  items: ModuleMenuItem[],
  base: string,
  t: (key: string) => string,
): MenuItemType[] {
  return items.map((item) => {
    if (item.children && item.children.length > 0) {
      return {
        key: item.key,
        icon: item.icon ? createElement(item.icon) : undefined,
        label: t(item.labelKey),
        children: mapToMenuItems(item.children, base, t),
      };
    }
    return {
      key: item.key,
      icon: item.icon ? createElement(item.icon) : undefined,
      label: (
        <Link to={item.path ? `${base}/${item.path}` : base}>
          {t(item.labelKey)}
        </Link>
      ),
    };
  });
}

type NavBarProps = {
  user: { name: string; email: string; image?: string | null };
  workspaces: Workspace[];
  activeId?: number;
  onWorkspaceChange: (id: number) => void;
  onSignOut: () => void;
};

/**
 * Odoo-style navbar. Left: the app switcher button, the brand, the current
 * app name and its menu. Right (the "systray"): workspace, language, theme
 * and user.
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
  const base = meta ? `/${meta.id}` : "";

  const currentWorkspace = workspaces.find((w) => w.id === activeId);
  const visibleMenuItems = filterMenuItems(
    meta?.menu ?? [],
    currentWorkspace?.personal,
  );

  const selected = getSelectedKeys(visibleMenuItems, base, pathname);

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
        <>
          <Link
            to={base}
            className="ml-2 flex items-center gap-1.5 border-0 border-l border-solid pl-3 pr-2 font-medium"
            style={{
              borderColor: "var(--omed-border)",
              color: "var(--omed-primary-text)",
            }}
          >
            <span className="flex text-base">{createElement(meta.icon)}</span>
            <span>{t(meta.labelKey)}</span>
          </Link>
          <Menu
            mode="horizontal"
            selectedKeys={selected}
            style={{
              background: "transparent",
              borderBottom: 0,
              flex: 1,
              minWidth: 0,
            }}
            items={mapToMenuItems(visibleMenuItems, base, t)}
          />
        </>
      ) : (
        <span className="flex-1" />
      )}
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
