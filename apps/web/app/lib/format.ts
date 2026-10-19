export type Lang = "en" | "id";

const locale = (lang: string) => (lang === "id" ? "id-ID" : "en-US");

/**
 * Standard currency decimals mapping following ISO 4217.
 * For unlisted currencies, standard 2 decimal places is used.
 */
const CURRENCY_DECIMALS: Record<string, number> = {
  IDR: 0,
  JPY: 0,
  KRW: 0,
  VND: 0,
  KWD: 3,
  BHD: 3,
  OMR: 3,
  JOD: 3,
  USD: 2,
  EUR: 2,
  GBP: 2,
  SGD: 2,
  AUD: 2,
  CAD: 2,
  CHF: 2,
  CNY: 2,
  HKD: 2,
};

export function getCurrencyDecimals(currency = "IDR"): number {
  const upper = currency.toUpperCase();
  if (upper in CURRENCY_DECIMALS) {
    return CURRENCY_DECIMALS[upper] ?? 2;
  }
  return 2;
}

export function formatNumber(value: number, lang: string): string {
  return new Intl.NumberFormat(locale(lang)).format(value);
}

/** Indonesian rupiah, no decimals. */
export function formatIDR(value: number, lang: string): string {
  return formatMoney(value, "IDR", lang);
}

/**
 * Currency-aware money formatter. Decimal places are driven by currency (e.g. IDR=0, USD=2, KWD=3).
 */
export function formatMoney(
  amount: number,
  currency = "IDR",
  lang: string = "id",
): string {
  const decimals = getCurrencyDecimals(currency);
  return new Intl.NumberFormat(locale(lang), {
    style: "currency",
    currency: currency.toUpperCase(),
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(amount);
}

/**
 * Compares two monetary amounts for equality by rounding to currency decimals.
 * Prevents floating-point rounding issues (e.g. 0.1 + 0.2 !== 0.3).
 */
export function isBalanced(
  amountA: number,
  amountB: number,
  currency = "IDR",
): boolean {
  const decimals = getCurrencyDecimals(currency);
  const factor = 10 ** decimals;
  return Math.round(amountA * factor) === Math.round(amountB * factor);
}

/**
 * Calculates net balance difference rounded to currency decimals.
 */
export function roundToCurrency(amount: number, currency = "IDR"): number {
  const decimals = getCurrencyDecimals(currency);
  const factor = 10 ** decimals;
  return Math.round(amount * factor) / factor;
}

export function formatDate(value: string | undefined, lang: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return new Intl.DateTimeFormat(locale(lang), {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

export function formatDateTime(value: string | undefined, lang: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return new Intl.DateTimeFormat(locale(lang), {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
