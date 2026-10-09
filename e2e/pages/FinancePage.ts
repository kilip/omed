import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class FinancePage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async open() {
    await this.goto("/fin");
  }

  get onboardingCard() {
    return this.page.locator(".ant-card");
  }

  get setupCoaButton() {
    return this.page.getByRole("button", {
      name: /Setup Chart of Accounts|Setup Bagan Akun/i,
    });
  }

  get viewAccountsButton() {
    return this.page.getByRole("button", {
      name: /View Accounts|Lihat Akun/i,
    });
  }

  async clickSetupCoa() {
    await this.setupCoaButton.waitFor({ state: "visible" });
    await this.setupCoaButton.click();
  }

  async clickViewAccounts() {
    await this.viewAccountsButton.waitFor({ state: "visible" });
    await this.viewAccountsButton.click();
  }
}
