import { Menu } from "antd";
import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation } from "react-router";
import { modules } from "~/lib/modules";

/** Sider menu (from the module's `module.ts`) next to the module's routes. */
export function ModuleLayout({ moduleId }: { moduleId: string }) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const meta = modules.find((m) => m.id === moduleId);
  if (!meta) return <Outlet />;

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
    <div className="flex flex-1 flex-wrap">
      <aside
        className="glass-strong shrink-0 px-2 py-3"
        style={{
          width: "var(--omed-sider-width)",
          borderBlock: 0,
          borderInlineStart: 0,
          borderRadius: 0,
        }}
      >
        <Menu
          mode="inline"
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
      </aside>
      <main className="min-w-0 flex-[999_1_560px]">
        <Outlet />
      </main>
    </div>
  );
}
