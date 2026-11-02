import type { components } from "@omed/openapi";
import dayjs, { type Dayjs } from "dayjs";
import { api, unwrapOne } from "~/lib/api";
import type { FinancialSuggestion } from "~/modules/agent/queries";

export type AccountResponse = components["schemas"]["model.AccountResponse"];

export interface PaymentAccountOption {
  label: string;
  value: number;
  account: AccountResponse;
}

export interface PaymentAccountGroup {
  label: string;
  options: PaymentAccountOption[];
}

/**
 * Filter active, postable ASSET accounts and group them by parent liquid account.
 * Standard groupings:
 * - 1110 Kas & E-Wallet
 * - 1120 Bank Operasional
 * - 1130 Platform & Gateway
 * Excludes 114x (AR), 115x (Prepaid), 12xx+ (Fixed Assets).
 * Falls back to all active postable ASSET accounts if no standard liquid accounts found.
 */
export function groupPaymentAccounts(
  accounts: AccountResponse[],
): PaymentAccountGroup[] {
  const byId = new Map<number, AccountResponse>();
  for (const acc of accounts) {
    if (acc.id !== undefined) {
      byId.set(acc.id, acc);
    }
  }

  const postableAssets = accounts.filter(
    (acc) =>
      acc.isActive !== false && acc.isPostable === true && acc.type === "ASSET",
  );

  const isExcluded = (code?: string) => {
    if (!code) return false;
    if (code.startsWith("114") || code.startsWith("115")) return true;
    const num = Number.parseInt(code, 10);
    if (!Number.isNaN(num) && num >= 1200) return true;
    return false;
  };

  const liquidAccounts = postableAssets.filter((acc) => !isExcluded(acc.code));
  const targetAccounts =
    liquidAccounts.length > 0 ? liquidAccounts : postableAssets;

  const groupsMap = new Map<string, PaymentAccountOption[]>();

  for (const acc of targetAccounts) {
    let groupLabel = "Other";

    if (acc.parentId !== undefined && byId.has(acc.parentId)) {
      const parent = byId.get(acc.parentId);
      if (parent?.name) {
        groupLabel = parent.name;
      }
    } else if (acc.code) {
      if (acc.code.startsWith("111")) {
        groupLabel = "Kas & E-Wallet";
      } else if (acc.code.startsWith("112")) {
        groupLabel = "Bank Operasional";
      } else if (acc.code.startsWith("113")) {
        groupLabel = "Platform & Gateway";
      }
    }

    const option: PaymentAccountOption = {
      label: `${acc.code ?? ""} - ${acc.name ?? ""}`,
      value: acc.id as number,
      account: acc,
    };

    const existing = groupsMap.get(groupLabel);
    if (existing) {
      existing.push(option);
    } else {
      groupsMap.set(groupLabel, [option]);
    }
  }

  const result: PaymentAccountGroup[] = [];
  for (const [label, options] of groupsMap.entries()) {
    result.push({ label, options });
  }

  return result;
}

/**
 * Uploads a file to POST /workspaces/{wid}/files and returns the uploaded file ID.
 */
export async function uploadWorkspaceFile(
  wid: number,
  file: File,
): Promise<number> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("name", file.name);

  const res = await unwrapOne(
    api.POST("/workspaces/{wid}/files", {
      params: { path: { wid } },
      body: formData as unknown as { file: string; name?: string },
      bodySerializer: (body) => body as unknown as BodyInit,
    }),
  );

  if (!res?.id) {
    throw new Error("File upload failed: missing id");
  }

  return res.id;
}

export interface CashflowItemValue {
  accountId?: number;
  description?: string;
  quantity?: number;
  unitPrice?: number;
}

export function computeLineAmount(
  quantity?: number,
  unitPrice?: number,
  decimals?: number,
): number {
  const qty = quantity ?? 0;
  const price = unitPrice ?? 0;
  const raw = qty * price;
  if (decimals !== undefined) {
    const factor = 10 ** decimals;
    return Math.round(raw * factor) / factor;
  }
  return raw;
}

export function computeTotal(
  items: CashflowItemValue[],
  decimals?: number,
): number {
  const sum = items.reduce(
    (acc, item) =>
      acc + computeLineAmount(item.quantity, item.unitPrice, decimals),
    0,
  );
  if (decimals !== undefined) {
    const factor = 10 ** decimals;
    return Math.round(sum * factor) / factor;
  }
  return sum;
}

export type PrefillMode = "default" | "overwrite";

export type PrefillScalar =
  | "transactionDate"
  | "paymentAccountId"
  | "currency"
  | "exchangeRate"
  | "reference"
  | "description";

export interface PrefillResult {
  values: Partial<{
    transactionDate: Dayjs;
    paymentAccountId: number;
    currency: string;
    exchangeRate: number;
    reference: string;
    description: string;
    items: CashflowItemValue[];
  }>;
  filledFields: string[]; // scalar names plus "items" when items changed
}

function isBlankDefaultRow(item?: CashflowItemValue): boolean {
  if (!item) return false;
  const noAccount = item.accountId == null;
  const noDescription = !item.description || item.description.trim() === "";
  const isDefaultQty = item.quantity === 1 || item.quantity == null;
  const isDefaultPrice = item.unitPrice === 0 || item.unitPrice == null;
  return noAccount && noDescription && isDefaultQty && isDefaultPrice;
}

function areItemsEqual(a: CashflowItemValue, b: CashflowItemValue): boolean {
  const accountA = a.accountId ?? undefined;
  const accountB = b.accountId ?? undefined;
  if (accountA !== accountB) return false;

  const descA = (a.description ?? "").trim().toLowerCase();
  const descB = (b.description ?? "").trim().toLowerCase();
  if (descA !== descB) return false;

  const qtyA = a.quantity ?? 0;
  const qtyB = b.quantity ?? 0;
  if (qtyA !== qtyB) return false;

  const priceA = a.unitPrice ?? 0;
  const priceB = b.unitPrice ?? 0;
  if (priceA !== priceB) return false;

  return true;
}

function areItemArraysEqual(
  a: CashflowItemValue[],
  b: CashflowItemValue[],
): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (!areItemsEqual(a[i], b[i])) return false;
  }
  return true;
}

function isEmpty(val: unknown): boolean {
  if (val === undefined || val === null) return true;
  if (typeof val === "string" && val.trim() === "") return true;
  return false;
}

export function mergeItems(
  existing: CashflowItemValue[],
  incoming: NonNullable<FinancialSuggestion["items"]>,
): CashflowItemValue[] {
  if (incoming.length === 0) {
    return existing;
  }

  const isOnlyBlankRow =
    existing.length === 1 && isBlankDefaultRow(existing[0]);
  const result: CashflowItemValue[] = isOnlyBlankRow ? [] : [...existing];

  for (const item of incoming) {
    const rawQty = item.quantity != null ? Number(item.quantity) : 1;
    const rawPrice = item.unitPrice != null ? Number(item.unitPrice) : 0;
    const normalized: CashflowItemValue = {
      accountId: item.accountId != null ? item.accountId : undefined,
      description: item.description ?? undefined,
      quantity: Number.isNaN(rawQty) ? 1 : rawQty,
      unitPrice: Number.isNaN(rawPrice) ? 0 : rawPrice,
    };

    const isDuplicate = result.some((r) => areItemsEqual(r, normalized));
    if (!isDuplicate) {
      result.push(normalized);
    }
  }

  return result;
}

export function applyAnalysis(args: {
  current: { [K in PrefillScalar]?: unknown } & { items: CashflowItemValue[] };
  isTouched: (field: PrefillScalar) => boolean;
  suggestion: FinancialSuggestion;
  mode: PrefillMode;
}): PrefillResult {
  const { current, isTouched, suggestion, mode } = args;
  const values: PrefillResult["values"] = {};
  const filledFields: string[] = [];

  const isScalarEligible = (field: PrefillScalar, currentVal: unknown) => {
    if (mode === "overwrite") return true;
    return isEmpty(currentVal) || !isTouched(field);
  };

  // 1. transactionDate
  if (suggestion.transactionDate != null && suggestion.transactionDate !== "") {
    const suggestedDate = dayjs(suggestion.transactionDate);
    if (
      suggestedDate.isValid() &&
      isScalarEligible("transactionDate", current.transactionDate)
    ) {
      values.transactionDate = suggestedDate;
      const cur = current.transactionDate;
      let changed = false;
      if (isEmpty(cur)) {
        changed = true;
      } else {
        const curDate = dayjs.isDayjs(cur) ? cur : dayjs(cur as string);
        changed = !curDate.isValid() || !curDate.isSame(suggestedDate, "day");
      }
      if (changed) {
        filledFields.push("transactionDate");
      }
    }
  }

  // 2. paymentAccountId
  if (suggestion.paymentAccountId != null) {
    const accountId = Number(suggestion.paymentAccountId);
    if (
      !Number.isNaN(accountId) &&
      isScalarEligible("paymentAccountId", current.paymentAccountId)
    ) {
      values.paymentAccountId = accountId;
      const cur = current.paymentAccountId;
      const changed = cur == null || Number(cur) !== accountId;
      if (changed) {
        filledFields.push("paymentAccountId");
      }
    }
  }

  // 3. currency
  if (suggestion.currency != null && suggestion.currency !== "") {
    if (isScalarEligible("currency", current.currency)) {
      values.currency = suggestion.currency;
      const cur = current.currency;
      const changed = isEmpty(cur) || String(cur) !== suggestion.currency;
      if (changed) {
        filledFields.push("currency");
      }
    }
  }

  // 4. exchangeRate
  if (suggestion.exchangeRate != null && suggestion.exchangeRate !== "") {
    const rate = Number(suggestion.exchangeRate);
    if (
      !Number.isNaN(rate) &&
      isScalarEligible("exchangeRate", current.exchangeRate)
    ) {
      values.exchangeRate = rate;
      const cur = current.exchangeRate;
      const changed = cur == null || cur === "" || Number(cur) !== rate;
      if (changed) {
        filledFields.push("exchangeRate");
      }
    }
  }

  // 5. reference
  if (suggestion.reference != null && suggestion.reference !== "") {
    if (isScalarEligible("reference", current.reference)) {
      values.reference = suggestion.reference;
      const cur = current.reference;
      const changed = isEmpty(cur) || String(cur) !== suggestion.reference;
      if (changed) {
        filledFields.push("reference");
      }
    }
  }

  // 6. description
  if (suggestion.description != null && suggestion.description !== "") {
    if (isScalarEligible("description", current.description)) {
      values.description = suggestion.description;
      const cur = current.description;
      const changed = isEmpty(cur) || String(cur) !== suggestion.description;
      if (changed) {
        filledFields.push("description");
      }
    }
  }

  // 7. items
  const isPaymentProof =
    suggestion.documentType?.toLowerCase() === "payment_proof";

  if (!isPaymentProof && suggestion.items != null) {
    let nextItems: CashflowItemValue[];
    if (mode === "overwrite") {
      nextItems = mergeItems([], suggestion.items);
    } else {
      nextItems = mergeItems(current.items, suggestion.items);
    }

    if (!areItemArraysEqual(current.items, nextItems)) {
      values.items = nextItems;
      filledFields.push("items");
    }
  }

  return { values, filledFields };
}

export function checkTotalMismatch(args: {
  suggestion: FinancialSuggestion;
  formTotal: number;
  formCurrency: string;
}): { fileTotal: number; compared: number } | null {
  const { suggestion, formTotal, formCurrency } = args;

  if (suggestion.documentTotal == null || suggestion.documentTotal === "") {
    return null;
  }
  const fileTotal = Number(suggestion.documentTotal);
  if (Number.isNaN(fileTotal)) {
    return null;
  }

  if (
    suggestion.currency != null &&
    suggestion.currency !== "" &&
    suggestion.currency.toUpperCase() !== formCurrency.toUpperCase()
  ) {
    return null;
  }

  const isPaymentProof =
    suggestion.documentType?.toLowerCase() === "payment_proof";
  let compared: number;

  if (isPaymentProof) {
    compared = formTotal;
  } else {
    const items: CashflowItemValue[] = (suggestion.items ?? []).map((it) => {
      const q = it.quantity != null ? Number(it.quantity) : 1;
      const p = it.unitPrice != null ? Number(it.unitPrice) : 0;
      return {
        accountId: it.accountId ?? undefined,
        description: it.description ?? undefined,
        quantity: Number.isNaN(q) ? 1 : q,
        unitPrice: Number.isNaN(p) ? 0 : p,
      };
    });
    compared = computeTotal(items);
  }

  const diff = Math.abs(fileTotal - compared);
  if (Math.round(diff * 100) / 100 <= 0.01) {
    return null;
  }

  return { fileTotal, compared };
}
