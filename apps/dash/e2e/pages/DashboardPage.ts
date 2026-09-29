import type { Locator, Page } from "playwright";
import { BasePage } from "./BasePage";

export class DashboardPage extends BasePage {
  readonly sider: Locator;
  readonly header: Locator;
  readonly content: Locator;
  readonly brandLogo: Locator;
  readonly themeToggle: Locator;
  readonly sidebarToggle: Locator;

  constructor(page: Page) {
    super(page);
    this.sider = page.locator(".ant-layout-sider");
    this.header = page.locator(".ant-layout-header");
    this.content = page.locator(".ant-layout-content");
    this.brandLogo = page.locator("img[alt='Omed']");
    this.sidebarToggle = this.header.locator("button").first();
    this.themeToggle = this.header.locator("button").nth(1);
  }

  async isDashboardDisplayed(): Promise<boolean> {
    await this.sider.waitFor({ state: "visible" });
    return (await this.sider.isVisible()) && (await this.header.isVisible());
  }

  async clickMenuItem(label: string) {
    const menuItem = this.sider
      .locator(".ant-menu-item")
      .filter({ hasText: label });
    await menuItem.click();
  }

  async openSubmenu(label: string) {
    const submenu = this.sider
      .locator(".ant-menu-submenu-title")
      .filter({ hasText: label });
    await submenu.click();
  }

  async logout() {
    // Open user dropdown if exists or trigger logout
    const dropdownTrigger = this.header.locator(".ant-dropdown-trigger");
    if (await dropdownTrigger.isVisible()) {
      await dropdownTrigger.click();
      const logoutItem = this.page
        .locator(".ant-dropdown-menu-item")
        .filter({ hasText: /log out/i });
      await logoutItem.click();
    }
  }
}
