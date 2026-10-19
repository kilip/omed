import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class WorkspacesPage extends BasePage {
  readonly path = "/core";

  get newWorkspaceButton(): Locator {
    return this.page.getByRole("button", { name: "New workspace" });
  }

  get nameInput(): Locator {
    return this.page.getByPlaceholder("e.g. Acme Corp");
  }

  get descriptionInput(): Locator {
    return this.page.getByPlaceholder("Optional workspace description");
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: "Save" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get activeWorkspaceSelector(): Locator {
    return this.page.locator(
      ".ant-select:has([aria-label='Workspace']), [aria-label='Workspace']",
    );
  }

  async clickNewWorkspace(): Promise<void> {
    await this.newWorkspaceButton.waitFor({ state: "visible" });
    await this.newWorkspaceButton.click();
  }

  async fillName(name: string): Promise<void> {
    await this.nameInput.waitFor({ state: "visible" });
    await this.nameInput.fill(name);
  }

  async fillDescription(description: string): Promise<void> {
    await this.descriptionInput.waitFor({ state: "visible" });
    await this.descriptionInput.fill(description);
  }

  async submit(): Promise<void> {
    await this.saveButton.waitFor({ state: "visible" });
    await this.saveButton.click();
  }

  async expectWorkspaceInTable(name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: name });
    await row.waitFor({ state: "visible" });
    const text = await row.innerText();
    assert.ok(
      text.includes(name),
      `Expected workspace "${name}" to be in table, got "${text}"`,
    );
  }

  async expectActiveWorkspace(name: string): Promise<void> {
    const selector = this.activeWorkspaceSelector.first();
    await selector.waitFor({ state: "visible" });
    const text = await selector.innerText();
    assert.ok(
      text.includes(name),
      `Expected active workspace to be "${name}", got "${text}"`,
    );
  }
}
