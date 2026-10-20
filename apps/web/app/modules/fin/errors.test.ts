import { describe, expect, it } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";

const STANDARD_FIN_ERROR_CODES = [
  "ENTRY_NOT_BALANCED",
  "ENTRY_BASE_NOT_BALANCED",
  "ENTRY_MIN_POSTINGS",
  "ENTRY_DATE_OUTSIDE_PERIOD",
  "ENTRY_NOT_FOUND",
  "ENTRY_NOT_DRAFT",
  "ENTRY_ALREADY_REVERSED",
  "TRANSACTION_NOT_FOUND",
  "TRANSACTION_NUMBER_TAKEN",
  "TRANSACTION_HAS_POSTED_ENTRIES",
  "ATTACHMENT_NOT_FOUND",
  "ATTACHMENT_TOO_LARGE",
  "ATTACHMENT_INVALID_TYPE",
  "POSTING_ACCOUNT_NOT_POSTABLE",
  "POSTING_ACCOUNT_INACTIVE",
  "POSTING_CURRENCY_MISMATCH",
  "POSTING_INVALID_AMOUNT",
  "PERIOD_NOT_FOUND",
  "PERIOD_CLOSED",
  "PERIOD_ALREADY_CLOSED",
  "PERIOD_ALREADY_OPEN",
  "PERIOD_HAS_DRAFT_ENTRIES",
  "PERIOD_ALREADY_REVALUED",
  "PERIOD_INVALID_DATES",
  "PERIOD_OVERLAP",
  "CURRENCY_NOT_FOUND",
  "EXCHANGE_RATE_NOT_FOUND",
  "EXCHANGE_RATE_DUPLICATE",
  "EXCHANGE_RATE_INVALID",
  "BASE_CURRENCY_LOCKED",
  "ACCOUNT_CODE_TAKEN",
  "ACCOUNT_HAS_CHILDREN",
  "ACCOUNT_PARENT_NOT_FOUND",
  "ACCOUNT_CYCLIC_PARENT",
  "INVALID_CURSOR",
] as const;

describe("fin error code translations completeness", () => {
  it("en contains all standard error codes with non-empty translation strings", () => {
    const enErrors = en.fin.errors as Record<string, string>;
    for (const code of STANDARD_FIN_ERROR_CODES) {
      expect(enErrors[code]).toBeDefined();
      expect(enErrors[code].trim().length).toBeGreaterThan(0);
    }
  });

  it("id contains all standard error codes with non-empty translation strings", () => {
    const idErrors = id.fin.errors as Record<string, string>;
    for (const code of STANDARD_FIN_ERROR_CODES) {
      expect(idErrors[code]).toBeDefined();
      expect(idErrors[code].trim().length).toBeGreaterThan(0);
    }
  });

  it("en and id define matching error code keys", () => {
    const enKeys = Object.keys(en.fin.errors).sort();
    const idKeys = Object.keys(id.fin.errors).sort();
    expect(enKeys).toEqual(idKeys);
  });
});
