import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

/** The module launcher shown at `/` once a session exists. */
export class HomePage extends BasePage {
  readonly path = "/";

  get title(): Locator {
    return this.page.getByRole("heading", {
      name: /^(Apps|Aplikasi)$/,
      exact: true,
    });
  }

  /** Avatar button in the shell header; its accessible name is the user's name. */
  accountButton(userName: string): Locator {
    return this.page.getByRole("button", { name: userName });
  }

  get languageButton(): Locator {
    return this.page
      .locator("header")
      .getByRole("button", { name: /^(Language|Bahasa)$/i });
  }

  async expectRendered(userName: string): Promise<void> {
    await this.title.waitFor({ state: "visible" });
    await this.accountButton(userName).waitFor({ state: "visible" });
  }

  async clickLanguageMenu(): Promise<void> {
    await this.languageButton.waitFor({ state: "visible" });
    await this.languageButton.click();
  }

  async selectLanguage(language: string): Promise<void> {
    const item = this.page
      .locator(
        ".ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item",
      )
      .filter({ hasText: language });
    await item.waitFor({ state: "visible" });
    await item.click();
  }

  async switchLanguage(language: string): Promise<void> {
    const updatePromise = this.page.waitForResponse(
      (res) =>
        res.url().includes("/users/me") && res.request().method() === "PUT",
    );
    await this.clickLanguageMenu();
    await this.selectLanguage(language);
    await updatePromise;
  }

  async expectLanguage(code: "EN" | "ID"): Promise<void> {
    await this.languageButton.waitFor({ state: "visible" });
    let text = (await this.languageButton.textContent())?.trim();
    const start = Date.now();
    while (text !== code && Date.now() - start < 5000) {
      await this.page.waitForTimeout(100);
      text = (await this.languageButton.textContent())?.trim();
    }
    assert.equal(
      text,
      code,
      `Expected language button text to be "${code}", got "${text}"`,
    );
  }

  async expectHeading(title: string): Promise<void> {
    const heading = this.page.getByRole("heading", {
      name: title,
      exact: true,
    });
    await heading.waitFor({ state: "visible" });
  }
}
