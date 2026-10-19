import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinSetupPage extends BasePage {
  readonly path = "/fin/setup";

  get applyButton(): Locator {
    return this.page.getByRole("button", { name: "Apply template" });
  }

  async clickApplyTemplate(): Promise<void> {
    await this.applyButton.waitFor({ state: "visible" });
    await this.applyButton.click();
  }
}
