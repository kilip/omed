export type Lang = "en" | "id";

const locale = (lang: string) => (lang === "id" ? "id-ID" : "en-US");

export function formatNumber(value: number, lang: string): string {
  return new Intl.NumberFormat(locale(lang)).format(value);
}

/** Indonesian rupiah, no decimals. */
export function formatIDR(value: number, lang: string): string {
  return new Intl.NumberFormat(locale(lang), {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);
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
