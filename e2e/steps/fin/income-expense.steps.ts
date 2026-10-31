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

When(
  "I add cashflow item {int} with account {string}, description {string}, quantity {int} and unit price {int}",
  async function (
    this: OmedWorld,
    index: number,
    account: string,
    description: string,
    quantity: number,
    unitPrice: number,
  ) {
    if (index > 0) {
      await this.pages.fin.incomeExpense.addItem();
    }
    await this.pages.fin.incomeExpense.fillItem(index, {
      account,
      description,
      quantity,
      unitPrice,
    });
  },
);

When("I click the cashflow posting button", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.clickPosting();
});

When("I click the cashflow draft button", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.clickDraft();
});

When(
  "I click cashflow row {string}",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.clickRow(text);
  },
);

When("I click the edit cashflow button", async function (this: OmedWorld) {
  await this.pages.fin.incomeExpense.clickEdit();
});

Then(
  "I see cashflow status {string}",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.expectStatusTag(text);
  },
);

Then(
  "I see cashflow accounting preview {string}",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.expectPreviewText(text);
  },
);

Then(
  "I see cashflow item {string} in the table",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.incomeExpense.expectItemInTable(text);
  },
);
