import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

Then("I see the cms launcher", async function (this: OmedWorld) {
  await this.pages.cms.launcher.expectLoaded();
});

When(
  "I click the cms launcher tile {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.cms.launcher.clickTile(name);
  },
);

Then("I see the posts page", async function (this: OmedWorld) {
  await this.pages.cms.posts.expectLoaded();
});

Then("I see the pages page", async function (this: OmedWorld) {
  await this.pages.cms.pages.expectLoaded();
});

Then("I see the media library page", async function (this: OmedWorld) {
  await this.pages.cms.media.expectLoaded();
});
