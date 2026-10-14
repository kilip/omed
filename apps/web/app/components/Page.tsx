import { Breadcrumb } from "antd";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router";
import { findModule } from "~/lib/modules";

type PageProps = {
  /** Title of the current page, shown as the last breadcrumb. */
  title: string;
  actions?: ReactNode;
  children: ReactNode;
};

/** Odoo-style control panel (breadcrumb + actions) above the page content. */
export function Page({ title, actions, children }: PageProps) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const meta = findModule(pathname);

  const items = [
    ...(meta
      ? [{ title: <Link to={`/${meta.id}`}>{t(meta.labelKey)}</Link> }]
      : []),
    { title: <span style={{ color: "var(--omed-text)" }}>{title}</span> },
  ];

  return (
    <div className="flex min-w-0 flex-col gap-3 px-4 pb-6">
      <div className="flex flex-wrap items-center justify-between gap-2 pt-3">
        <Breadcrumb items={items} />
        {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
      </div>
      {children}
    </div>
  );
}
