import type { Locator, Page } from "playwright";
import { BasePage } from "./BasePage";

export class LoginPage extends BasePage {
  readonly loginCardTitle: Locator;
  readonly loginSubtitle: Locator;
  readonly googleLoginButton: Locator;
  readonly githubLoginButton: Locator;

  constructor(page: Page) {
    super(page);
    this.loginCardTitle = page.getByRole("heading", {
      name: "Login to Omed",
      level: 3,
    });
    this.loginSubtitle = page.getByText("Sign in to continue to your account");
    this.googleLoginButton = page.getByRole("button", {
      name: /sign in with google/i,
    });
    this.githubLoginButton = page.getByRole("button", {
      name: /sign in with github/i,
    });
  }

  async isLoginPageDisplayed(): Promise<boolean> {
    return await this.loginCardTitle.isVisible();
  }
}
