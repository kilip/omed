import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinEntriesPage extends BasePage {
  readonly path = "/fin/journal";

  get newEntryButton(): Locator {
    return this.page.getByRole("button", { name: "New entry" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  get searchInput(): Locator {
    return this.page.getByPlaceholder("Search description or number...");
  }

  // Drawer locators
  get drawer(): Locator {
    return this.page.locator(".ant-drawer");
  }

  get drawerSaveButton(): Locator {
    return this.drawer.getByRole("button", { name: "Save" });
  }

  get transactionModeNew(): Locator {
    return this.drawer.getByRole("radio", { name: "Create new transaction" });
  }

  get transactionModeExisting(): Locator {
    return this.drawer.getByRole("radio", { name: "Select existing" });
  }

  get descriptionInput(): Locator {
    return this.drawer.getByPlaceholder("Description");
  }

  get addLineButton(): Locator {
    return this.drawer.getByRole("button", { name: "Add line" });
  }

  async clickNewEntry(): Promise<void> {
    await this.newEntryButton.waitFor({ state: "visible" });
    await this.newEntryButton.click();
  }

  async expectDrawerVisible(): Promise<void> {
    await this.drawer.waitFor({ state: "visible" });
  }

  async fillDescription(desc: string): Promise<void> {
    await this.descriptionInput.fill(desc);
  }

  async fillPostingLine(
    index: number,
    opts: { accountSearch: string; debit?: number; credit?: number },
  ): Promise<void> {
    const lines = this.drawer.locator(".grid.grid-cols-12.items-center");
    const line = lines.nth(index);
    await line.waitFor({ state: "visible" });

    // Select account
    const select = line.locator(".ant-select");
    await select.click();
    await this.page.keyboard.type(opts.accountSearch);
    await this.page.locator(".ant-select-item-option").first().click();

    // Debit or credit
    if (opts.debit !== undefined) {
      const debitInput = line.locator("input.ant-input-number-input").first();
      await debitInput.fill(String(opts.debit));
    } else if (opts.credit !== undefined) {
      const creditInput = line.locator("input.ant-input-number-input").nth(1);
      await creditInput.fill(String(opts.credit));
    }
  }

  async saveDrawer(): Promise<void> {
    await this.drawerSaveButton.click();
  }

  async expectEntryInTable(text: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    const rowText = await row.first().innerText();
    assert.ok(
      rowText.includes(text),
      `Expected row to contain "${text}", got "${rowText}"`,
    );
  }

  async clickRow(text: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: text });
    await row.first().click();
  }

  async expectNewEntryVisible(): Promise<void> {
    await this.newEntryButton.waitFor({ state: "visible" });
  }

  async expectNewEntryHidden(): Promise<void> {
    await this.newEntryButton.waitFor({ state: "hidden" });
  }
}
