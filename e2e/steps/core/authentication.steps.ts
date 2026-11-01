import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import { expect } from "playwright/test";
import type { OmedWorld } from "../../support/world";

Then("I see the login page", async function (this: OmedWorld) {
  await this.pages.core.login.expectLoaded();
});

Given("the mailbox is empty", async function (this: OmedWorld) {
  await this.client.resetEmails();
});

When(
  "I request a sign-in code for a new email",
  async function (this: OmedWorld) {
    this.otpEmail = `otp-${Date.now()}@example.com`;
    await this.pages.core.login.emailInput.fill(this.otpEmail);
    await this.pages.core.login.sendCodeButton.click();
  },
);

When(
  "I request a sign-in code for the existing user",
  async function (this: OmedWorld) {
    if (!this.user) {
      this.user = await this.client.ensureUser({
        email: `otp-existing-${Date.now()}@example.com`,
        name: "Existing User",
      });
    }
    this.otpEmail = this.user.email;
    await this.pages.core.login.emailInput.fill(this.otpEmail);
    await this.pages.core.login.sendCodeButton.click();
  },
);

When("I enter the code from the email", async function (this: OmedWorld) {
  assert.ok(this.otpEmail, "otpEmail must be set to retrieve code");
  const code = await this.client.waitForOtpCode(this.otpEmail);
  await this.pages.core.login.fillCode(code);
});

When(
  "I enter the code {string}",
  async function (this: OmedWorld, code: string) {
    await this.pages.core.login.fillCode(code);
  },
);

Then("I see the invalid code message", async function (this: OmedWorld) {
  await expect(this.pages.core.login.invalidCodeMessage).toBeVisible();
});

Then("the resend button is disabled", async function (this: OmedWorld) {
  await expect(this.pages.core.login.resendButton).toBeDisabled();
});
