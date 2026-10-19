import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinTransactionDetailPage extends BasePage {
  readonly path = "/fin/transactions";

  get editButton(): Locator {
    return this.page.getByRole("button", { name: "Edit transaction" });
  }

  get saveChangesButton(): Locator {
    return this.page.getByRole("button", { name: "Save changes" });
  }

  get deleteButton(): Locator {
    return this.page.getByRole("button", { name: "Delete transaction" });
  }

  get addEntryButton(): Locator {
    return this.page.getByRole("button", { name: "Add entry" });
  }

  get popconfirmOkButton(): Locator {
    return this.page
      .locator(".ant-popconfirm")
      .getByRole("button", { name: "OK" });
  }

  get editReferenceInput(): Locator {
    return this.page.locator(".ant-form").getByPlaceholder("Reference");
  }

  get editDescriptionInput(): Locator {
    return this.page.locator(".ant-form").getByPlaceholder("Description");
  }

  get entryRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  async clickEdit(): Promise<void> {
    await this.editButton.waitFor({ state: "visible" });
    await this.editButton.click();
  }

  async fillReference(ref: string): Promise<void> {
    await this.editReferenceInput.fill(ref);
  }

  async fillDescription(desc: string): Promise<void> {
    await this.editDescriptionInput.fill(desc);
  }

  async saveChanges(): Promise<void> {
    await this.saveChangesButton.click();
  }

  async clickDelete(): Promise<void> {
    await this.deleteButton.waitFor({ state: "visible" });
    await this.deleteButton.click();
  }

  async confirmDelete(): Promise<void> {
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }

  async clickAddEntry(): Promise<void> {
    await this.addEntryButton.waitFor({ state: "visible" });
    await this.addEntryButton.click();
  }

  async expectEntryInTable(text: string): Promise<void> {
    const row = this.entryRows.filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
    const rowText = await row.first().innerText();
    assert.ok(
      rowText.includes(text),
      `Expected entry row to contain "${text}", got "${rowText}"`,
    );
  }

  async clickEntryRow(text: string): Promise<void> {
    const row = this.entryRows.filter({ hasText: text });
    await row.first().click();
  }

  async expectText(text: string): Promise<void> {
    const el = this.page.getByText(text).first();
    await el.waitFor({ state: "visible" });
  }

  async expectNoEditButton(): Promise<void> {
    const count = await this.editButton.count();
    assert.equal(count, 0, "Expected edit transaction button to be hidden");
  }

  async expectNoDeleteButton(): Promise<void> {
    const count = await this.deleteButton.count();
    assert.equal(count, 0, "Expected delete transaction button to be hidden");
  }

  async expectNoAddEntryButton(): Promise<void> {
    const count = await this.addEntryButton.count();
    assert.equal(count, 0, "Expected add entry button to be hidden");
  }

  async expectAddEntryButtonVisible(): Promise<void> {
    await this.addEntryButton.waitFor({ state: "visible" });
  }

  async expectEditButtonVisible(): Promise<void> {
    await this.editButton.waitFor({ state: "visible" });
  }

  async expectDeleteButtonVisible(): Promise<void> {
    await this.deleteButton.waitFor({ state: "visible" });
  }
}
