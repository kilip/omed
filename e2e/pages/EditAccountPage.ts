import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class EditAccountPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  get nameInput() {
    return this.page.locator("input#name");
  }

  get codeInput() {
    return this.page.locator("input#code");
  }

  get typeSelect() {
    return this.page.locator("#type");
  }

  get currencySelect() {
    return this.page.locator("#currency");
  }

  get descriptionInput() {
    return this.page.locator("textarea#description");
  }

  get statusSelect() {
    return this.page.locator("#status");
  }

  get parentIdSelect() {
    return this.page.locator("#parentId");
  }

  get submitButton() {
    return this.page.locator("button[type='submit']");
  }

  get deleteButton() {
    return this.page.getByRole("button", {
      name: /Delete Account|Hapus Akun/i,
    });
  }

  get popconfirmOkButton() {
    return this.page
      .locator(".ant-popconfirm-buttons")
      .getByRole("button", { name: /Yes, Delete|Ya, Hapus|OK/i });
  }

  async fillName(name: string) {
    await this.nameInput.waitFor({ state: "visible" });
    await this.nameInput.fill(name);
  }

  async fillDescription(description: string) {
    await this.descriptionInput.waitFor({ state: "visible" });
    await this.descriptionInput.fill(description);
  }

  async selectStatus(status: string) {
    await this.statusSelect.waitFor({ state: "visible" });
    await this.statusSelect.click();
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: new RegExp(status, "i") })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async clickSubmit() {
    await this.submitButton.waitFor({ state: "visible" });
    await this.submitButton.click();
  }

  async clickDelete() {
    await this.deleteButton.waitFor({ state: "visible" });
    await this.deleteButton.click();
  }

  async confirmDelete() {
    await this.popconfirmOkButton.waitFor({ state: "visible" });
    await this.popconfirmOkButton.click();
  }
}
