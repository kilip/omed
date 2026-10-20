import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinLedgerPage extends BasePage {
  readonly path = "/fin/ledger";

  get accountSelect(): Locator {
    return this.page.locator("main .ant-select").first();
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  async selectAccount(searchQuery: string): Promise<void> {
    await this.accountSelect.waitFor({ state: "visible" });
    await this.accountSelect.click();
    await this.page.waitForTimeout(300);
    const input = this.accountSelect.locator("input");
    await input.fill(searchQuery);
    await this.page.waitForTimeout(400);

    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: searchQuery });
    await option.first().waitFor({ state: "visible" });
    await option.first().click();
    await option.first().waitFor({ state: "hidden" });
  }

  async expectPostingInTable(text: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    const rowText = await row.first().innerText();
    assert.ok(
      rowText.includes(text),
      `Expected posting row containing "${text}", got "${rowText}"`,
    );
  }

  async clickEntryLinkInRow(entryNumberOrId: string | number): Promise<void> {
    const link = this.page
      .locator(".ant-table-tbody")
      .getByRole("link", { name: new RegExp(`#${entryNumberOrId}`) });
    await link.first().waitFor({ state: "visible" });
    await link.first().click();
  }

  async expectRunningBalanceVisible(): Promise<void> {
    const balanceSpan = this.page.locator("span.num, div.num").first();
    await balanceSpan.waitFor({ state: "visible" });
  }
}
