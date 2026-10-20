import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinPeriodsPage extends BasePage {
  readonly path = "/fin/periods";

  get newPeriodButton(): Locator {
    return this.page.getByRole("button", { name: "New period" });
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get modal(): Locator {
    return this.page.locator(".ant-modal");
  }

  get modalSaveButton(): Locator {
    return this.modal.getByRole("button", { name: "Save" });
  }

  get modalNameInput(): Locator {
    return this.modal.locator("#name");
  }

  get modalFiscalYearInput(): Locator {
    return this.modal.locator("#fiscalYear");
  }

  get modalDescriptionInput(): Locator {
    return this.modal.locator("#description");
  }

  async clickNewPeriod(): Promise<void> {
    await this.newPeriodButton.waitFor({ state: "visible" });
    await this.newPeriodButton.click();
    await this.modal.waitFor({ state: "visible" });
  }

  async fillPeriodForm(values: {
    name: string;
    fiscalYear?: number;
    description?: string;
  }): Promise<void> {
    if (values.fiscalYear !== undefined) {
      await this.modalFiscalYearInput.fill(String(values.fiscalYear));
      await this.modalFiscalYearInput.press("Enter");
      const rangeInputs = this.modal.locator(".ant-picker-input input");
      if ((await rangeInputs.count()) >= 2) {
        await rangeInputs.first().fill(`${values.fiscalYear}-01-01`);
        await rangeInputs.first().press("Enter");
        await rangeInputs.last().fill(`${values.fiscalYear}-12-31`);
        await rangeInputs.last().press("Enter");
      }
    }
    await this.modalNameInput.fill(values.name);
    if (values.description !== undefined) {
      await this.modalDescriptionInput.fill(values.description);
    }
  }

  async clickSave(): Promise<void> {
    await this.modalSaveButton.waitFor({ state: "visible" });
    await this.modalSaveButton.click();
    await this.modal.waitFor({ state: "hidden" });
  }

  async expectPeriodInTable(name: string, status?: string): Promise<void> {
    let row = this.tableRows.filter({ hasText: name });
    if (status) {
      row = row.filter({ hasText: status });
    }
    await row.first().waitFor({ state: "visible" });
    const text = await row.first().innerText();
    assert.ok(
      text.includes(name),
      `Expected period with name "${name}" to be in table, got "${text}"`,
    );
    if (status) {
      assert.ok(
        text.includes(status),
        `Expected period to have status "${status}", got "${text}"`,
      );
    }
  }

  async expectPeriodNotInTable(name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: name });
    const isVisible = await row
      .first()
      .isVisible({ timeout: 1500 })
      .catch(() => false);
    assert.strictEqual(
      isVisible,
      false,
      `Expected period "${name}" not to be in table`,
    );
  }

  get popconfirmOkButton(): Locator {
    return this.page.locator(
      ".ant-popover:not(.ant-popover-hidden) .ant-popconfirm-buttons .ant-btn-primary",
    );
  }

  async clickClosePeriod(name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: name });
    await row.first().waitFor({ state: "visible" });
    const closeBtn = row.first().locator("button:has(.anticon-lock)");
    await closeBtn.click();
    const okBtn = this.page
      .locator(".ant-popconfirm-buttons .ant-btn-primary")
      .last();
    await okBtn.waitFor({ state: "visible" });
    await okBtn.click();
    await okBtn.waitFor({ state: "hidden" });
  }

  async clickReopenPeriod(name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: name });
    await row.first().waitFor({ state: "visible" });
    const reopenBtn = row.first().locator("button:has(.anticon-unlock)");
    await reopenBtn.click();
    const okBtn = this.page
      .locator(".ant-popconfirm-buttons .ant-btn-primary")
      .last();
    await okBtn.waitFor({ state: "visible" });
    await okBtn.click();
    await okBtn.waitFor({ state: "hidden" });
  }

  async clickDeletePeriod(name: string): Promise<void> {
    const row = this.tableRows.filter({ hasText: name });
    await row.first().waitFor({ state: "visible" });
    const deleteBtn = row.first().locator("button:has(.anticon-delete)");
    await deleteBtn.click();
    const okBtn = this.page
      .locator(".ant-popconfirm-buttons .ant-btn-primary")
      .last();
    await okBtn.waitFor({ state: "visible" });
    await okBtn.click();
    await okBtn.waitFor({ state: "hidden" });
  }
}
