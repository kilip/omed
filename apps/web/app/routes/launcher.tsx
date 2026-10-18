import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { modules } from "~/lib/modules";

const APP_COLORS = 3;

/** Odoo-style home menu: rounded app tiles on a dotted backdrop. */
export default function Launcher() {
  const { t } = useTranslation();

  return (
    <div className="home-backdrop flex flex-1 justify-center">
      <div className="flex w-full max-w-[880px] flex-col gap-6 px-4 py-10">
        <div className="text-center">
          <h1 className="m-0 text-2xl font-medium">{t("launcher.title")}</h1>
          <p style={{ color: "var(--omed-text-secondary)" }}>
            {t("launcher.subtitle")}
          </p>
        </div>
        <div className="grid justify-center gap-2 [grid-template-columns:repeat(auto-fill,minmax(132px,1fr))]">
          {modules.map((m, index) => (
            <Link
              key={m.id}
              to={`/${m.id}`}
              className={`app-tile app-tile--${(index % APP_COLORS) + 1}`}
            >
              <span className="app-tile__icon">{createElement(m.icon)}</span>
              <span className="min-w-0">
                <span className="block font-medium">{t(m.labelKey)}</span>
                <span
                  className="block text-xs"
                  style={{ color: "var(--omed-text-secondary)" }}
                >
                  {t(m.descriptionKey)}
                </span>
              </span>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
