import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class InvitePage extends BasePage {
  readonly path = "/invite";

  get acceptButton(): Locator {
    return this.page.getByRole("button", { name: "Accept invitation" });
  }

  get invalidAlert(): Locator {
    return this.page.getByRole("alert");
  }

  async accept(): Promise<void> {
    await this.acceptButton.click();
  }

  async expectCanAccept(): Promise<void> {
    await this.acceptButton.waitFor({ state: "visible" });
  }
}
