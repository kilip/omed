import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new journal entry button", async function (this: OmedWorld) {
  await this.pages.fin.entries.clickNewEntry();
});

Then("I see the create entry drawer", async function (this: OmedWorld) {
  await this.pages.fin.entries.expectDrawerVisible();
});

Then("I see the create entry page", async function (this: OmedWorld) {
  await this.pages.fin.entries.expectPageVisible();
});

When(
  "I enter entry description {string}",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.entries.fillDescription(desc);
  },
);

When(
  "I add a debit line of {int} for account {string}",
  async function (this: OmedWorld, amount: number, account: string) {
    await this.pages.fin.entries.fillPostingLine(0, {
      accountSearch: account,
      debit: amount,
    });
  },
);

When(
  "I add a debit line of quantity {int} and unit price {int} for account {string}",
  async function (
    this: OmedWorld,
    qty: number,
    unitPrice: number,
    account: string,
  ) {
    await this.pages.fin.entries.fillPostingLine(0, {
      accountSearch: account,
      quantity: qty,
      unitPrice: unitPrice,
      debit: qty * unitPrice,
    });
  },
);

When(
  "I add a credit line of {int} for account {string}",
  async function (this: OmedWorld, amount: number, account: string) {
    await this.pages.fin.entries.fillPostingLine(1, {
      accountSearch: account,
      credit: amount,
    });
  },
);

When("I save the entry drawer", async function (this: OmedWorld) {
  await this.pages.fin.entries.saveDrawer();
});

When("I save the entry form", async function (this: OmedWorld) {
  await this.pages.fin.entries.saveForm();
});

Then(
  "I see entry {string} in the journal list",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.entries.expectEntryInTable(text);
  },
);

When(
  "I click on entry {string} in the list",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.entries.clickRow(text);
  },
);

Then(
  "I see the entry status is {string}",
  async function (this: OmedWorld, status: string) {
    await this.pages.fin.entryDetail.expectStatus(status);
  },
);

When("I click the post entry button", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.clickPost();
});

When("I click the edit entry button", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.clickEdit();
});

When(
  "I change entry description to {string}",
  async function (this: OmedWorld, desc: string) {
    await this.pages.fin.entryDetail.fillDescription(desc);
  },
);

When("I save the entry changes", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.clickSaveChanges();
});

When("I click the reverse entry button", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.clickReverse();
});

When("I confirm the reverse modal", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.submitReverseModal();
});

When("I click the void entry button", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.clickVoid();
});

Then("I see the entry unbalanced warning", async function (this: OmedWorld) {
  await this.pages.fin.entries.expectUnbalancedWarning();
});
