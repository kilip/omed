import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class LoginPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  get googleLoginButton() {
    return this.page.getByRole("button", { name: "Continue with Google" });
  }

  get githubLoginButton() {
    return this.page.getByRole("button", { name: "Continue with GitHub" });
  }

  async open() {
    await this.goto("/login");
  }
}
