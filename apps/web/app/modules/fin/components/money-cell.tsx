import type { HTMLAttributes } from "react";
import { useTranslation } from "react-i18next";
import { formatMoney } from "~/lib/format";

export type MoneyCellProps = HTMLAttributes<HTMLDivElement> & {
  amount?: number | string | null;
  currency?: string;
  locale?: string;
};

export function MoneyCell({
  amount,
  currency = "IDR",
  locale,
  className = "",
  ...props
}: MoneyCellProps) {
  const { i18n } = useTranslation();
  const currentLang = locale ?? i18n.language;

  const formatted =
    amount !== undefined && amount !== null
      ? formatMoney(amount, currency, currentLang)
      : "-";

  return (
    <div className={`num text-right font-mono ${className}`.trim()} {...props}>
      {formatted}
    </div>
  );
}
