import { Given, Then, When } from "@cucumber/cucumber";
import { expect } from "expect";
import { e2eEnv } from "../env";
import { LoginPage } from "../pages";
import type { CustomWorld } from "./world";

Given("I am not signed in", async function (this: CustomWorld) {
  await this.context.clearCookies();
});

Given("I am signed in", async function (this: CustomWorld) {
  await this.signInUser();
});

When("I open {string}", async function (this: CustomWorld, url: string) {
  await this.page.goto(url);
});

Then(
  "I should be on {string}",
  async function (this: CustomWorld, expectedPath: string) {
    const loginPage = new LoginPage(this.page);
    await loginPage.expectUrlPath(expectedPath);
    const currentPath = new URL(this.page.url()).pathname;
    expect(currentPath).toBe(expectedPath);
  },
);

Then(
  "I should see the heading {string}",
  async function (this: CustomWorld, expectedText: string) {
    const loginPage = new LoginPage(this.page);
    const heading = await loginPage.expectHeading(expectedText);
    expect(await heading.isVisible()).toBe(true);
  },
);

Then(
  "I should see the button {string}",
  async function (this: CustomWorld, expectedText: string) {
    const loginPage = new LoginPage(this.page);
    const button = await loginPage.expectButton(expectedText);
    expect(await button.isVisible()).toBe(true);
  },
);

When(
  "I open {string} in a fresh browser session",
  async function (this: CustomWorld, url: string) {
    const cookies = await this.context.cookies();
    await this.context.close();
    this.context = await this.browser.newContext({
      baseURL: e2eEnv.DASH_URL,
      locale: "en-US",
    });
    await this.context.addCookies(cookies);
    this.page = await this.context.newPage();
    await this.page.goto(url);
  },
);
