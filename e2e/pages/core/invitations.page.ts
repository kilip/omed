import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class InvitationsPage extends BasePage {
  readonly path = "/core/invitations";

  get newButton(): Locator {
    return this.page.getByRole("button", { name: "New invitation" });
  }

  get rolesSelect(): Locator {
    return this.page.getByRole("combobox", { name: "Roles" });
  }

  get emailInput(): Locator {
    return this.page.getByPlaceholder("user@example.com");
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: "Save" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  async openModal(): Promise<void> {
    await this.newButton.click();
    await this.rolesSelect.waitFor({ state: "visible" });
  }

  async selectRole(roleName: string): Promise<void> {
    await this.rolesSelect.click();
    const option = this.page
      .locator(".ant-select-item-option-content")
      .filter({ hasText: roleName })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
    // Click outside or press Escape to close dropdown if needed
    await this.page.keyboard.press("Escape");
  }

  async fillEmail(email: string): Promise<void> {
    await this.emailInput.fill(email);
  }

  async submit(): Promise<void> {
    await this.saveButton.click();
    await this.saveButton.waitFor({ state: "hidden" });
  }

  async expectRendered(): Promise<void> {
    await this.newButton.waitFor({ state: "visible" });
  }
}
