import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export interface CashflowItemInput {
  account?: string;
  description?: string;
  quantity?: number | string;
  unitPrice?: number | string;
}

export class FinIncomeExpensePage extends BasePage {
  readonly path = "/fin/expense";

  // List locators
  get newExpenseButton(): Locator {
    return this.page.getByRole("button", { name: /New Expense/i });
  }

  get newIncomeButton(): Locator {
    return this.page.getByRole("button", { name: /New Income/i });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  get searchInput(): Locator {
    return this.page.getByPlaceholder(/Search/i);
  }

  // Form locators
  get descriptionInput(): Locator {
    return this.page.locator("#description");
  }

  get referenceInput(): Locator {
    return this.page.locator("#reference");
  }

  get addItemButton(): Locator {
    return this.page.getByRole("button", { name: /Add item/i });
  }

  get postingButton(): Locator {
    return this.page.getByRole("button", { name: "Posting" });
  }

  get draftButton(): Locator {
    return this.page.getByRole("button", { name: "Draft" });
  }

  get editButton(): Locator {
    return this.page.getByRole("button", { name: "Edit" });
  }

  get accountingPreview(): Locator {
    return this.page.locator('[data-testid="cashflow-double-entry-preview"]');
  }

  async clickNewExpense(): Promise<void> {
    await this.newExpenseButton.waitFor({ state: "visible" });
    await this.newExpenseButton.click();
  }

  async clickNewIncome(): Promise<void> {
    await this.newIncomeButton.waitFor({ state: "visible" });
    await this.newIncomeButton.click();
  }

  async selectOption(selectId: string, query: string): Promise<void> {
    const select = this.page.locator(`.ant-select:has(#${selectId})`);
    await select.waitFor({ state: "visible" });
    await select.click();
    await this.page.keyboard.type(query);
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: query })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
    await option.waitFor({ state: "hidden" });
  }

  async selectPaymentAccount(accountQuery: string): Promise<void> {
    await this.selectOption("paymentAccountId", accountQuery);
  }

  async selectDepositAccount(accountQuery: string): Promise<void> {
    await this.selectPaymentAccount(accountQuery);
  }

  async addItem(): Promise<void> {
    await this.addItemButton.waitFor({ state: "visible" });
    await this.addItemButton.click();
  }

  async fillItem(index: number, item: CashflowItemInput): Promise<void> {
    if (item.account) {
      await this.selectOption(`items_${index}_accountId`, item.account);
    }
    if (item.description !== undefined) {
      const desc = this.page.locator(`#items_${index}_description`);
      await desc.waitFor({ state: "visible" });
      await desc.fill(item.description);
    }
    if (item.quantity !== undefined) {
      const qty = this.page.locator(`#items_${index}_quantity`);
      await qty.waitFor({ state: "visible" });
      await qty.fill(String(item.quantity));
    }
    if (item.unitPrice !== undefined) {
      const price = this.page.locator(`#items_${index}_unitPrice`);
      await price.waitFor({ state: "visible" });
      await price.fill(String(item.unitPrice));
    }
  }

  async clickPosting(): Promise<void> {
    await this.postingButton.waitFor({ state: "visible" });
    await this.postingButton.click();
    await this.page.waitForURL(
      (url) =>
        !url.pathname.endsWith("/new") && !url.pathname.endsWith("/edit"),
      { timeout: 10_000 },
    );
  }

  async clickDraft(): Promise<void> {
    await this.draftButton.waitFor({ state: "visible" });
    await this.draftButton.click();
    await this.page.waitForURL(
      (url) =>
        !url.pathname.endsWith("/new") && !url.pathname.endsWith("/edit"),
      { timeout: 10_000 },
    );
  }

  async clickEdit(): Promise<void> {
    await this.editButton.waitFor({ state: "visible" });
    await this.editButton.click();
    await this.page.waitForURL((url) => url.pathname.endsWith("/edit"), {
      timeout: 10_000,
    });
  }

  async fillDescription(desc: string): Promise<void> {
    await this.descriptionInput.waitFor({ state: "visible" });
    await this.descriptionInput.fill(desc);
  }

  async fillReference(ref: string): Promise<void> {
    await this.referenceInput.waitFor({ state: "visible" });
    await this.referenceInput.fill(ref);
  }

  async expectPreviewText(text: string): Promise<void> {
    await this.accountingPreview.waitFor({ state: "visible" });
    const content = await this.accountingPreview.innerText();
    assert.ok(
      content.includes(text),
      `Expected accounting preview to contain "${text}", got "${content}"`,
    );
  }

  async expectStatusTag(text: string): Promise<void> {
    const tag = this.page.locator(".ant-tag").filter({ hasText: text });
    await tag.first().waitFor({ state: "visible" });
    assert.ok(
      await tag.first().isVisible(),
      `Expected status tag "${text}" to be visible`,
    );
  }

  async clickRow(text: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    await row.first().click();
  }

  async expectItemInTable(text: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    const rowText = await row.first().innerText();
    assert.ok(
      rowText.includes(text),
      `Expected row to contain "${text}", got "${rowText}"`,
    );
  }

  async search(query: string): Promise<void> {
    await this.searchInput.fill(query);
  }

  async expectEmptyStateVisible(): Promise<void> {
    await this.emptyState.waitFor({ state: "visible" });
  }

  get detailTableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get detailTotal(): Locator {
    return this.page.locator(".border-t").filter({ hasText: "Total:" });
  }

  get uploadAttachmentButton(): Locator {
    return this.page.getByRole("button", { name: "Upload attachment" });
  }

  get attachmentFileInput(): Locator {
    return this.page.locator('input[type="file"]').first();
  }

  get attachmentItems(): Locator {
    return this.page.getByTestId("transaction-attachment-item");
  }

  get popconfirmOkButton(): Locator {
    return this.page
      .locator(".ant-popconfirm")
      .getByRole("button", { name: "OK" });
  }

  async expectDetailLineItem(text: string): Promise<void> {
    const row = this.detailTableRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    const rowText = await row.first().innerText();
    assert.ok(
      rowText.includes(text),
      `Expected detail line item to contain "${text}", got "${rowText}"`,
    );
  }

  async expectDetailTotal(amount: string): Promise<void> {
    await this.detailTotal.waitFor({ state: "visible" });
    const text = await this.detailTotal.innerText();
    assert.ok(
      text.includes(amount),
      `Expected detail total to contain "${amount}", got "${text}"`,
    );
  }

  async uploadAttachment(filename: string, content: string): Promise<void> {
    await this.attachmentFileInput.setInputFiles({
      name: filename,
      mimeType: filename.endsWith(".pdf") ? "application/pdf" : "image/png",
      buffer: Buffer.from(content, "utf-8"),
    });
  }

  async expectAttachmentVisible(filename: string): Promise<void> {
    const item = this.attachmentItems.filter({ hasText: filename });
    await item.first().waitFor({ state: "visible" });
  }

  async expectAttachmentNotVisible(filename: string): Promise<void> {
    const item = this.attachmentItems.filter({ hasText: filename });
    await item.first().waitFor({ state: "hidden" });
  }

  async deleteAttachment(filename: string): Promise<void> {
    const item = this.attachmentItems.filter({ hasText: filename }).first();
    await item.waitFor({ state: "visible" });
    await item.getByRole("button", { name: "Delete attachment" }).click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }

  async expectReadOnlyDetail(): Promise<void> {
    const editCount = await this.editButton.count();
    assert.equal(editCount, 0, "Expected edit button to be hidden");
    const uploadCount = await this.uploadAttachmentButton.count();
    assert.equal(
      uploadCount,
      0,
      "Expected upload attachment button to be hidden",
    );
  }
}
