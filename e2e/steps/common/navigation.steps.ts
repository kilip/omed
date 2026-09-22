import { Given, Then } from "@cucumber/cucumber";
import type { CustomWorld } from "@e2e/support/world";
import { expect } from "playwright/test";

Given("I go to {string}", async function (this: CustomWorld, path: string) {
  const response = await this.page.goto(path);
  this.testContext.lastResponse = response;

  await this.page.waitForLoadState("domcontentloaded");
});

Then(
  "I should be on {string}",
  async function (this: CustomWorld, path: string) {
    expect(this.page.url()).toContain(path);
  },
);

Then("I should see {string}", async function (this: CustomWorld, text: string) {
  await expect(this.page.locator("body")).toContainText(text);
});

Then(
  "page title should contain {string}",
  async function (this: CustomWorld, text: string) {
    const titleRegex = new RegExp(text, "i");

    await expect(this.page).toHaveTitle(titleRegex);
  },
);
