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

  get emailInput(): Locator {
    return this.page.getByLabel("Email");
  }

  get sendCodeButton(): Locator {
    return this.page.getByRole("button", { name: "Send code" });
  }

  get codeInput(): Locator {
    return this.page.locator(".ant-otp");
  }

  get resendButton(): Locator {
    return this.page.getByRole("button", { name: /^Resend code/ });
  }

  get invalidCodeMessage(): Locator {
    return this.page.getByText("That code is invalid or expired.");
  }

  async fillCode(code: string): Promise<void> {
    const inputs = this.codeInput.locator("input");
    await inputs.first().waitFor({ state: "visible" });
    await inputs.first().focus();
    for (const char of code) {
      await this.page.keyboard.type(char);
    }
  }

  async expectLoaded(): Promise<void> {
    await this.title.waitFor({ state: "visible" });
    await this.googleButton.waitFor({ state: "visible" });
    await this.githubButton.waitFor({ state: "visible" });
  }
}
