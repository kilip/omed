import assert from "node:assert/strict";
import { Then } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the homepage", async function (this: OmedWorld) {
  assert.ok(this.user, "Tag the scenario with @authenticated");
  await this.pages.core.home.expectRendered(this.user.name);
});
