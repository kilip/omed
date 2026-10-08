import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class SeedCoaPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async open() {
    await this.goto("/fin/seed-coa");
  }

  get searchInput() {
    return this.page.locator(".ant-input-search input");
  }

  get previewTable() {
    return this.page.locator(".ant-table");
  }

  get submitButton() {
    return this.page.getByRole("button", {
      name: /Seed Chart of Accounts|Inisialisasi Bagan Akun/i,
    });
  }

  get resultSuccess() {
    return this.page.locator(".ant-result-success");
  }

  get goToFinanceButton() {
    return this.page.getByRole("button", {
      name: /Go to Finance|Buka Keuangan/i,
    });
  }

  get overwriteCheckbox() {
    return this.page.locator(".ant-checkbox-input");
  }

  async searchAccount(query: string) {
    await this.searchInput.waitFor({ state: "visible" });
    await this.searchInput.fill(query);
  }

  async clickSeedAccounts() {
    await this.submitButton.waitFor({ state: "visible" });
    await this.submitButton.click();
  }

  async expectSuccessResult() {
    await this.resultSuccess.waitFor({ state: "visible", timeout: 15000 });
  }

  async clickGoToFinance() {
    await this.goToFinanceButton.waitFor({ state: "visible" });
    await this.goToFinanceButton.click();
  }
}
