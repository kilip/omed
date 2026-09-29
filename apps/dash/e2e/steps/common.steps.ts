import { Given, Then, When } from "@cucumber/cucumber";
import { expect } from "playwright/test";
import { BasePage } from "../pages/BasePage";
import type { CustomWorld } from "../support/world";

Given(
  "I navigate to {string}",
  async function (this: CustomWorld, path: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const basePage = new BasePage(this.page);
    await basePage.navigate(path);
  },
);

When(
  "I navigate to protected page {string}",
  async function (this: CustomWorld, path: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const basePage = new BasePage(this.page);
    await basePage.navigate(path);
  },
);

Then(
  "I should be redirected to {string}",
  async function (this: CustomWorld, expectedPath: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const basePage = new BasePage(this.page);
    await basePage.waitForUrl(new RegExp(expectedPath));
    const currentPath = await basePage.getPathname();
    expect(currentPath).toBe(expectedPath);
  },
);

Then(
  "I should see text {string}",
  async function (this: CustomWorld, text: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    await expect(this.page.getByText(text).first()).toBeVisible();
  },
);
