import type { components } from "@omed/openapi";
import { api, unwrapOne } from "~/lib/api";

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
