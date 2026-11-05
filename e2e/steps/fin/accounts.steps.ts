import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new account button", async function (this: OmedWorld) {
  await this.pages.fin.accounts.clickNewAccount();
});

Then("I see the account drawer", async function (this: OmedWorld) {
  await this.pages.fin.accounts.expectDrawerVisible();
});

When(
  "I fill the account form with code {string}, name {string}, type {string}",
  async function (this: OmedWorld, code: string, name: string, type: string) {
    await this.pages.fin.accounts.fillAccountForm(code, name, type);
  },
);

When("I click the save account button", async function (this: OmedWorld) {
  await this.pages.fin.accounts.saveAccount();
});

When(
  "I click the edit button for account {string}",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.accounts.clickEditAccount(code);
  },
);

When(
  "I update the account name to {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.fin.accounts.updateAccountName(name);
  },
);

When("I toggle the account active switch", async function (this: OmedWorld) {
  await this.pages.fin.accounts.toggleActiveSwitch();
});

When(
  "I delete account {string}",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.accounts.deleteAccount(code);
  },
);

Then(
  "I see account {string} with status {string} in the accounts table",
  async function (this: OmedWorld, code: string, status: string) {
    await this.pages.fin.accounts.expectAccountStatus(code, status);
  },
);

Then(
  "I see account {string} with name {string} in the accounts table",
  async function (this: OmedWorld, code: string, name: string) {
    await this.pages.fin.accounts.expectAccountName(code, name);
  },
);

Then(
  "I do not see account {string} in the accounts table",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.accounts.expectAccountNotInTable(code);
  },
);
