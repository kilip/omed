import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class LayoutPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  getNavigationItem(label: string) {
    return this.page
      .getByRole("menuitem", { name: label })
      .or(this.page.locator(".ant-menu-item").filter({ hasText: label }));
  }

  async clickNavigationItem(label: string) {
    const item = this.getNavigationItem(label);
    await item.waitFor({ state: "visible" });
    await item.click();
  }

  async switchLanguage(languageLabel: string) {
    const switcher = this.page.locator("button").filter({
      has: this.page.locator(".anticon-global"),
    });
    await switcher.waitFor({ state: "visible" });
    await switcher.click();

    const menuItem = this.page
      .locator(".ant-dropdown-menu-item")
      .filter({ hasText: languageLabel });
    await menuItem.waitFor({ state: "visible" });
    await menuItem.click();
  }
}
