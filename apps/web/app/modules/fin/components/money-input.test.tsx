import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";
import {
  formatMoneyInputValue,
  MONEY_INPUT_MIN_WIDTH,
  MoneyInput,
  parseMoneyInputValue,
} from "./money-input";

describe("money-input", () => {
  it("formats value with comma decimal style and 2 decimals", () => {
    expect(formatMoneyInputValue(999999999, "comma", 2)).toBe("999.999.999");
  });

  it("formats value with dot decimal style and 2 decimals", () => {
    expect(formatMoneyInputValue(1234.5, "dot", 2)).toBe("1,234.5");
  });

  it("parses money input value in comma style", () => {
    expect(parseMoneyInputValue("999.999.999,5", "comma")).toBe(999999999.5);
  });

  it("parses money input value in dot style", () => {
    expect(parseMoneyInputValue("1,234.5", "dot")).toBe(1234.5);
  });

  it("returns null when parsing invalid input", () => {
    expect(parseMoneyInputValue("abc", "dot")).toBeNull();
    expect(parseMoneyInputValue("abc", "comma")).toBeNull();
  });

  it("renders MoneyInput with minimum width of 168px", () => {
    expect(MONEY_INPUT_MIN_WIDTH).toBe(168);
    const html = renderToString(<MoneyInput value={999999999} />);
    expect(html).toContain("min-width:168px");
  });
});
