import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

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
  get saveButton(): Locator {
    return this.page.getByRole("button", { name: /Save (Expense|Income)/i });
  }

  get amountInput(): Locator {
    return this.page.locator("#amount");
  }

  get descriptionInput(): Locator {
    return this.page.locator("#description");
  }

  get referenceInput(): Locator {
    return this.page.locator("#reference");
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
    const select = this.page.locator(
      `.ant-select:has(#${selectId}) .ant-select-selector`,
    );
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
    await this.page.waitForTimeout(100);
  }

  async selectPaymentAccount(accountQuery: string): Promise<void> {
    await this.selectOption("paymentAccountId", accountQuery);
  }

  async selectDepositAccount(accountQuery: string): Promise<void> {
    await this.selectPaymentAccount(accountQuery);
  }

  async selectExpenseAccount(accountQuery: string): Promise<void> {
    await this.selectOption("expenseAccountId", accountQuery);
  }

  async selectRevenueAccount(accountQuery: string): Promise<void> {
    await this.selectOption("revenueAccountId", accountQuery);
  }

  async fillAmount(amount: number | string): Promise<void> {
    await this.amountInput.waitFor({ state: "visible" });
    await this.amountInput.fill(String(amount));
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

  async saveForm(): Promise<void> {
    await this.saveButton.waitFor({ state: "visible" });
    await this.saveButton.click();
    await this.page.waitForURL((url) => !url.pathname.endsWith("/new"), {
      timeout: 10_000,
    });
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
}
