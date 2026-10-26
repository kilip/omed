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

const toNum = (v: number | string): number => {
  if (typeof v === "number") return v;
  const n = Number.parseFloat(v);
  return Number.isNaN(n) ? 0 : n;
};

export function formatNumber(value: number | string, lang: string): string {
  return new Intl.NumberFormat(locale(lang)).format(toNum(value));
}

/** Indonesian rupiah, no decimals. */
export function formatIDR(value: number | string, lang: string): string {
  return formatMoney(value, "IDR", lang);
}

/**
 * Currency-aware money formatter. Decimal places are driven by currency (e.g. IDR=0, USD=2, KWD=3).
 */
export function formatMoney(
  amount: number | string,
  currency = "IDR",
  lang: string = "id",
): string {
  const decimals = getCurrencyDecimals(currency);
  return new Intl.NumberFormat(locale(lang), {
    style: "currency",
    currency: currency.toUpperCase(),
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(toNum(amount));
}

/**
 * Compares two monetary amounts for equality by rounding to currency decimals.
 * Prevents floating-point rounding issues (e.g. 0.1 + 0.2 !== 0.3).
 */
export function isBalanced(
  amountA: number | string,
  amountB: number | string,
  currency = "IDR",
): boolean {
  const numA = toNum(amountA);
  const numB = toNum(amountB);
  const decimals = getCurrencyDecimals(currency);
  const factor = 10 ** decimals;
  return Math.round(numA * factor) === Math.round(numB * factor);
}

/**
 * Calculates net balance difference rounded to currency decimals.
 */
export function roundToCurrency(
  amount: number | string,
  currency = "IDR",
): number {
  const num = toNum(amount);
  const decimals = getCurrencyDecimals(currency);
  const factor = 10 ** decimals;
  return Math.round(num * factor) / factor;
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
