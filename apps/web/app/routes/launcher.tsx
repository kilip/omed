import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { modules } from "~/lib/modules";

export default function Launcher() {
  const { t } = useTranslation();

  return (
    <div className="mx-auto flex w-full max-w-[1040px] flex-col gap-6 px-4 py-10">
      <div>
        <h1 className="m-0 text-2xl font-medium">{t("launcher.title")}</h1>
        <p style={{ color: "var(--omed-text-secondary)" }}>
          {t("launcher.subtitle")}
        </p>
      </div>
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(220px,1fr))]">
        {modules.map((m) => (
          <Link
            key={m.id}
            to={`/${m.id}`}
            className="surface-muted flex min-w-0 flex-col gap-2 p-3"
            style={{
              borderRadius: "var(--omed-radius-lg)",
              color: "var(--omed-text)",
            }}
          >
            <span
              className="flex h-11 w-11 items-center justify-center text-[22px]"
              style={{
                borderRadius: "var(--omed-radius-lg)",
                background: "var(--omed-tint)",
                color: "var(--omed-tint-text)",
              }}
            >
              {createElement(m.icon)}
            </span>
            <div>
              <div className="font-medium">{t(m.labelKey)}</div>
              <div
                className="text-xs"
                style={{ color: "var(--omed-text-secondary)" }}
              >
                {t(m.descriptionKey)}
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
