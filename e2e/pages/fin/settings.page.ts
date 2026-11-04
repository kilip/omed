import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinSettingsPage extends BasePage {
  readonly path = "/fin/settings";

  get heading(): Locator {
    return this.page.getByRole("heading", { name: /^finance settings$/i });
  }

  get currencyPanel(): Locator {
    return this.page.locator("section").filter({
      has: this.page.getByRole("heading", {
        name: /currency & number format/i,
      }),
    });
  }

  get periodsPanel(): Locator {
    return this.page.locator("section").filter({
      has: this.page.getByRole("heading", { name: /fiscal periods/i }),
    });
  }

  get coaFxPanel(): Locator {
    return this.page.locator("section").filter({
      has: this.page.getByRole("heading", {
        name: /chart of accounts & fx mappings/i,
      }),
    });
  }

  get baseCurrencySelect(): Locator {
    return this.currencyPanel.locator(".ant-select").first();
  }

  get saveCurrencyButton(): Locator {
    return this.currencyPanel.getByRole("button", { name: /save changes/i });
  }

  get newPeriodButton(): Locator {
    return this.periodsPanel.getByRole("button", { name: /new period/i });
  }

  get periodTableRows(): Locator {
    return this.periodsPanel.locator(".ant-table-tbody .ant-table-row");
  }

  get saveFxButton(): Locator {
    return this.coaFxPanel.getByRole("button", { name: /save changes/i });
  }

  get modal(): Locator {
    return this.page.locator(".ant-modal");
  }

  get modalFiscalYearInput(): Locator {
    return this.modal.getByRole("spinbutton", {
      name: /fiscal year|tahun fiskal/i,
    });
  }

  get modalPeriodNameInput(): Locator {
    return this.modal.getByRole("textbox", {
      name: /period name|nama periode/i,
    });
  }

  get modalOkButton(): Locator {
    return this.modal.getByRole("button", { name: /^ok$/i });
  }

  private decimalCard(style: "dot" | "comma"): Locator {
    const text =
      style === "dot"
        ? /dot decimal|format titik/i
        : /comma decimal|format koma/i;
    return this.currencyPanel
      .locator(".ant-col .ant-card")
      .filter({ hasText: text });
  }

  private fxAccountSelect(label: string): Locator {
    return this.coaFxPanel
      .locator("div.flex.flex-col.gap-1")
      .filter({ hasText: label })
      .locator(".ant-select")
      .first();
  }

  async expectLoaded(): Promise<void> {
    await this.heading.waitFor({ state: "visible", timeout: 10_000 });
    await this.currencyPanel.waitFor({ state: "visible", timeout: 10_000 });
    await this.periodsPanel.waitFor({ state: "visible", timeout: 10_000 });
    await this.coaFxPanel.waitFor({ state: "visible", timeout: 10_000 });
  }

  async selectBaseCurrency(code: string): Promise<void> {
    await this.baseCurrencySelect.click();
    const searchInput = this.baseCurrencySelect.locator("input");
    await searchInput.fill(code);
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: code })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async expectBaseCurrency(code: string): Promise<void> {
    await this.baseCurrencySelect
      .filter({ hasText: code })
      .waitFor({ state: "visible", timeout: 10_000 })
      .catch(async () => {
        const actual = await this.baseCurrencySelect.innerText();
        assert.fail(
          `Expected base currency "${code}", select shows "${actual}"`,
        );
      });
  }

  async selectDecimalStyle(style: "dot" | "comma"): Promise<void> {
    const card = this.decimalCard(style);
    await card.waitFor({ state: "visible" });
    await card.click();
  }

  async expectDecimalStyle(style: "dot" | "comma"): Promise<void> {
    const card = this.decimalCard(style);
    await card
      .locator(".ant-radio-checked")
      .waitFor({ state: "visible", timeout: 10_000 });
  }

  async saveCurrencyFormat(): Promise<void> {
    await this.saveCurrencyButton.click();
  }

  async selectFxAccount(label: string, accountCode: string): Promise<void> {
    const select = this.fxAccountSelect(label);
    await select.waitFor({ state: "visible" });
    await select.click();
    const searchInput = select.locator("input");
    await searchInput.fill(accountCode);
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: accountCode })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async expectFxAccount(label: string, accountCode: string): Promise<void> {
    const select = this.fxAccountSelect(label);
    await select
      .filter({ hasText: accountCode })
      .waitFor({ state: "visible", timeout: 10_000 })
      .catch(async () => {
        const actual = await select.innerText();
        assert.fail(
          `Expected FX account "${label}" to show "${accountCode}", got "${actual}"`,
        );
      });
  }

  async saveFxMappings(): Promise<void> {
    await this.saveFxButton.click();
  }

  async createPeriod(values: {
    fiscalYear: number;
    name: string;
  }): Promise<void> {
    await this.newPeriodButton.waitFor({ state: "visible" });
    await this.newPeriodButton.click();
    await this.modal.waitFor({ state: "visible" });
    await this.modalFiscalYearInput.fill(String(values.fiscalYear));
    await this.modalFiscalYearInput.press("Enter");
    await this.modalPeriodNameInput.fill(values.name);
    await this.modalOkButton.click();
    await this.modal.waitFor({ state: "hidden", timeout: 10_000 });
  }

  async expectPeriodInTable(name: string, status?: string): Promise<void> {
    let row = this.periodTableRows.filter({ hasText: name });
    if (status) {
      row = row.filter({ hasText: status });
    }
    await row.first().waitFor({ state: "visible", timeout: 10_000 });
  }

  async expectSuccessMessage(text: RegExp): Promise<void> {
    const toast = this.page
      .locator(".ant-message-success")
      .filter({ hasText: text })
      .first();
    await toast.waitFor({ state: "visible", timeout: 10_000 });
  }
}
