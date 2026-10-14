import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class LoginPage extends BasePage {
  readonly path = "/login";

  get title(): Locator {
    return this.page.getByRole("heading", { name: "Sign in to Omed" });
  }

  get googleButton(): Locator {
    return this.page.getByRole("button", { name: "Continue with Google" });
  }

  get githubButton(): Locator {
    return this.page.getByRole("button", { name: "Continue with GitHub" });
  }

  async expectLoaded(): Promise<void> {
    await this.title.waitFor({ state: "visible" });
    await this.googleButton.waitFor({ state: "visible" });
    await this.githubButton.waitFor({ state: "visible" });
  }
}
