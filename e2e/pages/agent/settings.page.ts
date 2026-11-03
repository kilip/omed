import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class AgentSettingsPage extends BasePage {
  readonly path = "/agent/settings";

  get enabledSwitch(): Locator {
    return this.page.locator("#financialEnabled");
  }

  get providerSelect(): Locator {
    return this.page.locator(".ant-select:has(#financialProviderId)");
  }

  get saveButton(): Locator {
    return this.page.getByRole("button", { name: /save settings/i });
  }

  get errorAlert(): Locator {
    return this.page.locator(".ant-alert-error");
  }

  async expectLoaded(): Promise<void> {
    await this.enabledSwitch.waitFor({ state: "visible", timeout: 10_000 });
  }

  async toggleEnabled(): Promise<void> {
    await this.enabledSwitch.click();
  }

  async selectProvider(name: string): Promise<void> {
    await this.providerSelect.click();
    const option = this.page
      .locator(
        ".ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option",
      )
      .filter({ hasText: name })
      .first();
    await option.waitFor({ state: "visible" });
    await option.click();
  }

  async save(): Promise<void> {
    await this.saveButton.click();
  }

  async expectEnabled(expected: boolean): Promise<void> {
    await this.page
      .locator(`#financialEnabled[aria-checked="${expected}"]`)
      .waitFor({ state: "visible", timeout: 10_000 });
  }

  async expectProvider(name: string): Promise<void> {
    await this.providerSelect
      .filter({ hasText: name })
      .waitFor({ state: "visible", timeout: 10_000 })
      .catch(async () => {
        const actual = await this.providerSelect.innerText();
        assert.fail(
          `Expected financial provider "${name}", select shows "${actual}"`,
        );
      });
  }

  async expectSavedMessage(): Promise<void> {
    await this.page
      .locator(".ant-message-success", {
        hasText: /settings saved successfully/i,
      })
      .waitFor({ state: "visible", timeout: 10_000 });
  }

  async expectError(text: string): Promise<void> {
    await this.errorAlert.waitFor({ state: "visible", timeout: 10_000 });
    const content = await this.errorAlert.innerText();
    assert.ok(
      content.toLowerCase().includes(text.toLowerCase()),
      `Expected error containing "${text}", got "${content}"`,
    );
  }

  async expectSwitchDisabled(): Promise<void> {
    assert.strictEqual(
      await this.enabledSwitch.isDisabled(),
      true,
      "Expected the financial agent switch to be disabled",
    );
  }

  async expectProviderSelectDisabled(): Promise<void> {
    await this.page
      .locator(".ant-select-disabled:has(#financialProviderId)")
      .waitFor({ state: "visible", timeout: 5000 });
  }

  async expectSaveButtonHidden(): Promise<void> {
    const visible = await this.saveButton
      .isVisible({ timeout: 1500 })
      .catch(() => false);
    assert.strictEqual(visible, false, "Expected save button to be hidden");
  }
}
