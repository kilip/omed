import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class AccessTokensPage extends BasePage {
  readonly path = "/core/access-tokens";

  get createButton(): Locator {
    return this.page.getByRole("button", { name: "Generate token" });
  }

  get modal(): Locator {
    return this.page.locator(".ant-modal");
  }

  get nameInput(): Locator {
    return this.modal.locator("#name");
  }

  get expiresSelect(): Locator {
    return this.modal.locator(".ant-select:has(#expiresInDays)");
  }

  get generateButton(): Locator {
    return this.modal.getByRole("button", { name: "Generate" });
  }

  get tokenInput(): Locator {
    return this.modal.locator("input[readonly]");
  }

  get doneButton(): Locator {
    return this.modal.getByRole("button", { name: "Done" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover:not(.ant-popover-hidden) .ant-btn-primary",
    );
  }

  rowFor(name: string): Locator {
    return this.tableRows.filter({ hasText: name });
  }

  async openCreateModal(): Promise<void> {
    await this.createButton.waitFor({ state: "visible" });
    await this.createButton.click();
    await this.nameInput.waitFor({ state: "visible" });
  }

  async fillForm(name: string, expiry?: string): Promise<void> {
    await this.nameInput.fill(name);
    if (expiry) {
      await this.expiresSelect.click();
      const option = this.page
        .locator(
          ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
        )
        .filter({ hasText: expiry })
        .first();
      await option.waitFor({ state: "visible" });
      await option.click();
    }
  }

  async submit(): Promise<void> {
    await this.generateButton.click();
  }

  async createToken(name: string, expiry?: string): Promise<void> {
    await this.openCreateModal();
    await this.fillForm(name, expiry);
    await this.submit();
  }

  async expectTokenRevealed(prefix = "omed_pat_"): Promise<string> {
    await this.tokenInput.waitFor({ state: "visible" });
    const token = await this.tokenInput.inputValue();
    assert.ok(
      token.startsWith(prefix),
      `Expected token to start with "${prefix}", got "${token}"`,
    );
    return token;
  }

  async closeModal(): Promise<void> {
    await this.doneButton.waitFor({ state: "visible" });
    await this.doneButton.click();
    await this.modal.waitFor({ state: "hidden" });
  }

  async expectTokenRevealClosed(): Promise<void> {
    await this.tokenInput.waitFor({ state: "hidden" });
    assert.equal(
      await this.tokenInput.isVisible(),
      false,
      "Expected token reveal input to be hidden",
    );
  }

  async expectTokenInTable(name: string, status: string): Promise<void> {
    const row = this.rowFor(name).first();
    await row.waitFor({ state: "visible" });
    const tag = row.locator(".ant-tag");
    await tag.waitFor({ state: "visible" });
    const tagText = await tag.innerText();
    assert.equal(
      tagText.trim().toLowerCase(),
      status.toLowerCase(),
      `Expected token status "${status}", got "${tagText}"`,
    );
  }

  async revokeToken(name: string): Promise<void> {
    const row = this.rowFor(name).first();
    await row.waitFor({ state: "visible" });
    const revokeBtn = row.getByRole("button", { name: "Revoke" });
    await revokeBtn.waitFor({ state: "visible" });
    await revokeBtn.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.popconfirmOkButton.waitFor({ state: "hidden" });
  }
}
