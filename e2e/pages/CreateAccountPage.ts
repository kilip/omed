import type { Page } from "playwright";
import { BasePage } from "./BasePage";

export class CreateAccountPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async open() {
    await this.goto("/fin/accounts/create");
  }

  get codeInput() {
    return this.page.locator("input#code");
  }

  get nameInput() {
    return this.page.locator("input#name");
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

  get parentIdSelect() {
    return this.page.locator("#parentId");
  }

  get submitButton() {
    return this.page.locator("button[type='submit']");
  }

  get cancelButton() {
    return this.page.getByRole("button", { name: /Cancel|Batal/i });
  }

  async fillCode(code: string) {
    await this.codeInput.waitFor({ state: "visible" });
    await this.codeInput.fill(code);
  }

  async fillName(name: string) {
    await this.nameInput.waitFor({ state: "visible" });
    await this.nameInput.fill(name);
  }

  async selectType(typeName: string) {
    await this.typeSelect.waitFor({ state: "visible" });
    await this.typeSelect.click();
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: new RegExp(typeName, "i") })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async fillForm(data: { code: string; name: string; type?: string }) {
    await this.fillCode(data.code);
    await this.fillName(data.name);
    if (data.type) {
      await this.selectType(data.type);
    }
  }

  async clickSubmit() {
    await this.submitButton.waitFor({ state: "visible" });
    await this.submitButton.click();
  }
}
