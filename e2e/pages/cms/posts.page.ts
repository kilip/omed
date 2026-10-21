import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class CmsPostsPage extends BasePage {
  readonly path = "/cms/posts";

  get newPostButton(): Locator {
    return this.page.getByRole("button", { name: /new post/i });
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
    return this.page.getByRole("button", { name: /^delete$/i });
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(".ant-popconfirm .ant-btn-primary");
  }

  get titleInput(): Locator {
    return this.page.locator("#title");
  }

  get slugInput(): Locator {
    return this.page.locator("#slug");
  }

  get excerptInput(): Locator {
    return this.page.locator("#excerpt");
  }

  get bodyInput(): Locator {
    return this.page.locator(".ant-input").last();
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get statusTag(): Locator {
    return this.page.locator(".ant-tag");
  }

  get selectCoverButton(): Locator {
    return this.page.getByRole("button", { name: /select cover/i });
  }

  get mediaPickerModal(): Locator {
    return this.page.locator(".ant-modal");
  }

  get mediaPickerFirstItem(): Locator {
    return this.page.locator(".ant-modal button[type='button']").first();
  }

  get mediaPickerSelectButton(): Locator {
    return this.page
      .locator(".ant-modal")
      .getByRole("button", { name: /select/i });
  }

  async goto(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async clickNewPost(): Promise<void> {
    await this.newPostButton.waitFor({ state: "visible" });
    await this.newPostButton.click();
    await this.titleInput.waitFor({ state: "visible" });
  }

  async fillPost(data: {
    title: string;
    slug?: string;
    excerpt?: string;
    body?: string;
  }): Promise<void> {
    await this.titleInput.fill(data.title);
    if (data.slug) {
      await this.slugInput.fill(data.slug);
    }
    if (data.excerpt) {
      await this.excerptInput.fill(data.excerpt);
    }
    if (data.body) {
      await this.bodyInput.fill(data.body);
    }
  }

  async selectCoverImage(): Promise<void> {
    await this.selectCoverButton.click();
    await this.mediaPickerModal.waitFor({ state: "visible" });
    await this.mediaPickerFirstItem.click();
    await this.mediaPickerSelectButton.click();
    await this.mediaPickerModal.waitFor({ state: "hidden" });
  }

  async savePost(): Promise<void> {
    await this.saveButton.click();
  }

  async publishPost(): Promise<void> {
    await this.publishNowButton.waitFor({ state: "visible" });
    await this.publishNowButton.click();
  }

  async schedulePost(): Promise<void> {
    await this.scheduleButton.waitFor({ state: "visible" });
    await this.scheduleButton.click();
    await this.scheduleConfirmButton.waitFor({ state: "visible" });
    await this.scheduleConfirmButton.click();
    await this.scheduleConfirmButton.waitFor({ state: "hidden" });
  }

  async unpublishPost(): Promise<void> {
    await this.unpublishButton.waitFor({ state: "visible" });
    await this.unpublishButton.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }

  async deletePost(): Promise<void> {
    await this.deleteButton.waitFor({ state: "visible" });
    await this.deleteButton.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.page.waitForURL((url) => !url.pathname.includes("/posts/"));
  }

  async expectPostInTable(title: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: title });
    await row.first().waitFor({ state: "visible" });
  }

  async expectPostNotInTable(title: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: title });
    await row.waitFor({ state: "hidden" });
  }

  async openPost(title: string): Promise<void> {
    const link = this.page.getByRole("link", { name: title });
    await link.waitFor({ state: "visible" });
    await link.click();
    await this.titleInput.waitFor({ state: "visible" });
  }

  async expectStatus(statusText: string): Promise<void> {
    const tag = this.page
      .locator(".ant-tag")
      .filter({ hasText: new RegExp(statusText, "i") });
    await tag.first().waitFor({ state: "visible" });
  }

  async expectNewButtonHidden(): Promise<void> {
    await this.newPostButton.waitFor({ state: "hidden" });
  }

  async expectNewButtonVisible(): Promise<void> {
    await this.newPostButton.waitFor({ state: "visible" });
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
