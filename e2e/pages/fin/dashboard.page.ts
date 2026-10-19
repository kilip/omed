import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinDashboardPage extends BasePage {
  readonly path = "/fin";

  get setupBanner(): Locator {
    return this.page.locator(".surface", {
      hasText: "Chart of accounts not configured",
    });
  }

  get setupButton(): Locator {
    return this.page.getByRole("button", { name: "Setup chart of accounts" });
  }

  async expectSetupBannerVisible(): Promise<void> {
    await this.setupBanner.waitFor({ state: "visible" });
  }

  async clickSetupButton(): Promise<void> {
    await this.setupButton.waitFor({ state: "visible" });
    await this.setupButton.click();
  }
}
