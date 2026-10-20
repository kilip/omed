import assert from "node:assert/strict";
import { Then, When } from "@cucumber/cucumber";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

Then("I see the homepage", async function (this: OmedWorld) {
  assert.ok(this.user, "Tag the scenario with @authenticated");
  await this.pages.core.home.expectRendered(this.user.name);
});

When(
  "I switch language to {string}",
  async function (this: OmedWorld, language: string) {
    await this.pages.core.home.switchLanguage(language);
  },
);

Then(
  "I see the language is {string}",
  async function (this: OmedWorld, code: "EN" | "ID") {
    await this.pages.core.home.expectLanguage(code);
  },
);

Then(
  "I see the launcher title is {string}",
  async function (this: OmedWorld, title: string) {
    await this.pages.core.home.expectHeading(title);
  },
);

Then(
  "my user preference locale is {string}",
  async function (this: OmedWorld, expectedLocale: string) {
    assert.ok(this.sessionToken, "Tag the scenario with @authenticated");
    const me = await testClient.getMe(this.sessionToken);
    assert.equal(
      me.locale,
      expectedLocale,
      `Expected user locale to be "${expectedLocale}", got "${me.locale}"`,
    );
  },
);
