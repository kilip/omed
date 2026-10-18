export type StatusStep = { value: string; label: string };

type StatusBarProps = {
  steps: StatusStep[];
  /** `value` of the current step. */
  current: string;
  label?: string;
};

/** Odoo-style statusbar: a row of pills with the current stage highlighted. */
export function StatusBar({ steps, current, label }: StatusBarProps) {
  return (
    <ol
      aria-label={label}
      className="m-0 flex list-none items-center p-0"
      style={{
        border: "1px solid var(--omed-border)",
        borderRadius: "var(--omed-radius)",
        background: "var(--omed-surface)",
        overflow: "hidden",
      }}
    >
      {steps.map((step, index) => {
        const active = step.value === current;
        return (
          <li
            key={step.value}
            aria-current={active ? "step" : undefined}
            className="px-3 py-1 text-xs font-medium"
            style={{
              background: active ? "var(--omed-primary)" : "transparent",
              color: active
                ? "var(--omed-on-primary)"
                : "var(--omed-text-secondary)",
              borderInlineStart:
                index === 0 ? 0 : "1px solid var(--omed-border-soft)",
            }}
          >
            {step.label}
          </li>
        );
      })}
    </ol>
  );
}
