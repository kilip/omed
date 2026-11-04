import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the agent launcher", async function (this: OmedWorld) {
  await this.pages.agent.launcher.expectLoaded();
});

When(
  "I click the agent launcher tile {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.launcher.clickTile(name);
  },
);
