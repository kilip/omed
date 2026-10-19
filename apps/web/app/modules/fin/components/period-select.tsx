import type { components } from "@omed/openapi";
import { Select, type SelectProps } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

type Period = components["schemas"]["model.PeriodResponse"];

export function findPeriodForDate(
  periods: Period[],
  dateStr?: string | Date,
): Period | undefined {
  if (!dateStr) return undefined;
  const target = new Date(dateStr).getTime();
  if (Number.isNaN(target)) return undefined;

  return periods.find((p) => {
    if (p.status !== "OPEN") return false;
    const start = p.startDate ? new Date(p.startDate).getTime() : -Infinity;
    const end = p.endDate ? new Date(p.endDate).getTime() : Infinity;
    return target >= start && target <= end;
  });
}

export type PeriodSelectProps = Omit<SelectProps<number>, "options"> & {
  periods: Period[];
};

export function PeriodSelect({
  periods,
  placeholder,
  ...props
}: PeriodSelectProps) {
  const { t } = useTranslation();

  const options = useMemo(() => {
    return periods
      .filter((p) => p.status === "OPEN")
      .map((p) => ({
        value: p.id as number,
        label: `${p.name ?? `FY${p.fiscalYear}`} (${p.startDate ?? ""} - ${p.endDate ?? ""})`,
        period: p,
      }));
  }, [periods]);

  return (
    <Select
      showSearch
      allowClear
      placeholder={placeholder ?? t("common.period", "Select Period")}
      optionFilterProp="label"
      filterOption={(input, option) =>
        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
      }
      options={options}
      {...props}
    />
  );
}
