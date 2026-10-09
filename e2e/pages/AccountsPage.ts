import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class AccountsPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async open() {
    await this.goto("/fin/accounts");
  }

  get createAccountButton() {
    return this.page.getByRole("button", {
      name: /Create Account|Tambah Akun/i,
    });
  }

  get table() {
    return this.page.locator(".ant-table");
  }

  async clickCreateAccount() {
    await this.createAccountButton.first().waitFor({ state: "visible" });
    await this.createAccountButton.first().click();
  }

  async expectAccountRow(code: string) {
    const row = this.page.locator(".ant-table-row").filter({ hasText: code });
    await row.first().waitFor({ state: "visible", timeout: 10000 });
    return row.first();
  }

  async clickEditAccount(code: string) {
    const row = await this.expectAccountRow(code);
    const editBtn = row.locator("button").last();
    await editBtn.waitFor({ state: "visible" });
    await editBtn.click();
  }

  async clickAccountCode(code: string) {
    const row = await this.expectAccountRow(code);
    const codeBtn = row.getByRole("button", { name: code });
    await codeBtn.waitFor({ state: "visible" });
    await codeBtn.click();
  }
}
