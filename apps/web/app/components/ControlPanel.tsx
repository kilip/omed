import { Breadcrumb } from "antd";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router";
import { findModule } from "~/lib/modules";
import { SearchBar, type SearchBarProps } from "./SearchBar";
import { ViewSwitcher, type ViewSwitcherProps } from "./ViewSwitcher";

export type Crumb = { title: string; onClick?: () => void };

export type ControlPanelProps = {
  /** Title of the current page, shown as the last breadcrumb. */
  title: string;
  /** Extra breadcrumbs between the module and the title (e.g. a list view). */
  trail?: Crumb[];
  /** Buttons next to the breadcrumb (e.g. "New"). */
  actions?: ReactNode;
  search?: SearchBarProps;
  views?: ViewSwitcherProps;
  /** Record counter shown before the view switcher. */
  count?: number;
};

/**
 * Odoo-style control panel. Top row: breadcrumb and actions on the left, the
 * search bar in the middle, the counter and view switcher on the right.
 */
export function ControlPanel({
  title,
  trail = [],
  actions,
  search,
  views,
  count,
}: ControlPanelProps) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const meta = findModule(pathname);

  const items = [
    ...(meta
      ? [{ title: <Link to={`/${meta.id}`}>{t(meta.labelKey)}</Link> }]
      : []),
    ...trail.map((crumb) => ({
      title: (
        <button
          type="button"
          onClick={crumb.onClick}
          className="cursor-pointer border-0 bg-transparent p-0"
          style={{ color: "var(--omed-primary-text)", font: "inherit" }}
        >
          {crumb.title}
        </button>
      ),
    })),
    { title: <span style={{ color: "var(--omed-text)" }}>{title}</span> },
  ];

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 pt-3">
      <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
        <Breadcrumb items={items} />
        {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
      </div>
      <div className="flex min-w-[220px] flex-[1_1_260px] justify-center">
        {search ? <SearchBar {...search} /> : null}
      </div>
      {views || count !== undefined ? (
        <div className="flex items-center gap-3">
          {count !== undefined ? (
            <span
              className="num text-xs"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {t("common.records", { count })}
            </span>
          ) : null}
          {views ? <ViewSwitcher {...views} /> : null}
        </div>
      ) : null}
    </div>
  );
}
