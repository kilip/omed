import assert from "node:assert/strict";
import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinDashboardPage extends BasePage {
  readonly path = "/fin/overview";

  get launcherTitle(): Locator {
    return this.page.getByRole("heading", {
      name: /^(Finance Apps|Aplikasi Keuangan|Finance|Keuangan)$/i,
    });
  }

  launcherTile(name: string): Locator {
    return this.page.locator(".app-tile").filter({ hasText: name });
  }

  async expectLauncherVisible(): Promise<void> {
    await this.launcherTitle.waitFor({ state: "visible" });
  }

  async clickLauncherTile(name: string): Promise<void> {
    const tile = this.launcherTile(name);
    await tile.first().waitFor({ state: "visible" });
    await tile.first().click();
  }

  get setupBanner(): Locator {
    return this.page.locator(".surface", {
      hasText: "Chart of accounts not configured",
    });
  }

  get setupButton(): Locator {
    return this.page.getByRole("button", { name: "Setup chart of accounts" });
  }

  get provisionalTag(): Locator {
    return this.page
      .locator(".ant-tag")
      .filter({ hasText: /provisional|sementara/i });
  }

  get revenueChart(): Locator {
    return this.page.locator("svg[role='img']");
  }

  async expectSetupBannerVisible(): Promise<void> {
    await this.setupBanner.waitFor({ state: "visible" });
  }

  async clickSetupButton(): Promise<void> {
    await this.setupButton.waitFor({ state: "visible" });
    await this.setupButton.click();
  }

  async expectNoProvisionalTags(): Promise<void> {
    const count = await this.provisionalTag.count();
    assert.equal(count, 0, "Expected no provisional tags on dashboard");
  }

  async expectStatistic(title: string, expectedValue: string): Promise<void> {
    const stat = this.page.locator(".ant-statistic").filter({ hasText: title });
    await stat.first().waitFor({ state: "visible" });
    const text = await stat.first().innerText();
    assert.ok(
      text.includes(expectedValue),
      `Expected statistic "${title}" to contain "${expectedValue}", got "${text}"`,
    );
  }

  async expectRecentEntry(text: string): Promise<void> {
    const row = this.page
      .locator(".ant-table-tbody .ant-table-row")
      .filter({ hasText: text });
    await row.first().waitFor({ state: "visible" });
  }

  async expectRevenueChartVisible(): Promise<void> {
    await this.revenueChart.waitFor({ state: "visible" });
  }
}
