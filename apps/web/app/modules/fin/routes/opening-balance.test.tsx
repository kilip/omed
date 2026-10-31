import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { MoneyInput } from "../components/money-input";

describe("opening balance ui money input", () => {
  it("renders money input capable of displaying 9-digit values", () => {
    const html = renderToString(
      <MoneyInput currency="IDR" className="w-56 sm:w-64" value={500000000} />,
    );
    expect(html).toContain("500,000,000");
    expect(html).toContain("w-56");
  });
});
