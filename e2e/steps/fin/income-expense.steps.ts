import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new expense button", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.clickNewExpense();
});

When("I click the new income button", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.clickNewIncome();
});

When(
  "I select payment account {string}",
  async function (this: OmedWorld, account: string) {
    await this.pages.fin.incomeExpense.selectPaymentAccount(account);
  },
);

When(
  "I select deposit account {string}",
  async function (this: OmedWorld, account: string) {
    await this.pages.fin.incomeExpense.selectDepositAccount(account);
  },
);

When(
  "I select expense category {string}",
  async function (this: OmedWorld, account: string) {
    await this.pages.fin.incomeExpense.selectExpenseAccount(account);
  },
);

When(
  "I select revenue category {string}",
  async function (this: OmedWorld, account: string) {
    await this.pages.fin.incomeExpense.selectRevenueAccount(account);
  },
);

When(
  "I enter cashflow amount {int}",
  async function (this: OmedWorld, amount: number) {
    await this.pages.fin.incomeExpense.fillAmount(amount);
  },
);

When(
  "I enter cashflow amount {string}",
  async function (this: OmedWorld, amount: string) {
    await this.pages.fin.incomeExpense.fillAmount(amount);
  },
);

When(
  "I enter cashflow description {string}",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.incomeExpense.fillDescription(desc);
  },
);

When(
  "I enter cashflow reference {string}",
  async function (this: OmedWorld, ref: string) {
    await this.pages.fin.incomeExpense.fillReference(ref);
  },
);

Then(
  "I see cashflow accounting preview {string}",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.expectPreviewText(text);
  },
);

When("I save the cashflow form", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.saveForm();
});

Then(
  "I see cashflow item {string} in the table",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.expectItemInTable(text);
  },
);
