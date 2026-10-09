import { Then, When } from "@cucumber/cucumber";
import { expect } from "expect";
import {
  AccountsPage,
  CreateAccountPage,
  EditAccountPage,
  FinancePage,
  SeedCoaPage,
} from "../pages";
import type { CustomWorld } from "./world";

Then(
  "I should see the setup chart of accounts card",
  async function (this: CustomWorld) {
    const finPage = new FinancePage(this.page);
    await finPage.setupCoaButton.waitFor({ state: "visible", timeout: 10000 });
    expect(await finPage.setupCoaButton.isVisible()).toBe(true);
  },
);

When("I click setup chart of accounts", async function (this: CustomWorld) {
  const finPage = new FinancePage(this.page);
  await finPage.clickSetupCoa();
});

Then(
  "I should see the account preview table",
  async function (this: CustomWorld) {
    const seedPage = new SeedCoaPage(this.page);
    await seedPage.previewTable.waitFor({ state: "visible", timeout: 15000 });
    expect(await seedPage.previewTable.isVisible()).toBe(true);
  },
);

When(
  "I search preview accounts for {string}",
  async function (this: CustomWorld, query: string) {
    const seedPage = new SeedCoaPage(this.page);
    await seedPage.searchAccount(query);
  },
);

Then(
  "I should see the account row with code {string}",
  async function (this: CustomWorld, code: string) {
    const row = this.page.locator(".ant-table-row").filter({ hasText: code });
    await row.first().waitFor({ state: "visible", timeout: 5000 });
    expect(await row.first().isVisible()).toBe(true);
  },
);

Then(
  "I should not see the account row with code {string}",
  async function (this: CustomWorld, code: string) {
    const row = this.page.locator(".ant-table-row").filter({ hasText: code });
    const count = await row.count();
    expect(count).toBe(0);
  },
);

When("I click seed chart of accounts", async function (this: CustomWorld) {
  const seedPage = new SeedCoaPage(this.page);
  await seedPage.clickSeedAccounts();
});

Then(
  "I should see the chart of accounts seeded successfully",
  async function (this: CustomWorld) {
    const seedPage = new SeedCoaPage(this.page);
    await seedPage.expectSuccessResult();
    expect(await seedPage.resultSuccess.isVisible()).toBe(true);
  },
);

When("I click go to finance", async function (this: CustomWorld) {
  const seedPage = new SeedCoaPage(this.page);
  await seedPage.clickGoToFinance();
});

Then(
  "I should see the view accounts menu card",
  async function (this: CustomWorld) {
    const finPage = new FinancePage(this.page);
    await finPage.viewAccountsButton.waitFor({
      state: "visible",
      timeout: 10000,
    });
    expect(await finPage.viewAccountsButton.isVisible()).toBe(true);
  },
);

When("I click view accounts", async function (this: CustomWorld) {
  const finPage = new FinancePage(this.page);
  await finPage.clickViewAccounts();
});

When("I click create account button", async function (this: CustomWorld) {
  const accountsPage = new AccountsPage(this.page);
  await accountsPage.clickCreateAccount();
});

When(
  "I fill create account form with code {string}, name {string}, and type {string}",
  async function (this: CustomWorld, code: string, name: string, type: string) {
    const createPage = new CreateAccountPage(this.page);
    await createPage.fillForm({ code, name, type });
  },
);

When("I submit create account form", async function (this: CustomWorld) {
  const createPage = new CreateAccountPage(this.page);
  await createPage.clickSubmit();
});

When(
  "I click edit button for account with code {string}",
  async function (this: CustomWorld, code: string) {
    const accountsPage = new AccountsPage(this.page);
    await accountsPage.clickEditAccount(code);
  },
);

When(
  "I click account code {string}",
  async function (this: CustomWorld, code: string) {
    const accountsPage = new AccountsPage(this.page);
    await accountsPage.clickAccountCode(code);
  },
);

Then(
  "I should see account code {string} disabled in form",
  async function (this: CustomWorld, code: string) {
    const editPage = new EditAccountPage(this.page);
    await editPage.codeInput.waitFor({ state: "visible" });
    expect(await editPage.codeInput.isDisabled()).toBe(true);
    expect(await editPage.codeInput.inputValue()).toBe(code);
  },
);

When(
  "I update account name to {string}",
  async function (this: CustomWorld, name: string) {
    const editPage = new EditAccountPage(this.page);
    await editPage.fillName(name);
  },
);

When(
  "I update account description to {string}",
  async function (this: CustomWorld, description: string) {
    const editPage = new EditAccountPage(this.page);
    await editPage.fillDescription(description);
  },
);

When(
  "I select account status {string}",
  async function (this: CustomWorld, status: string) {
    const editPage = new EditAccountPage(this.page);
    await editPage.selectStatus(status);
  },
);

When("I submit edit account form", async function (this: CustomWorld) {
  const editPage = new EditAccountPage(this.page);
  await editPage.clickSubmit();
});

When("I click delete account button", async function (this: CustomWorld) {
  const editPage = new EditAccountPage(this.page);
  await editPage.clickDelete();
});

When("I confirm account deletion", async function (this: CustomWorld) {
  const editPage = new EditAccountPage(this.page);
  await editPage.confirmDelete();
});
