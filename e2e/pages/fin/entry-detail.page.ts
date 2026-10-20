import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinEntryDetailPage extends BasePage {
  readonly path = "/fin/journal";

  get editButton(): Locator {
    return this.page.getByRole("button", { name: "Edit entry" });
  }

  get saveChangesButton(): Locator {
    return this.page.getByRole("button", { name: "Save changes" });
  }

  get postButton(): Locator {
    return this.page.getByRole("button", { name: "Post" });
  }

  get deleteButton(): Locator {
    return this.page.getByRole("button", { name: "Delete" });
  }

  get voidButton(): Locator {
    return this.page.getByRole("button", { name: "Void" });
  }

  get reverseButton(): Locator {
    return this.page.getByRole("button", { name: "Reverse" });
  }

  get reverseModal(): Locator {
    return this.page.locator(".ant-modal");
  }

  get reverseModalOkButton(): Locator {
    return this.reverseModal.getByRole("button", { name: "OK" });
  }

  get popconfirmOkButton(): Locator {
    return this.page
      .locator(
        ".ant-popconfirm .ant-btn-primary, .ant-popover .ant-btn-primary",
      )
      .first();
  }

  get statusTag(): Locator {
    return this.page.locator(".ant-tag");
  }

  async expectStatus(status: string): Promise<void> {
    const tag = this.statusTag.filter({ hasText: new RegExp(status, "i") });
    await tag.first().waitFor({ state: "visible" });
    const text = await tag.first().innerText();
    assert.ok(
      text.toLowerCase().includes(status.toLowerCase()),
      `Expected status tag to contain "${status}", got "${text}"`,
    );
  }

  async clickPost(): Promise<void> {
    await this.postButton.waitFor({ state: "visible" });
    await this.postButton.click();
  }

  async clickEdit(): Promise<void> {
    await this.editButton.waitFor({ state: "visible" });
    await this.editButton.click();
  }

  async fillDescription(desc: string): Promise<void> {
    const input = this.page.locator("form #description");
    await input.fill(desc);
  }

  async clickSaveChanges(): Promise<void> {
    await this.saveChangesButton.click();
  }

  async clickDelete(): Promise<void> {
    await this.page
      .locator(".ant-btn-loading")
      .waitFor({ state: "detached" })
      .catch(() => {});
    await this.deleteButton.waitFor({ state: "visible" });
    await this.deleteButton.click();
    const okBtn = this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover .ant-btn-primary, .popconfirm-ok-btn",
    );
    try {
      await okBtn.waitFor({ state: "visible", timeout: 2000 });
    } catch {
      await this.deleteButton.click();
      await okBtn.waitFor({ state: "visible", timeout: 5000 });
    }
    await okBtn.click();
  }

  async clickVoid(): Promise<void> {
    await this.page
      .locator(".ant-btn-loading")
      .waitFor({ state: "detached" })
      .catch(() => {});
    await this.voidButton.waitFor({ state: "visible" });
    await this.voidButton.click();
    const okBtn = this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover .ant-btn-primary, .popconfirm-ok-btn",
    );
    try {
      await okBtn.waitFor({ state: "visible", timeout: 2000 });
    } catch {
      await this.voidButton.click();
      await okBtn.waitFor({ state: "visible", timeout: 5000 });
    }
    await okBtn.click();
  }

  async clickReverse(): Promise<void> {
    await this.reverseButton.waitFor({ state: "visible" });
    await this.reverseButton.click();
  }

  async submitReverseModal(): Promise<void> {
    await this.reverseModal.waitFor({ state: "visible" });
    await this.reverseModalOkButton.click();
  }

  async clickPostingAccount(accountCode: string): Promise<void> {
    const link = this.page.getByRole("link", {
      name: new RegExp(`^${accountCode}\\s*-`),
    });
    await link.first().waitFor({ state: "visible" });
    await link.first().click();
  }

  async expectEditButtonVisible(): Promise<void> {
    await this.editButton.waitFor({ state: "visible" });
  }

  async expectNoEditButton(): Promise<void> {
    const count = await this.editButton.count();
    assert.equal(count, 0, "Expected edit entry button to be hidden");
  }

  async expectPostButtonVisible(): Promise<void> {
    await this.postButton.waitFor({ state: "visible" });
  }

  async expectNoPostButton(): Promise<void> {
    const count = await this.postButton.count();
    assert.equal(count, 0, "Expected post entry button to be hidden");
  }

  async expectDeleteButtonVisible(): Promise<void> {
    await this.deleteButton.waitFor({ state: "visible" });
  }

  async expectNoDeleteButton(): Promise<void> {
    const count = await this.deleteButton.count();
    assert.equal(count, 0, "Expected delete entry button to be hidden");
  }

  async expectVoidButtonVisible(): Promise<void> {
    await this.voidButton.waitFor({ state: "visible" });
  }

  async expectNoVoidButton(): Promise<void> {
    const count = await this.voidButton.count();
    assert.equal(count, 0, "Expected void entry button to be hidden");
  }

  async expectReverseButtonVisible(): Promise<void> {
    await this.reverseButton.waitFor({ state: "visible" });
  }

  async expectNoReverseButton(): Promise<void> {
    const count = await this.reverseButton.count();
    assert.equal(count, 0, "Expected reverse entry button to be hidden");
  }
}
