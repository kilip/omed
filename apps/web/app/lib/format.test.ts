import { describe, expect, it } from "bun:test";
import {
  formatIDR,
  formatMoney,
  getCurrencyDecimals,
  isBalanced,
  roundToCurrency,
} from "./format";

describe("formatMoney", () => {
  it("formats IDR with 0 decimals", () => {
    const formatted = formatMoney(50000, "IDR", "id");
    // IDR format in Indonesian: Rp 50.000 or Rp50.000
    expect(formatted.replace(/\s/g, " ")).toContain("50.000");
    expect(formatted).not.toContain(",00");
  });

  it("formats USD with 2 decimals", () => {
    const formatted = formatMoney(1234.5, "USD", "en");
    expect(formatted.replace(/\s/g, " ")).toContain("1,234.50");
  });

  it("formats KWD with 3 decimals", () => {
    const formatted = formatMoney(10.5, "KWD", "en");
    expect(formatted.replace(/\s/g, " ")).toContain("10.500");
  });

  it("keeps formatIDR working as before", () => {
    const formatted = formatIDR(1000000, "id");
    expect(formatted.replace(/\s/g, " ")).toContain("1.000.000");
  });
});

describe("getCurrencyDecimals", () => {
  it("returns correct decimals per currency", () => {
    expect(getCurrencyDecimals("IDR")).toBe(0);
    expect(getCurrencyDecimals("JPY")).toBe(0);
    expect(getCurrencyDecimals("USD")).toBe(2);
    expect(getCurrencyDecimals("EUR")).toBe(2);
    expect(getCurrencyDecimals("KWD")).toBe(3);
    expect(getCurrencyDecimals("UNKNOWN")).toBe(2);
  });
});

describe("balance helpers", () => {
  it("isBalanced compares floating numbers accurately rounded to currency decimals", () => {
    // 0.1 + 0.2 in JS is 0.30000000000000004
    const a = 0.1 + 0.2;
    const b = 0.3;
    expect(a === b).toBe(false);
    expect(isBalanced(a, b, "USD")).toBe(true);
  });

  it("isBalanced returns false when amounts genuinely differ", () => {
    expect(isBalanced(100.01, 100.02, "USD")).toBe(false);
    expect(isBalanced(100, 101, "IDR")).toBe(false);
  });

  it("roundToCurrency rounds to currency decimal places", () => {
    expect(roundToCurrency(0.1 + 0.2, "USD")).toBe(0.3);
    expect(roundToCurrency(10.556, "USD")).toBe(10.56);
    expect(roundToCurrency(10.556, "IDR")).toBe(11);
    expect(roundToCurrency(10.5556, "KWD")).toBe(10.556);
  });
});
