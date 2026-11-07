import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinTalkToAgentPage extends BasePage {
  readonly path = "/fin/agent";

  // Sidebar / Conversations locators
  get conversationSidebar(): Locator {
    return this.page.getByTestId("conversation-sidebar");
  }

  get newConversationButton(): Locator {
    return this.page.getByRole("button", { name: /New conversation/i });
  }

  get conversationItems(): Locator {
    return this.page.locator('[data-testid^="conversation-item-"]');
  }

  get deleteConversationButton(): Locator {
    return this.page
      .getByRole("button", { name: /Delete conversation/i })
      .first();
  }

  // Message list locators
  get messageList(): Locator {
    return this.page.getByTestId("message-list");
  }

  get emptyMessageList(): Locator {
    return this.page.getByTestId("message-list-empty");
  }

  get messages(): Locator {
    return this.page.locator('[data-testid^="message-"]');
  }

  get attachmentThumbnails(): Locator {
    return this.page.locator('[data-testid^="attachment-"]');
  }

  // Composer locators
  get composerInput(): Locator {
    return this.page.locator("textarea");
  }

  get fileInput(): Locator {
    return this.page.locator('input[type="file"]').first();
  }

  get sendButton(): Locator {
    return this.page.getByRole("button", { name: /Send/i });
  }

  get attachButton(): Locator {
    return this.page.getByRole("button", { name: /Attach file/i });
  }

  // Draft Preview Card locators
  get draftPreviewCard(): Locator {
    return this.page.getByTestId("draft-preview-card").first();
  }

  get confirmButton(): Locator {
    return this.page.getByRole("button", { name: "Confirm" }).first();
  }

  get discardButton(): Locator {
    return this.page.getByRole("button", { name: "Discard" }).first();
  }

  get inlineEditButton(): Locator {
    return this.page.getByRole("button", { name: "Edit" }).first();
  }

  get transactionLink(): Locator {
    return this.page.getByRole("link", { name: /View Transaction/i }).first();
  }

  // Status and Permission locators
  get forbiddenResult(): Locator {
    return this.page.locator(".ant-result-403");
  }

  // Actions
  async navigate(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async clickNewConversation(): Promise<void> {
    await this.newConversationButton.waitFor({ state: "visible" });
    await this.newConversationButton.click();
  }

  async sendText(content: string): Promise<void> {
    await this.composerInput.waitFor({ state: "visible" });
    await this.composerInput.fill(content);
    await this.composerInput.press("Enter");
  }

  async attachFile(fileName: string, content?: string): Promise<void> {
    const isPdf = fileName.endsWith(".pdf");
    const mimeType = isPdf ? "application/pdf" : "image/png";
    const sampleContent =
      content ??
      (isPdf
        ? `%PDF-1.4\n% ${fileName}\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF`
        : `\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB\`\x82/*${fileName}*/`);

    await this.fileInput.setInputFiles({
      name: fileName,
      mimeType,
      buffer: Buffer.from(sampleContent, isPdf ? "utf-8" : "binary"),
    });
    await this.sendButton.waitFor({ state: "visible" });
    await this.sendButton.click();
  }

  async confirmDraft(): Promise<void> {
    await this.confirmButton.waitFor({ state: "visible" });
    await this.confirmButton.click();
  }

  async discardDraft(): Promise<void> {
    await this.discardButton.waitFor({ state: "visible" });
    await this.discardButton.click();
    // Ant Design Popconfirm: confirm action
    const popconfirmOk = this.page.locator(".ant-popconfirm .ant-btn-primary");
    await popconfirmOk.waitFor({ state: "visible" });
    await popconfirmOk.click();
  }

  // Expectations
  async expectMessage(text: string, role?: string): Promise<void> {
    const msgLocator = this.messageList
      .locator('[data-testid^="message-"]')
      .filter({ hasText: text });
    await msgLocator.first().waitFor({ state: "visible", timeout: 15_000 });
    const count = await msgLocator.count();
    assert.ok(count > 0, `Expected message with text "${text}" to be visible`);

    if (role) {
      // Role can be user or assistant
      const textContent = await msgLocator.first().textContent();
      assert.ok(textContent, `Expected message content for role ${role}`);
    }
  }

  async expectDraftCard(kind: string): Promise<void> {
    await this.draftPreviewCard.waitFor({ state: "visible", timeout: 30_000 });
    const cardText = await this.draftPreviewCard.textContent();
    assert.ok(cardText, "Expected draft card to have content");

    const normalizedKind = kind.toLowerCase();
    if (normalizedKind.includes("journal")) {
      assert.ok(
        cardText.includes("Journal Draft") ||
          cardText.includes("Draft Jurnal") ||
          cardText.includes("Kas Utama") ||
          cardText.includes("Debit"),
        `Expected draft card to be journal draft, got: ${cardText}`,
      );
    } else if (normalizedKind.includes("expense")) {
      assert.ok(
        cardText.includes("Expense Draft") ||
          cardText.includes("Draft Pengeluaran"),
        `Expected draft card to be expense draft, got: ${cardText}`,
      );
    } else if (normalizedKind.includes("income")) {
      assert.ok(
        cardText.includes("Income Draft") ||
          cardText.includes("Draft Pendapatan"),
        `Expected draft card to be income draft, got: ${cardText}`,
      );
    }
  }

  async expectAttachmentThumbnail(fileName: string): Promise<void> {
    const thumb = this.attachmentThumbnails.filter({ hasText: fileName });
    await thumb.first().waitFor({ state: "visible", timeout: 10_000 });
    const count = await thumb.count();
    assert.ok(
      count > 0,
      `Expected attachment thumbnail for "${fileName}" to be visible`,
    );
  }

  async expectConfirmedState(): Promise<void> {
    const confirmedTag = this.draftPreviewCard
      .locator(".ant-tag")
      .filter({ hasText: /Confirmed|Dikonfirmasi/i });
    await confirmedTag.waitFor({ state: "visible", timeout: 15_000 });
    await this.transactionLink.waitFor({ state: "visible", timeout: 15_000 });
  }

  async expectDiscardedState(): Promise<void> {
    const discardedTag = this.draftPreviewCard
      .locator(".ant-tag")
      .filter({ hasText: /Discarded|Dibuang/i });
    await discardedTag.waitFor({ state: "visible", timeout: 15_000 });
    const confirmVisible = await this.confirmButton.isVisible();
    assert.strictEqual(
      confirmVisible,
      false,
      "Confirm button should not be visible when discarded",
    );
  }

  async expectForbidden(): Promise<void> {
    await this.forbiddenResult.waitFor({ state: "visible", timeout: 10_000 });
  }
}
