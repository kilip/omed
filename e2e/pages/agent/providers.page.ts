import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class AgentProvidersPage extends BasePage {
  readonly path = "/agent/providers";

  get addButton(): Locator {
    return this.page.getByRole("button", { name: /add provider/i });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get nameInput(): Locator {
    return this.page.locator("#name");
  }

  get kindSelect(): Locator {
    return this.page.locator(".ant-select:has(#kind)");
  }

  get modelInput(): Locator {
    return this.page.locator("#model");
  }

  get baseUrlInput(): Locator {
    return this.page.locator("#baseUrl");
  }

  get apiKeyInput(): Locator {
    return this.page.locator("#apiKey");
  }

  get isDefaultSwitch(): Locator {
    return this.page.locator("#isDefault");
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: /^save$/i });
  }

  get cancelButton(): Locator {
    return this.page.getByRole("button", { name: /^cancel$/i });
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(
      ".ant-popconfirm .ant-btn-primary, .ant-popover:not(.ant-popover-hidden) .ant-btn-primary",
    );
  }

  rowFor(name: string): Locator {
    return this.tableRows.filter({ hasText: name });
  }

  async goto(): Promise<void> {
    await this.page.goto(this.path);
    await this.page.waitForLoadState("networkidle");
  }

  async expectProvidersPageLoaded(): Promise<void> {
    await this.page.waitForSelector(".ant-table", { state: "visible" });
  }

  async clickAddProvider(): Promise<void> {
    await this.addButton.waitFor({ state: "visible" });
    await this.addButton.click();
    await this.nameInput.waitFor({ state: "visible" });
  }

  async fillName(name: string): Promise<void> {
    await this.nameInput.fill(name);
  }

  async selectKind(kind: string): Promise<void> {
    await this.kindSelect.click();
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: kind })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async fillModel(model: string): Promise<void> {
    await this.modelInput.fill(model);
  }

  async fillBaseUrl(url: string): Promise<void> {
    await this.baseUrlInput.fill(url);
  }

  async fillApiKey(key: string): Promise<void> {
    await this.apiKeyInput.fill(key);
  }

  async toggleDefault(): Promise<void> {
    await this.isDefaultSwitch.click();
  }

  async submitForm(): Promise<void> {
    await this.saveButton.click();
    await this.expectProvidersPageLoaded();
  }

  async expectProviderInTable(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible", timeout: 5000 });
    const text = await row.first().innerText();
    assert.ok(
      text.includes(name),
      `Expected provider "${name}" to be in table, got "${text}"`,
    );
  }

  async expectProviderNotInTable(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.waitFor({ state: "detached", timeout: 5000 }).catch(async () => {
      const count = await row.count();
      assert.strictEqual(
        count,
        0,
        `Expected provider "${name}" to be deleted from table`,
      );
    });
  }

  async clickTestConnection(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /test connection/i });
    await btn.click();
  }

  async expectTestSuccess(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const tag = row
      .first()
      .locator(".ant-tag", { hasText: /connection successful/i });
    await tag.waitFor({ state: "visible", timeout: 10000 });
    const text = await tag.innerText();
    assert.ok(
      /connection successful/i.test(text),
      `Expected connection successful tag, got "${text}"`,
    );
  }

  async clickSetDefault(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /set as default/i });
    await btn.click();
  }

  async expectProviderIsDefault(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const badge = row.first().locator(".ant-tag", { hasText: /^default$/i });
    await badge.waitFor({ state: "visible", timeout: 5000 });
  }

  async clickDelete(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /delete/i });
    await btn.click();
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }

  async expectAddButtonVisible(): Promise<void> {
    await this.addButton.waitFor({ state: "visible", timeout: 5000 });
  }

  async expectAddButtonHidden(): Promise<void> {
    const visible = await this.addButton
      .isVisible({ timeout: 1500 })
      .catch(() => false);
    assert.strictEqual(
      visible,
      false,
      "Expected add provider button to be hidden",
    );
  }

  async expectTestButtonHidden(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /test connection/i });
    const visible = await btn.isVisible({ timeout: 1500 }).catch(() => false);
    assert.strictEqual(
      visible,
      false,
      `Expected test connection button to be hidden for "${name}"`,
    );
  }

  async expectSetDefaultButtonHidden(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /set as default/i });
    const visible = await btn.isVisible({ timeout: 1500 }).catch(() => false);
    assert.strictEqual(
      visible,
      false,
      `Expected set default button to be hidden for "${name}"`,
    );
  }

  async expectEditButtonHidden(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /edit/i });
    const visible = await btn.isVisible({ timeout: 1500 }).catch(() => false);
    assert.strictEqual(
      visible,
      false,
      `Expected edit button to be hidden for "${name}"`,
    );
  }

  async expectDeleteButtonHidden(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.first().waitFor({ state: "visible" });
    const btn = row.first().getByRole("button", { name: /delete/i });
    const visible = await btn.isVisible({ timeout: 1500 }).catch(() => false);
    assert.strictEqual(
      visible,
      false,
      `Expected delete button to be hidden for "${name}"`,
    );
  }
}
