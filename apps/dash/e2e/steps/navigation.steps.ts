import { When } from "@cucumber/cucumber";
import { DashboardPage } from "../pages/DashboardPage";
import type { CustomWorld } from "../support/world";

When(
  "I click sidebar menu item {string}",
  async function (this: CustomWorld, label: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const dashboardPage = new DashboardPage(this.page);
    await dashboardPage.clickMenuItem(label);
  },
);

When(
  "I open sidebar submenu {string}",
  async function (this: CustomWorld, label: string) {
    if (!this.page) throw new Error("Playwright page not initialized");
    const dashboardPage = new DashboardPage(this.page);
    await dashboardPage.openSubmenu(label);
  },
);

When("I click the logout button", async function (this: CustomWorld) {
  if (!this.page) throw new Error("Playwright page not initialized");
  const dashboardPage = new DashboardPage(this.page);
  await dashboardPage.logout();
});
