import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new period button", async function (this: OmedWorld) {
  await this.pages.fin.periods.clickNewPeriod();
});

When(
  "I fill the period form with name {string} and fiscal year {int}",
  async function (this: OmedWorld, name: string, fiscalYear: number) {
    await this.pages.fin.periods.fillPeriodForm({ name, fiscalYear });
  },
);

When("I click the save period button", async function (this: OmedWorld) {
  await this.pages.fin.periods.clickSave();
});

Then(
  "I see period {string} in the periods table",
  async function (this: OmedWorld, name: string) {
    await this.pages.fin.periods.expectPeriodInTable(name);
  },
);

Then(
  "I see period {string} with status {string} in the periods table",
  async function (this: OmedWorld, name: string, status: string) {
    await this.pages.fin.periods.expectPeriodInTable(name, status);
  },
);

Then(
  "I do not see period {string} in the periods table",
  async function (this: OmedWorld, name: string) {
    await this.pages.fin.periods.expectPeriodNotInTable(name);
  },
);

When("I close period {string}", async function (this: OmedWorld, name: string) {
  await this.pages.fin.periods.clickClosePeriod(name);
});

When(
  "I reopen period {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.fin.periods.clickReopenPeriod(name);
  },
);

When(
  "I delete period {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.fin.periods.clickDeletePeriod(name);
  },
);
