import { useTranslation } from "react-i18next";

export type RevenuePoint = { month: string; revenue: number; expense: number };

const WIDTH = 560;
const HEIGHT = 220;
const PAD = { top: 12, right: 8, bottom: 24, left: 44 };
const STEP = 10_000_000;

export function RevenueChart({ data }: { data: RevenuePoint[] }) {
  const { t, i18n } = useTranslation();

  const peak = Math.max(...data.flatMap((d) => [d.revenue, d.expense]), STEP);
  const max = Math.ceil(peak / STEP) * STEP;
  const plotW = WIDTH - PAD.left - PAD.right;
  const plotH = HEIGHT - PAD.top - PAD.bottom;
  const groupW = plotW / data.length;
  const barW = Math.min(26, groupW / 2 - 4);
  const y = (value: number) => PAD.top + plotH - (value / max) * plotH;

  const compact = new Intl.NumberFormat(i18n.language, {
    notation: "compact",
    maximumFractionDigits: 1,
  });
  const monthName = (month: string) =>
    new Intl.DateTimeFormat(i18n.language, { month: "short" }).format(
      new Date(`${month}-01T00:00:00`),
    );

  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => f * max);

  return (
    <div>
      <div
        className="mb-1 flex gap-4 text-xs"
        style={{ color: "var(--omed-text-secondary)" }}
      >
        <span className="inline-flex items-center gap-1">
          <span
            aria-hidden="true"
            className="inline-block h-2 w-2 rounded-sm"
            style={{ background: "var(--omed-primary)" }}
          />
          {t("fin.dashboard.revenue")}
        </span>
        <span className="inline-flex items-center gap-1">
          <span
            aria-hidden="true"
            className="inline-block h-2 w-2 rounded-sm"
            style={{ background: "var(--omed-text-secondary)" }}
          />
          {t("fin.dashboard.expense")}
        </span>
      </div>
      <svg
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        role="img"
        aria-label={t("fin.dashboard.chartLabel")}
        className="h-auto w-full"
      >
        {ticks.map((tick) => (
          <g key={tick}>
            <line
              x1={PAD.left}
              x2={WIDTH - PAD.right}
              y1={y(tick)}
              y2={y(tick)}
              stroke="var(--omed-border-soft)"
            />
            <text
              x={PAD.left - 6}
              y={y(tick) + 4}
              textAnchor="end"
              fontSize="11"
              fill="var(--omed-text-secondary)"
            >
              {compact.format(tick)}
            </text>
          </g>
        ))}
        {data.map((d, i) => {
          const gx = PAD.left + i * groupW + groupW / 2;
          return (
            <g key={d.month}>
              <rect
                x={gx - barW - 1}
                y={y(d.revenue)}
                width={barW}
                height={PAD.top + plotH - y(d.revenue)}
                rx="3"
                fill="var(--omed-primary)"
              />
              <rect
                x={gx + 1}
                y={y(d.expense)}
                width={barW}
                height={PAD.top + plotH - y(d.expense)}
                rx="3"
                fill="var(--omed-text-secondary)"
              />
              <text
                x={gx}
                y={HEIGHT - 6}
                textAnchor="middle"
                fontSize="11"
                fill="var(--omed-text-secondary)"
              >
                {monthName(d.month)}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}
