import type { Locator } from "playwright";
import { BasePage } from "../base.page";

/** The module launcher shown at `/` once a session exists. */
export class HomePage extends BasePage {
  readonly path = "/";

  get title(): Locator {
    return this.page.getByRole("heading", { name: "Apps", exact: true });
  }

  /** Avatar button in the shell header; its accessible name is the user's name. */
  accountButton(userName: string): Locator {
    return this.page.getByRole("button", { name: userName });
  }

  async expectRendered(userName: string): Promise<void> {
    await this.title.waitFor({ state: "visible" });
    await this.accountButton(userName).waitFor({ state: "visible" });
  }
}
