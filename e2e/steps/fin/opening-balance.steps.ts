import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When(
  "I input opening balance for account {string} with amount {string}",
  async function (this: OmedWorld, code: string, amount: string) {
    await this.pages.fin.openingBalance.inputAccountBalance(code, amount);
  },
);

When(
  "I click the save opening balance button",
  async function (this: OmedWorld) {
    await this.pages.fin.openingBalance.clickSave();
  },
);

Then(
  "I see the opening balance reentry notice",
  async function (this: OmedWorld) {
    await this.pages.fin.openingBalance.expectReentryNoticeVisible();
  },
);
