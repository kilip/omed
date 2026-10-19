import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new transaction button", async function (this: OmedWorld) {
  await this.pages.fin.transactions.clickNewTransaction();
});

Then("I see the create transaction drawer", async function (this: OmedWorld) {
  await this.pages.fin.transactions.expectDrawerVisible();
});

When(
  "I enter transaction description {string}",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.transactions.fillDescription(desc);
  },
);

When(
  "I enter transaction reference {string}",
  async function (this: OmedWorld, ref: string) {
    await this.pages.fin.transactions.fillReference(ref);
  },
);

When(
  "I select transaction currency {string}",
  async function (this: OmedWorld, currency: string) {
    await this.pages.fin.transactions.selectCurrency(currency);
  },
);

When(
  "I enter transaction exchange rate {int}",
  async function (this: OmedWorld, rate: number) {
    await this.pages.fin.transactions.fillExchangeRate(rate);
  },
);

When("I save the transaction drawer", async function (this: OmedWorld) {
  await this.pages.fin.transactions.saveDrawer();
});

Then(
  "I see transaction {string} in the transactions list",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.transactions.expectTransactionInTable(text);
  },
);

When(
  "I click on transaction {string} in the list",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.transactions.clickRow(text);
  },
);

When(
  "I search transactions for {string}",
  async function (this: OmedWorld, query: string) {
    await this.pages.fin.transactions.search(query);
  },
);

When("I click the edit transaction button", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.clickEdit();
});

When(
  "I change transaction reference to {string}",
  async function (this: OmedWorld, ref: string) {
    await this.pages.fin.transactionDetail.fillReference(ref);
  },
);

When(
  "I change transaction description to {string}",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.transactionDetail.fillDescription(desc);
  },
);

When("I save the transaction changes", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.saveChanges();
});

When("I click the delete transaction button", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.clickDelete();
});

When("I confirm delete transaction", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.confirmDelete();
});

When(
  "I click add entry to transaction button",
  async function (this: OmedWorld) {
    await this.pages.fin.transactionDetail.clickAddEntry();
  },
);

Then(
  "I see entry {string} in transaction entries table",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.transactionDetail.expectEntryInTable(text);
  },
);

When(
  "I click entry {string} in transaction entries table",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.transactionDetail.clickEntryRow(text);
  },
);

Then(
  "I see text {string} on transaction detail",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.transactionDetail.expectText(text);
  },
);

Then(
  "I cannot edit or delete the transaction",
  async function (this: OmedWorld) {
    await this.pages.fin.transactionDetail.expectNoEditButton();
    await this.pages.fin.transactionDetail.expectNoDeleteButton();
  },
);
