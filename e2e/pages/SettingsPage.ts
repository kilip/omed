import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class SettingsPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  get currencySelect() {
    return this.page.locator(".ant-select");
  }

  get currencyContent() {
    return this.currencySelect.locator(".ant-select-content");
  }

  async open() {
    await this.goto("/settings");
  }

  async getSelectedCurrency(): Promise<string> {
    await this.currencySelect.waitFor({ state: "visible", timeout: 10000 });
    await this.currencyContent.waitFor({ state: "visible", timeout: 10000 });
    return await this.currencyContent.innerText();
  }

  async setCurrency(currency: string) {
    await this.currencySelect.waitFor({ state: "visible", timeout: 10000 });
    await this.currencySelect.click();

    const option = this.page
      .locator(".ant-select-item-option")
      .filter({ hasText: currency });
    await option.waitFor({ state: "visible", timeout: 10000 });
    await option.click();
  }

  async selectLanguageRadio(languageLabel: string) {
    const radio = this.page
      .locator("label.ant-radio-wrapper")
      .filter({ hasText: languageLabel });
    await radio.waitFor({ state: "visible" });
    await radio.click();
  }

  async selectThemeRadio(themeLabel: string) {
    const radio = this.page
      .locator("label.ant-radio-wrapper")
      .filter({ hasText: themeLabel });
    await radio.waitFor({ state: "visible" });
    await radio.click();
  }

  async expectNoticeMessage(messageText: string) {
    const msg = this.page.locator(".ant-message-notice").filter({
      hasText: messageText,
    });
    await msg.waitFor({ state: "visible" });
    return msg;
  }

  async getHtmlThemeAttribute(): Promise<string | null> {
    return await this.page.locator("html").getAttribute("data-theme");
  }
}
