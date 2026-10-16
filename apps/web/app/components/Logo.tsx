export function Logo({ size = 20 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <circle
        cx="12"
        cy="12"
        r="7"
        fill="none"
        stroke="var(--omed-brand)"
        strokeWidth="5"
      />
      <path
        d="M12 5A7 7 0 0 1 19 12"
        fill="none"
        stroke="var(--omed-brand-accent)"
        strokeWidth="5"
      />
    </svg>
  );
}
