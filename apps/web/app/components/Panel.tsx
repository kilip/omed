import type { ReactNode } from "react";

type PanelProps = {
  title?: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  /** Dense (white) surface for tables and forms. Muted surface otherwise. */
  strong?: boolean;
  className?: string;
};

export function Panel({
  title,
  extra,
  children,
  strong = false,
  className = "",
}: PanelProps) {
  return (
    <section
      className={`${strong ? "surface" : "surface-muted"} min-w-0 p-2.5 ${className}`}
      style={{ borderRadius: "var(--omed-radius-lg)" }}
    >
      {title || extra ? (
        <header className="mb-2 flex items-center justify-between gap-2">
          <h2 className="m-0 text-base font-medium">{title}</h2>
          {extra}
        </header>
      ) : null}
      {children}
    </section>
  );
}
