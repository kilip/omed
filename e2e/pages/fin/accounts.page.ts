import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinAccountsPage extends BasePage {
  readonly path = "/fin/accounts";

  get setupTemplateButton(): Locator {
    return this.page.getByRole("button", { name: "Setup template" });
  }

  get emptyState(): Locator {
    return this.page.locator(".ant-empty");
  }

  get emptyStateSeedButton(): Locator {
    return this.emptyState.getByRole("button", { name: "Setup template" });
  }

  get newAccountButton(): Locator {
    return this.page.getByRole("button", { name: "New account" });
  }

  get drawer(): Locator {
    return this.page.locator(".ant-drawer");
  }

  get codeInput(): Locator {
    return this.drawer.locator("#code");
  }

  get nameInput(): Locator {
    return this.drawer.locator("#name");
  }

  get typeSelect(): Locator {
    return this.drawer.locator("#type");
  }

  get activeSwitch(): Locator {
    return this.drawer.locator("#isActive");
  }

  get saveButton(): Locator {
    return this.drawer
      .locator(".ant-drawer-extra")
      .getByRole("button", { name: "Save" });
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover:not(.ant-popover-hidden) .ant-btn-primary",
    );
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  async expectEmptyStateVisible(): Promise<void> {
    await this.emptyState.waitFor({ state: "visible" });
  }

  async clickEmptyStateSeedButton(): Promise<void> {
    await this.emptyStateSeedButton.waitFor({ state: "visible" });
    await this.emptyStateSeedButton.click();
  }

  async expectSetupTemplateButtonHidden(): Promise<void> {
    const isVisible = await this.setupTemplateButton
      .isVisible({ timeout: 1500 })
      .catch(() => false);
    assert.strictEqual(
      isVisible,
      false,
      "Expected setup template button to be hidden",
    );
  }

  async expectAccountInTable(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code });
    await row.first().waitFor({ state: "visible" });
    const text = await row.first().innerText();
    assert.ok(
      text.includes(code),
      `Expected account with code "${code}" to be in table, got "${text}"`,
    );
  }

  async clickAccountLedger(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code });
    await row.first().waitFor({ state: "visible" });
    const ledgerLink = row.first().getByRole("link", { name: code });
    await ledgerLink.click();
  }

  async clickNewAccount(): Promise<void> {
    await this.newAccountButton.waitFor({ state: "visible" });
    await this.newAccountButton.click();
    await this.expectDrawerVisible();
  }

  async expectDrawerVisible(): Promise<void> {
    await this.drawer.waitFor({ state: "visible" });
  }

  async expectDrawerHidden(): Promise<void> {
    await this.drawer.waitFor({ state: "hidden" });
  }

  async selectType(type: string): Promise<void> {
    await this.typeSelect.click();
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: type })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async fillAccountForm(
    code: string,
    name: string,
    type: string,
  ): Promise<void> {
    await this.codeInput.fill(code);
    await this.nameInput.fill(name);
    await this.selectType(type);
  }

  async updateAccountName(name: string): Promise<void> {
    await this.nameInput.fill(name);
  }

  async toggleActiveSwitch(): Promise<void> {
    await this.activeSwitch.click();
  }

  async saveAccount(): Promise<void> {
    await this.saveButton.click();
    await this.expectDrawerHidden();
  }

  async clickEditAccount(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code }).first();
    await row.waitFor({ state: "visible" });
    const editBtn = row.getByRole("button", { name: "Edit" });
    await editBtn.click();
    await this.expectDrawerVisible();
  }

  async deleteAccount(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code }).first();
    await row.waitFor({ state: "visible" });
    const deleteBtn = row.getByRole("button", { name: "Delete" });
    await deleteBtn.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
    await this.popconfirmOkButton.waitFor({ state: "hidden" });
  }

  async expectAccountStatus(code: string, status: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code }).first();
    await row.waitFor({ state: "visible" });
    const text = await row.innerText();
    assert.ok(
      text.includes(status),
      `Expected account with code "${code}" to have status "${status}", got "${text}"`,
    );
  }

  async expectAccountName(code: string, name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code }).first();
    await row.waitFor({ state: "visible" });
    const text = await row.innerText();
    assert.ok(
      text.includes(name),
      `Expected account with code "${code}" to have name "${name}", got "${text}"`,
    );
  }

  async expectAccountNotInTable(code: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: code });
    const isVisible = await row
      .first()
      .isVisible({ timeout: 2000 })
      .catch(() => false);
    assert.strictEqual(
      isVisible,
      false,
      `Expected account with code "${code}" to not be in table`,
    );
  }
}
