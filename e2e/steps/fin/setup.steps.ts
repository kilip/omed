import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then(
  "I see the setup chart of accounts banner",
  async function (this: OmedWorld) {
    await this.pages.fin.dashboard.expectSetupBannerVisible();
  },
);

When(
  "I click the setup chart of accounts banner button",
  async function (this: OmedWorld) {
    await this.pages.fin.dashboard.clickSetupButton();
  },
);

When("I click the apply template button", async function (this: OmedWorld) {
  await this.pages.fin.setup.clickApplyTemplate();
});

Then("I see the accounts empty state", async function (this: OmedWorld) {
  await this.pages.fin.accounts.expectEmptyStateVisible();
});

When(
  "I click the seed template button in empty state",
  async function (this: OmedWorld) {
    await this.pages.fin.accounts.clickEmptyStateSeedButton();
  },
);

Then(
  "I see account {string} in the accounts table",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.accounts.expectAccountInTable(code);
  },
);

Then(
  "I do not see the setup template button",
  async function (this: OmedWorld) {
    await this.pages.fin.accounts.expectSetupTemplateButtonHidden();
  },
);

Then("I see the setup initial period form", async function (this: OmedWorld) {
  await this.pages.fin.setup.expectPeriodFormVisible();
});
