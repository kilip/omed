import { MenuFoldOutlined, MenuUnfoldOutlined } from "@ant-design/icons";
import { Button, Grid, Menu } from "antd";
import { createElement, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation } from "react-router";
import { modules } from "~/lib/modules";

const SIDER_STORAGE_KEY = "omed.sider";

function readCollapsed(): boolean | undefined {
  try {
    const stored = localStorage.getItem(SIDER_STORAGE_KEY);
    if (stored === "collapsed") return true;
    if (stored === "expanded") return false;
  } catch {
    // Storage can be unavailable (private mode). Fall back to the default.
  }
  return undefined;
}

/**
 * Odoo-style app sidebar (from the module's `module.ts`) next to the module's
 * routes. It collapses to icons, and starts collapsed on narrow screens.
 */
export function ModuleLayout({ moduleId }: { moduleId: string }) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const screens = Grid.useBreakpoint();
  const [preference, setPreference] = useState<boolean | undefined>();
  const meta = modules.find((m) => m.id === moduleId);

  useEffect(() => {
    setPreference(readCollapsed());
  }, []);

  if (!meta) return <Outlet />;

  const collapsed = preference ?? screens.md === false;
  const toggle = () => {
    const next = !collapsed;
    setPreference(next);
    try {
      localStorage.setItem(SIDER_STORAGE_KEY, next ? "collapsed" : "expanded");
    } catch {
      // Ignore storage failures; the choice still applies this session.
    }
  };

  const base = `/${meta.id}`;
  const selected = meta.menu
    .filter((item) =>
      item.path
        ? pathname === `${base}/${item.path}` ||
          pathname.startsWith(`${base}/${item.path}/`)
        : pathname === base,
    )
    .map((item) => item.key);

  return (
    <div className="flex flex-1">
      <aside
        className="surface flex shrink-0 flex-col px-2 py-2"
        style={{
          width: collapsed
            ? "var(--omed-sider-width-collapsed)"
            : "var(--omed-sider-width)",
          borderBlock: 0,
          borderInlineStart: 0,
          borderRadius: 0,
        }}
      >
        <div
          className="mb-2 flex items-center gap-2 px-2 font-medium"
          style={{ color: "var(--omed-primary)" }}
        >
          <span className="flex text-base">{createElement(meta.icon)}</span>
          {collapsed ? null : (
            <span className="truncate">{t(meta.labelKey)}</span>
          )}
        </div>
        <Menu
          mode="inline"
          inlineCollapsed={collapsed}
          selectedKeys={selected}
          style={{ background: "transparent", borderInlineEnd: 0 }}
          items={meta.menu.map((item) => ({
            key: item.key,
            icon: item.icon ? createElement(item.icon) : undefined,
            label: (
              <Link to={item.path ? `${base}/${item.path}` : base}>
                {t(item.labelKey)}
              </Link>
            ),
          }))}
        />
        <span className="flex-1" />
        <Button
          type="text"
          className="self-start"
          icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          aria-label={t(collapsed ? "shell.expandMenu" : "shell.collapseMenu")}
          aria-expanded={!collapsed}
          onClick={toggle}
        />
      </aside>
      <main className="min-w-0 flex-1">
        <Outlet />
      </main>
    </div>
  );
}
