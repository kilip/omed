import { Given, Then } from "@cucumber/cucumber";
import { expect } from "playwright/test";
import { DashboardPage } from "../pages/DashboardPage";
import { LoginPage } from "../pages/LoginPage";
import { type AuthRole, loginAsRole } from "../support/auth-helper";
import type { CustomWorld } from "../support/world";

Given("I am an unauthenticated user", async function (this: CustomWorld) {
  if (!this.context) throw new Error("Playwright context not initialized");
  await this.context.clearCookies();
  this.currentUser = undefined;
});

Given(
  "I am logged in as {string}",
  async function (this: CustomWorld, role: string) {
    if (!this.context) throw new Error("Playwright context not initialized");
    const result = await loginAsRole(this.context, role as AuthRole);
    this.currentUser = result.user;
    this.addCleanupTask(result.cleanup);
  },
);

Then(
  "I should see the login card with title {string}",
  async function (this: CustomWorld, title: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const loginPage = new LoginPage(this.page);
    await expect(loginPage.loginCardTitle).toBeVisible();
    await expect(loginPage.loginCardTitle).toHaveText(title);
  },
);

Then(
  "I should see social login buttons for Google and GitHub",
  async function (this: CustomWorld) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const loginPage = new LoginPage(this.page);
    await expect(loginPage.googleLoginButton).toBeVisible();
    await expect(loginPage.githubLoginButton).toBeVisible();
  },
);

Then("I should see the dashboard layout", async function (this: CustomWorld) {
  if (!this.page) throw new Error("Playwright page not initialized");
  const dashboardPage = new DashboardPage(this.page);
  const isDisplayed = await dashboardPage.isDashboardDisplayed();
  expect(isDisplayed).toBe(true);
});

Then(
  "I should see the user name on the page",
  async function (this: CustomWorld) {
    if (!this.page) throw new Error("Playwright page not initialized");
    if (!this.currentUser) throw new Error("No current user logged in");
    await expect(
      this.page.getByText(this.currentUser.name).first(),
    ).toBeVisible();
  },
);
