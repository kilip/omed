import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class MembersPage extends BasePage {
  readonly path = "/core/members";

  get inviteButton(): Locator {
    return this.page.getByRole("button", { name: "Invite member" });
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

  roleRadio(roleId: string): Locator {
    return this.page
      .locator(".ant-radio-wrapper")
      .filter({ hasText: roleId })
      .locator("input[type='radio']");
  }

  async selectRole(roleId: string): Promise<void> {
    const radio = this.page
      .locator(".ant-radio-wrapper")
      .filter({ hasText: roleId });
    await radio.waitFor({ state: "visible" });
    await radio.click();
  }

  async fillEmail(email: string): Promise<void> {
    await this.emailInput.fill(email);
  }

  async submit(): Promise<void> {
    await this.saveButton.click();
  }

  async expectRoleChecked(roleId: string): Promise<void> {
    const radio = this.roleRadio(roleId);
    await radio.waitFor({ state: "attached" });
    assert.equal(
      await radio.isChecked(),
      true,
      `Expected role ${roleId} to be checked`,
    );
  }

  async expectRoleUnchecked(roleId: string): Promise<void> {
    const radio = this.roleRadio(roleId);
    await radio.waitFor({ state: "attached" });
    assert.equal(
      await radio.isChecked(),
      false,
      `Expected role ${roleId} to be unchecked`,
    );
  }

  async expectMemberWithRole(email: string, roleName: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: email });
    await row.waitFor({ state: "visible" });
    const text = await row.innerText();
    assert.ok(
      text.includes(roleName),
      `Expected member ${email} to have role containing "${roleName}", got "${text}"`,
    );
  }

  get personalNotice(): Locator {
    return this.page.getByText(
      "Personal workspaces are for single-user use and cannot have members.",
    );
  }

  async expectPersonalNotice(): Promise<void> {
    await this.personalNotice.waitFor({ state: "visible" });
  }

  async expectMenuHidden(name: string): Promise<void> {
    const menuItem = this.page
      .locator(".ant-menu-item")
      .filter({ hasText: name });
    assert.equal(
      await menuItem.count(),
      0,
      `Expected menu "${name}" to be hidden, but found visible elements`,
    );
  }
}
