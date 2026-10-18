import type { ReactNode } from "react";
import { ControlPanel, type ControlPanelProps } from "./ControlPanel";

type PageProps = ControlPanelProps & {
  children: ReactNode;
};

/** Odoo-style page: a control panel above the page content. */
export function Page({ children, ...controlPanel }: PageProps) {
  return (
    <div className="flex min-w-0 flex-col gap-3 px-4 pb-6">
      <ControlPanel {...controlPanel} />
      {children}
    </div>
  );
}
