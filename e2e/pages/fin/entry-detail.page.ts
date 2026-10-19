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
      .locator(".ant-popconfirm")
      .getByRole("button", { name: "OK" });
  }

  get statusTag(): Locator {
    return this.page.locator(".ant-tag");
  }

  async expectStatus(status: string): Promise<void> {
    await this.statusTag.first().waitFor({ state: "visible" });
    const text = await this.statusTag.first().innerText();
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
    const input = this.page.getByPlaceholder("Description");
    await input.fill(desc);
  }

  async clickSaveChanges(): Promise<void> {
    await this.saveChangesButton.click();
  }

  async clickDelete(): Promise<void> {
    await this.deleteButton.click();
    await this.popconfirmOkButton.click();
  }

  async clickVoid(): Promise<void> {
    await this.voidButton.click();
    await this.popconfirmOkButton.click();
  }

  async clickReverse(): Promise<void> {
    await this.reverseButton.waitFor({ state: "visible" });
    await this.reverseButton.click();
  }

  async submitReverseModal(): Promise<void> {
    await this.reverseModal.waitFor({ state: "visible" });
    await this.reverseModalOkButton.click();
  }
}
