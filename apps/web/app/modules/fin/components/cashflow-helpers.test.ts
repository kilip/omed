import { describe, expect, it } from "vitest";
import {
  type CashflowItemValue,
  computeLineAmount,
  computeTotal,
} from "./cashflow-helpers";

describe("cashflow-helpers line calculation", () => {
  it("computes integer line amount correctly", () => {
    expect(computeLineAmount(6, 500000, 0)).toBe(3000000);
  });

  it("computes decimal line amount with rounding to currency decimals", () => {
    expect(computeLineAmount(2.5, 3333, 0)).toBe(8333);
  });

  it("computes total of the owner expense items", () => {
    const items: CashflowItemValue[] = [
      { accountId: 6710, description: "telur", quantity: 2, unitPrice: 55000 },
      { accountId: 6710, description: "indomie", quantity: 5, unitPrice: 3000 },
    ];
    expect(computeTotal(items)).toBe(125000);
  });
});
