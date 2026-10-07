import { z } from "zod";

/**
 * UI locales supported by Omed clients. Stored on `user.locale`.
 * Keep in sync with `apps/dash/app/shared/i18n/config.ts`.
 */
export const SUPPORTED_LOCALES = ["en", "id"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

export const localeSchema = z.enum(SUPPORTED_LOCALES).nullish();

/**
 * Currencies supported for user profile defaults in Omed.
 * Stored on `user.defaultCurrency`.
 */
export const SUPPORTED_CURRENCIES = [
  "IDR",
  "USD",
  "EUR",
  "SGD",
  "GBP",
  "JPY",
  "AUD",
] as const;
export type Currency = (typeof SUPPORTED_CURRENCIES)[number];
export const DEFAULT_CURRENCY: Currency = "IDR";

export const currencySchema = z
  .string()
  .length(3)
  .regex(/^[A-Z]{3}$/)
  .nullish();
