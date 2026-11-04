import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the finance settings page", async function (this: OmedWorld) {
  await this.pages.fin.settings.expectLoaded();
});

When(
  "I select base currency {string} in finance settings",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.settings.selectBaseCurrency(code);
  },
);

Then(
  "the base currency in finance settings is {string}",
  async function (this: OmedWorld, code: string) {
    await this.pages.fin.settings.expectBaseCurrency(code);
  },
);

When(
  "I select decimal style {string} in finance settings",
  async function (this: OmedWorld, style: string) {
    if (style !== "dot" && style !== "comma") {
      throw new Error(`Invalid decimal style: ${style}`);
    }
    await this.pages.fin.settings.selectDecimalStyle(style);
  },
);

Then(
  "the decimal style in finance settings is {string}",
  async function (this: OmedWorld, style: string) {
    if (style !== "dot" && style !== "comma") {
      throw new Error(`Invalid decimal style: ${style}`);
    }
    await this.pages.fin.settings.expectDecimalStyle(style);
  },
);

When(
  "I save currency and number format settings",
  async function (this: OmedWorld) {
    await this.pages.fin.settings.saveCurrencyFormat();
  },
);

When(
  "I select FX account {string} for {string}",
  async function (this: OmedWorld, accountCode: string, label: string) {
    await this.pages.fin.settings.selectFxAccount(label, accountCode);
  },
);

Then(
  "the FX account for {string} is {string}",
  async function (this: OmedWorld, label: string, accountCode: string) {
    await this.pages.fin.settings.expectFxAccount(label, accountCode);
  },
);

When("I save FX account mappings", async function (this: OmedWorld) {
  await this.pages.fin.settings.saveFxMappings();
});

When(
  "I create a fiscal period {string} for fiscal year {int} from finance settings",
  async function (this: OmedWorld, name: string, fiscalYear: number) {
    await this.pages.fin.settings.createPeriod({ fiscalYear, name });
  },
);

Then(
  "I see period {string} with status {string} in finance settings",
  async function (this: OmedWorld, name: string, status: string) {
    await this.pages.fin.settings.expectPeriodInTable(name, status);
  },
);

Then(
  "I see the finance settings saved message",
  async function (this: OmedWorld) {
    await this.pages.fin.settings.expectSuccessMessage(
      /settings saved successfully/i,
    );
  },
);

Then(
  "I see the fiscal period created message",
  async function (this: OmedWorld) {
    await this.pages.fin.settings.expectSuccessMessage(
      /fiscal period created successfully/i,
    );
  },
);
