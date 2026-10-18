import type { ReactNode } from "react";
import { Panel } from "./Panel";

type FormViewProps = {
  /** Heading of the sheet, usually the record's number or name. */
  title: ReactNode;
  /** Action buttons, shown on the left above the sheet. */
  buttons?: ReactNode;
  /** A `StatusBar`, shown on the right above the sheet. */
  statusbar?: ReactNode;
  /** A `Chatter`. Sits beside the sheet on wide screens, below it otherwise. */
  chatter?: ReactNode;
  children: ReactNode;
};

/**
 * Odoo-style form view: action buttons and statusbar above a white sheet, with
 * the chatter next to it.
 */
export function FormView({
  title,
  buttons,
  statusbar,
  chatter,
  children,
}: FormViewProps) {
  return (
    <div className="flex flex-wrap items-start gap-3">
      <div className="flex min-w-0 flex-[3_1_480px] flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex flex-wrap gap-2">{buttons}</div>
          {statusbar}
        </div>
        <Panel strong>
          <h2 className="mb-3 mt-0 text-lg font-medium">{title}</h2>
          {children}
        </Panel>
      </div>
      {chatter ? (
        <aside
          className="min-w-0"
          style={{ flex: "1 1 var(--omed-chatter-width)" }}
        >
          {chatter}
        </aside>
      ) : null}
    </div>
  );
}
