import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class CmsMediaPage extends BasePage {
  readonly path = "/cms/media";

  get uploadButton(): Locator {
    return this.page.getByRole("button", { name: /upload/i });
  }

  get fileInput(): Locator {
    return this.page.locator('input[type="file"]');
  }

  get cards(): Locator {
    return this.page.locator(".ant-card");
  }

  get drawer(): Locator {
    return this.page.locator(".ant-drawer");
  }

  get altTextInput(): Locator {
    return this.page.locator("#altText");
  }

  get drawerSaveButton(): Locator {
    return this.page.locator(".ant-drawer .ant-btn-primary");
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(".ant-popconfirm .ant-btn-primary");
  }

  get errorMessage(): Locator {
    return this.page.locator(".ant-message-error");
  }

  async goto(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async uploadMedia(
    filename: string,
    content = "sample content",
  ): Promise<void> {
    await this.fileInput.setInputFiles({
      name: filename,
      mimeType: filename.endsWith(".pdf") ? "application/pdf" : "image/png",
      buffer: Buffer.from(content, "utf-8"),
    });
    // Wait for card to appear
    await this.cards.first().waitFor({ state: "visible" });
  }

  async openEditDrawer(titleOrName: string): Promise<void> {
    const card = this.cards.filter({ hasText: titleOrName }).first();
    await card.waitFor({ state: "visible" });
    const editBtn = card.locator(".anticon-edit").locator("..");
    await editBtn.click();
    await this.drawer.waitFor({ state: "visible" });
  }

  async editAltText(altText: string): Promise<void> {
    await this.altTextInput.fill(altText);
    await this.drawerSaveButton.click();
    await this.drawer.waitFor({ state: "hidden" });
  }

  async deleteMedia(titleOrName: string): Promise<void> {
    const card = this.cards.filter({ hasText: titleOrName }).first();
    await card.waitFor({ state: "visible" });
    const deleteBtn = card.locator(".anticon-delete").locator("..");
    await deleteBtn.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }

  async expectMediaVisible(titleOrName: string): Promise<void> {
    const card = this.cards.filter({ hasText: titleOrName }).first();
    await card.waitFor({ state: "visible" });
  }

  async expectMediaNotVisible(titleOrName: string): Promise<void> {
    const card = this.cards.filter({ hasText: titleOrName });
    await card.waitFor({ state: "hidden" });
  }

  async expectDeleteBlocked(): Promise<void> {
    await this.errorMessage.waitFor({ state: "visible" });
    const text = await this.errorMessage.innerText();
    assert.ok(
      text.toLowerCase().includes("cannot delete media") ||
        text.toLowerCase().includes("used") ||
        text.toLowerCase().includes("cover"),
      `Expected delete blocked message, got: ${text}`,
    );
  }

  async expectUploadButtonHidden(): Promise<void> {
    await this.uploadButton.waitFor({ state: "hidden" });
  }

  async expectUploadButtonVisible(): Promise<void> {
    await this.uploadButton.waitFor({ state: "visible" });
  }
}
