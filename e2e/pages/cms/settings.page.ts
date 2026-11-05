import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class CmsSettingsPage extends BasePage {
  readonly path = "/cms/settings";

  get defaultLocaleSelect(): Locator {
    return this.page.locator("#defaultLocale");
  }

  get supportedLocalesSelect(): Locator {
    return this.page.locator("#supportedLocales");
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: /^save$/i });
  }

  get errorAlert(): Locator {
    return this.page.locator(".ant-alert-error");
  }

  async goto(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async expectLoaded(): Promise<void> {
    await this.saveButton.waitFor({ state: "visible", timeout: 10_000 });
  }

  async selectDefaultLocale(locale: string): Promise<void> {
    const selectWrapper = this.page.locator(".ant-select", {
      has: this.defaultLocaleSelect,
    });
    await selectWrapper.click();
    const dropdownOption = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: new RegExp(locale, "i") });
    await dropdownOption.first().waitFor({ state: "visible", timeout: 5000 });
    await dropdownOption.first().click();
  }

  async addSupportedLocale(locale: string): Promise<void> {
    const selectWrapper = this.page.locator(".ant-select", {
      has: this.supportedLocalesSelect,
    });
    const existing = selectWrapper
      .locator(".ant-select-selection-item")
      .filter({ hasText: new RegExp(locale, "i") });
    if ((await existing.count()) > 0) {
      return;
    }
    await selectWrapper.click();
    const dropdownOption = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: new RegExp(locale, "i") });
    try {
      await dropdownOption.first().waitFor({ state: "visible", timeout: 1500 });
      await dropdownOption.first().click();
    } catch {
      await this.supportedLocalesSelect.fill(locale);
      await this.supportedLocalesSelect.press("Enter");
    }
    await this.page.keyboard.press("Escape");
  }

  async save(): Promise<void> {
    await this.saveButton.waitFor({ state: "visible" });
    await this.saveButton.click();
  }

  async expectSuccess(): Promise<void> {
    const success = this.page.locator(".ant-message-success");
    await success.first().waitFor({ state: "visible", timeout: 10_000 });
  }

  async expectError(regex: RegExp): Promise<void> {
    const error = this.page
      .locator(".ant-alert-error, .ant-message-error")
      .filter({ hasText: regex });
    await error.first().waitFor({ state: "visible", timeout: 10_000 });
  }
}
