import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class CmsPagesPage extends BasePage {
  readonly path = "/cms/pages";

  get newPageButton(): Locator {
    return this.page.getByRole("button", { name: /new page/i });
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: /^save$/i });
  }

  get publishNowButton(): Locator {
    return this.page.getByRole("button", { name: /publish now/i });
  }

  get scheduleButton(): Locator {
    return this.page.getByRole("button", { name: /schedule\.\.\./i });
  }

  get scheduleConfirmButton(): Locator {
    return this.page
      .locator(".ant-modal")
      .getByRole("button", { name: /^schedule$/i });
  }

  get unpublishButton(): Locator {
    return this.page.getByRole("button", { name: /unpublish/i });
  }

  get deleteButton(): Locator {
    return this.page.getByRole("button", { name: /delete/i });
  }

  get deleteTranslationButton(): Locator {
    return this.page.getByRole("button", { name: /delete translation/i });
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover .ant-btn-primary",
    );
  }

  get titleInput(): Locator {
    return this.page.locator("#title");
  }

  get slugInput(): Locator {
    return this.page.locator("#slug");
  }

  get bodyInput(): Locator {
    return this.page.locator(".ant-input").last();
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  async goto(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async expectLoaded(): Promise<void> {
    await this.page.waitForSelector(".ant-table", {
      state: "visible",
      timeout: 10_000,
    });
  }

  async clickNewPage(): Promise<void> {
    await this.newPageButton.waitFor({ state: "visible" });
    await this.newPageButton.click();
    try {
      await this.titleInput.waitFor({ state: "visible", timeout: 5000 });
    } catch {
      if (await this.newPageButton.isVisible()) {
        await this.newPageButton.click();
      }
      await this.titleInput.waitFor({ state: "visible" });
    }
  }

  async fillPage(data: {
    title: string;
    slug?: string;
    body?: string;
  }): Promise<void> {
    await this.titleInput.fill(data.title);
    if (data.slug) {
      await this.slugInput.fill(data.slug);
    }
    if (data.body) {
      await this.bodyInput.fill(data.body);
    }
  }

  async savePage(): Promise<void> {
    const isNew = this.page.url().includes("/cms/pages/new");
    await this.saveButton.click();
    if (isNew) {
      await this.page.waitForURL(/\/cms\/pages\/\d+/);
    } else {
      await this.page
        .locator(".ant-message-success")
        .first()
        .waitFor({ state: "visible" });
    }
  }

  async publishPage(): Promise<void> {
    await this.publishNowButton.waitFor({ state: "visible" });
    await this.publishNowButton.click();
    await this.page
      .locator(".ant-message-success")
      .first()
      .waitFor({ state: "visible" });
  }

  async schedulePage(): Promise<void> {
    await this.scheduleButton.waitFor({ state: "visible" });
    await this.scheduleButton.click();
    await this.scheduleConfirmButton.waitFor({ state: "visible" });
    await this.scheduleConfirmButton.click();
    await this.scheduleConfirmButton.waitFor({ state: "hidden" });
    await this.page
      .locator(".ant-message-success")
      .first()
      .waitFor({ state: "visible" });
  }

  async unpublishPage(): Promise<void> {
    await this.unpublishButton.waitFor({ state: "visible" });
    await this.unpublishButton.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.page
      .locator(".ant-message-success")
      .first()
      .waitFor({ state: "visible" });
  }

  async deletePage(): Promise<void> {
    await this.deleteButton.waitFor({ state: "visible" });
    await this.deleteButton.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.page.waitForURL((url) => url.pathname === "/cms/pages");
  }

  async expectPageInTable(title: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: title });
    await row.first().waitFor({ state: "visible" });
  }

  async expectPageNotInTable(title: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: title });
    await row.waitFor({ state: "hidden" });
  }

  async openPage(title: string): Promise<void> {
    const link = this.page.getByRole("link", { name: title });
    await link.waitFor({ state: "visible" });
    await link.click();
    await this.titleInput.waitFor({ state: "visible" });
  }

  async switchTranslationTab(locale: string): Promise<void> {
    const tab = this.page
      .locator(".ant-tabs-tab")
      .filter({ hasText: new RegExp(locale, "i") });
    await tab.first().waitFor({ state: "visible", timeout: 5000 });
    await tab.first().click();
    const activeTab = this.page
      .locator(".ant-tabs-tab-active")
      .filter({ hasText: new RegExp(locale, "i") });
    await activeTab.first().waitFor({ state: "visible", timeout: 5000 });
    await this.page.waitForLoadState("networkidle");
  }

  async expectTitleValue(title: string): Promise<void> {
    await this.page.waitForFunction(
      "(expected) => document.querySelector('#title')?.value === expected",
      title,
      { timeout: 5000 },
    );
    const value = await this.titleInput.inputValue();
    assert.strictEqual(
      value,
      title,
      `Expected title to be "${title}", got "${value}"`,
    );
  }

  async deleteTranslation(): Promise<void> {
    await this.deleteTranslationButton.waitFor({ state: "visible" });
    await this.deleteTranslationButton.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.page
      .locator(".ant-message-success")
      .first()
      .waitFor({ state: "visible" });
  }

  async attemptDeleteTranslation(): Promise<void> {
    if (await this.deleteTranslationButton.isVisible()) {
      await this.deleteTranslationButton.click();
      await this.popconfirmOkButton.waitFor({ state: "visible" });
      await this.popconfirmOkButton.click();
    }
  }

  async expectDeleteTranslationHidden(): Promise<void> {
    await this.deleteTranslationButton.waitFor({ state: "hidden" });
  }

  async expectDeleteTranslationVisible(): Promise<void> {
    await this.deleteTranslationButton.waitFor({ state: "visible" });
  }

  async expectStatus(statusText: string): Promise<void> {
    const tag = this.page
      .locator(".ant-tag")
      .filter({ hasText: new RegExp(statusText, "i") });
    await tag.first().waitFor({ state: "visible" });
  }

  async expectNewButtonHidden(): Promise<void> {
    await this.page.waitForLoadState("networkidle");
    await this.newPageButton.waitFor({ state: "hidden" });
  }

  async expectNewButtonVisible(): Promise<void> {
    await this.newPageButton.waitFor({ state: "visible" });
  }

  async expectEditDisabled(): Promise<void> {
    assert.ok(
      await this.titleInput.isDisabled(),
      "Title input should be disabled",
    );
    assert.ok(
      await this.saveButton.isDisabled(),
      "Save button should be disabled",
    );
  }
}
