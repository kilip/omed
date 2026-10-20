import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the trial balance table", async function (this: OmedWorld) {
  await this.pages.fin.trialBalance.expectLoaded();
});

Then(
  "I see no provisional notice in trial balance",
  async function (this: OmedWorld) {
    await this.pages.fin.trialBalance.expectNoProvisionalAlert();
  },
);

Then(
  "I see the trial balance status is balanced",
  async function (this: OmedWorld) {
    await this.pages.fin.trialBalance.expectBalancedTag(true);
  },
);

Then(
  "I see account {string} has balance {string} in trial balance",
  async function (this: OmedWorld, code: string, amount: string) {
    await this.pages.fin.trialBalance.expectAccountBalance(code, amount);
  },
);

Then(
  "I see total trial balance amount {string}",
  async function (this: OmedWorld, amount: string) {
    await this.pages.fin.trialBalance.expectSummaryTotals(amount);
  },
);

When(
  "I set trial balance as of date to {string}",
  async function (this: OmedWorld, date: string) {
    await this.pages.fin.trialBalance.setAsOfDate(date);
  },
);
