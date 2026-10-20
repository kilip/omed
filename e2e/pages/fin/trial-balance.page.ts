import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinTrialBalancePage extends BasePage {
  readonly path = "/fin/trial-balance";

  get table(): Locator {
    return this.page.locator(".ant-table");
  }

  get tableRows(): Locator {
    return this.page.locator(".ant-table-tbody .ant-table-row");
  }

  get datePickerInput(): Locator {
    return this.page.locator(".ant-picker-input input");
  }

  get statusTag(): Locator {
    return this.page.locator(".ant-tag").first();
  }

  get summaryRow(): Locator {
    return this.page.locator(".ant-table-summary");
  }

  get provisionalAlert(): Locator {
    return this.page.locator(".ant-alert");
  }

  async expectLoaded(): Promise<void> {
    await this.table.waitFor({ state: "visible" });
    await this.statusTag.waitFor({ state: "visible" });
  }

  async expectNoProvisionalAlert(): Promise<void> {
    const count = await this.provisionalAlert.count();
    assert.equal(
      count,
      0,
      "Expected no provisional notice alert on Trial Balance page",
    );
  }

  async expectBalancedTag(balanced: boolean): Promise<void> {
    await this.statusTag.waitFor({ state: "visible" });
    const text = (await this.statusTag.innerText()).trim().toLowerCase();
    if (balanced) {
      assert.ok(
        text.includes("balanced") || text.includes("seimbang"),
        `Expected status tag to indicate balanced, got "${text}"`,
      );
    } else {
      assert.ok(
        text.includes("unbalanced") || text.includes("tidak seimbang"),
        `Expected status tag to indicate unbalanced, got "${text}"`,
      );
    }
  }

  async expectAccountBalance(
    code: string,
    expectedDebitOrCredit: string,
  ): Promise<void> {
    const row = this.tableRows.filter({ hasText: code });
    await row.first().waitFor({ state: "visible" });
    const text = await row.first().innerText();
    assert.ok(
      text.includes(expectedDebitOrCredit),
      `Expected row for account ${code} to contain "${expectedDebitOrCredit}", got "${text}"`,
    );
  }

  async expectSummaryTotals(totalAmount: string): Promise<void> {
    await this.summaryRow.waitFor({ state: "visible" });
    const text = await this.summaryRow.innerText();
    assert.ok(
      text.includes(totalAmount),
      `Expected summary row to contain "${totalAmount}", got "${text}"`,
    );
  }

  async setAsOfDate(dateStr: string): Promise<void> {
    await this.datePickerInput.waitFor({ state: "visible" });
    await this.datePickerInput.fill(dateStr);
    await this.datePickerInput.press("Enter");
    await this.page.waitForLoadState("networkidle");
  }
}
