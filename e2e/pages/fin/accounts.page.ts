import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinAccountsPage extends BasePage {
  readonly path = "/fin/accounts";

  get setupTemplateButton(): Locator {
    return this.page.getByRole("button", { name: "Setup template" });
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  get emptyStateSeedButton(): Locator {
    return this.emptyState.getByRole("button", { name: "Setup template" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  async expectEmptyStateVisible(): Promise<void> {
    await this.emptyState.waitFor({ state: "visible" });
  }

  async clickEmptyStateSeedButton(): Promise<void> {
    await this.emptyStateSeedButton.waitFor({ state: "visible" });
    await this.emptyStateSeedButton.click();
  }

  async expectSetupTemplateButtonHidden(): Promise<void> {
    const isVisible = await this.setupTemplateButton
      .isVisible({ timeout: 1500 })
      .catch(() => false);
    assert.strictEqual(
      isVisible,
      false,
      "Expected setup template button to be hidden",
    );
  }

  async expectAccountInTable(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code });
    await row.first().waitFor({ state: "visible" });
    const text = await row.first().innerText();
    assert.ok(
      text.includes(code),
      `Expected account with code "${code}" to be in table, got "${text}"`,
    );
  }
}
