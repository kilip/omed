import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinSetupPage extends BasePage {
  readonly path = "/fin/setup";

  get applyButton(): Locator {
    return this.page.getByRole("button", { name: "Apply template" });
  }

  get fiscalYearInput(): Locator {
    return this.page.getByRole("spinbutton", { name: "Fiscal year" });
  }

  get periodNameInput(): Locator {
    return this.page.getByRole("textbox", { name: "Period name" });
  }

  async clickApplyTemplate(): Promise<void> {
    await this.applyButton.waitFor({ state: "visible" });
    await this.applyButton.click();
  }

  async expectPeriodFormVisible(): Promise<void> {
    await this.fiscalYearInput.waitFor({ state: "visible" });
    await this.periodNameInput.waitFor({ state: "visible" });
  }
}
