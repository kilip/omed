import type { ReactNode } from "react";

type PanelProps = {
  title?: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  /** Dense surface for tables and forms. Thin glass otherwise. */
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
      className={`${strong ? "glass-strong" : "glass"} min-w-0 p-3 ${className}`}
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
