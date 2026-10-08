// Keep SUPPORTED_LOCALES in sync with `packages/better-auth/src/locale.ts`
// (duplicated on purpose: importing `@omed/better-auth` at runtime would pull
// server-only code into the browser bundle).
export const SUPPORTED_LOCALES = ["en", "id"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";
export const LOCALE_STORAGE_KEY = "omed-lang";

export const NAMESPACES = [
  "common",
  "nav",
  "auth",
  "home",
  "finance",
  "errors",
  "settings",
  "invoices",
  "contacts",
  "payments",
] as const;
export type Namespace = (typeof NAMESPACES)[number];

/** Language names are always shown in their own language. */
export const LOCALE_LABELS: Record<Locale, string> = {
  en: "English",
  id: "Bahasa Indonesia",
};

/** BCP 47 tags used for Intl formatting. */
export const INTL_LOCALES: Record<Locale, string> = {
  en: "en-US",
  id: "id-ID",
};

export const isLocale = (v: unknown): v is Locale =>
  typeof v === "string" && (SUPPORTED_LOCALES as readonly string[]).includes(v);

export const toLocale = (v: unknown): Locale =>
  isLocale(v) ? v : DEFAULT_LOCALE;
