import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import finModule from "../module";
import FinLauncher from "./launcher";

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => key,
      i18n: { language: "en" },
    }),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({
      children,
      to,
      className,
    }: {
      children: React.ReactNode;
      to: string;
      className?: string;
    }) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
  };
});

describe("FinLauncher component", () => {
  it("renders tiles linking to all fin sub-routes with correct styling", () => {
    const html = renderToString(<FinLauncher />);

    expect(html).toContain("home-backdrop");
    expect(html).toContain("app-tile");

    const expectedLinks = [
      "/fin/overview",
      "/fin/income",
      "/fin/expense",
      "/fin/accounts",
      "/fin/periods",
      "/fin/opening-balance",
      "/fin/transactions",
      "/fin/journal",
      "/fin/ledger",
      "/fin/trial-balance",
    ];

    for (const link of expectedLinks) {
      expect(html).toContain(`href="${link}"`);
    }

    expect(html).toContain("fin.menu.overview");
    expect(html).toContain("fin.menu.income");
    expect(html).toContain("fin.menu.expense");
    expect(html).toContain("fin.menu.accounts");
    expect(html).toContain("fin.menu.periods");
    expect(html).toContain("fin.menu.openingBalance");
    expect(html).toContain("fin.menu.transactions");
    expect(html).toContain("fin.menu.journal");
    expect(html).toContain("fin.menu.ledger");
    expect(html).toContain("fin.menu.trialBalance");
  });
});

describe("fin module configuration", () => {
  it("defines expected top-level menu and accounting sub-menu", () => {
    expect(finModule.id).toBe("fin");
    const menuKeys = finModule.menu.map((m) => m.key);
    expect(menuKeys).toEqual(["overview", "income", "expense", "accounting"]);

    const overview = finModule.menu.find((m) => m.key === "overview");
    expect(overview?.path).toBe("overview");
    expect(overview?.labelKey).toBe("fin.menu.overview");

    const income = finModule.menu.find((m) => m.key === "income");
    expect(income?.path).toBe("income");
    expect(income?.labelKey).toBe("fin.menu.income");

    const expense = finModule.menu.find((m) => m.key === "expense");
    expect(expense?.path).toBe("expense");
    expect(expense?.labelKey).toBe("fin.menu.expense");

    const accounting = finModule.menu.find((m) => m.key === "accounting");
    expect(accounting?.labelKey).toBe("fin.menu.accounting");
    expect(accounting?.children?.map((c) => c.key)).toEqual([
      "accounts",
      "periods",
      "opening-balance",
      "transactions",
      "journal",
      "ledger",
      "trial-balance",
    ]);
  });
});
