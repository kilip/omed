import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When(
  "I select ledger account {string}",
  async function (this: OmedWorld, account: string) {
    await this.pages.fin.ledger.selectAccount(account);
  },
);

Then(
  "I see posting with description {string} in the ledger",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.ledger.expectPostingInTable(desc);
  },
);

Then("I see the ledger running balance", async function (this: OmedWorld) {
  await this.pages.fin.ledger.expectRunningBalanceVisible();
});

When(
  "I click account {string} ledger link in accounts table",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.accounts.clickAccountLedger(code);
  },
);

When(
  "I click posting line account {string} in entry detail",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.entryDetail.clickPostingAccount(code);
  },
);

When("I click entry link in ledger row", async function (this: OmedWorld) {
  await this.pages.fin.ledger.clickEntryLinkInRow("");
});
