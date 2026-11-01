import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When(
  "I create an access token named {string} with expiry {string}",
  async function (this: OmedWorld, name: string, expiry: string) {
    await this.pages.core.accessTokens.createToken(name, expiry);
  },
);

Then(
  "the generated token starting with {string} is displayed",
  async function (this: OmedWorld, prefix: string) {
    await this.pages.core.accessTokens.expectTokenRevealed(prefix);
  },
);

When("I close the token modal", async function (this: OmedWorld) {
  await this.pages.core.accessTokens.closeModal();
});

Then("the token reveal is no longer visible", async function (this: OmedWorld) {
  await this.pages.core.accessTokens.expectTokenRevealClosed();
});

Then(
  "the access token {string} is listed with status {string}",
  async function (this: OmedWorld, name: string, status: string) {
    await this.pages.core.accessTokens.expectTokenInTable(name, status);
  },
);

When(
  "I revoke the access token {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.accessTokens.revokeToken(name);
  },
);
