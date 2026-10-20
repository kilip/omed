import { Then } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then(
  "I see financial statistic {string} with amount {string}",
  async function (this: OmedWorld, title: string, amount: string) {
    await this.pages.fin.dashboard.expectStatistic(title, amount);
  },
);

Then(
  "I see no provisional tags on dashboard",
  async function (this: OmedWorld) {
    await this.pages.fin.dashboard.expectNoProvisionalTags();
  },
);

Then(
  "I see recent journal entry {string} in dashboard",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.dashboard.expectRecentEntry(text);
  },
);

Then("I see the revenue vs expense chart", async function (this: OmedWorld) {
  await this.pages.fin.dashboard.expectRevenueChartVisible();
});
