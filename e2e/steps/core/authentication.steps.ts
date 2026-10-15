import { Then } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the login page", async function (this: OmedWorld) {
  await this.pages.core.login.expectLoaded();
});
