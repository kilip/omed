import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinTransactionsPage extends BasePage {
  readonly path = "/fin/transactions";

  get newTransactionButton(): Locator {
    return this.page.getByRole("button", { name: "New transaction" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  get searchInput(): Locator {
    return this.page.getByPlaceholder(/Search number or description/i);
  }

  // Form locators
  get saveButton(): Locator {
    return this.page.getByRole("button", { name: "Save" });
  }

  get descriptionInput(): Locator {
    return this.page.locator("#description");
  }

  get referenceInput(): Locator {
    return this.page.locator("#reference");
  }

  get currencySelect(): Locator {
    return this.page.locator(".ant-select").first();
  }

  get exchangeRateInput(): Locator {
    return this.page.locator(".ant-input-number-input").first();
  }

  get attachmentFileInput(): Locator {
    return this.page.locator('input[type="file"]').first();
  }

  async attachFile(filename: string, content: string): Promise<void> {
    await this.attachmentFileInput.setInputFiles({
      name: filename,
      mimeType: filename.endsWith(".pdf") ? "application/pdf" : "image/png",
      buffer: Buffer.from(content, "utf-8"),
    });
  }

  async clickNewTransaction(): Promise<void> {
    await this.newTransactionButton.waitFor({ state: "visible" });
    await this.newTransactionButton.click();
  }

  async expectDrawerVisible(): Promise<void> {
    await this.descriptionInput.waitFor({ state: "visible" });
  }

  async fillDescription(desc: string): Promise<void> {
    await this.descriptionInput.fill(desc);
  }

  async fillReference(ref: string): Promise<void> {
    await this.referenceInput.fill(ref);
  }

  async selectCurrency(currency: string): Promise<void> {
    await this.currencySelect.click();
    await this.page.keyboard.type(currency);
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
    await option.waitFor({ state: "hidden" });
  }

  async fillExchangeRate(rate: number): Promise<void> {
    await this.exchangeRateInput.fill(String(rate));
  }

  async saveDrawer(): Promise<void> {
    await this.saveButton.click();
    await this.page.waitForURL((url) => !url.pathname.endsWith("/new"));
  }

  async saveForm(): Promise<void> {
    await this.saveButton.click();
    await this.page.waitForURL((url) => !url.pathname.endsWith("/new"));
  }

  async expectTransactionInTable(text: string): Promise<void> {
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
    await row.first().waitFor({ state: "visible" });
    await row.first().click();
  }

  async search(query: string): Promise<void> {
    await this.searchInput.fill(query);
  }

  async expectNewTransactionVisible(): Promise<void> {
    await this.newTransactionButton.waitFor({ state: "visible" });
  }

  async expectNewTransactionHidden(): Promise<void> {
    await this.newTransactionButton.waitFor({ state: "hidden" });
  }
}
