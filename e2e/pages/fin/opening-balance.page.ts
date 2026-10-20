import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinOpeningBalancePage extends BasePage {
  readonly path = "/fin/opening-balance";

  get saveButton(): Locator {
    return this.page.getByRole("button", {
      name: /Save Opening Balance|Simpan Saldo Awal/i,
    });
  }

  async inputAccountBalance(code: string, amount: string): Promise<void> {
    const input = this.page.getByLabel(`balance-${code}`);
    await input.waitFor({ state: "visible" });
    await input.click();
    await input.fill(amount);
    await input.press("Enter");
  }

  async clickSave(): Promise<void> {
    await this.saveButton.waitFor({ state: "visible" });
    await this.saveButton.click();
  }

  async expectReentryNoticeVisible(): Promise<void> {
    await this.page.locator(".ant-alert-info").waitFor({ state: "visible" });
  }
}
