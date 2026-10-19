import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I click the new workspace button", async function (this: OmedWorld) {
  await this.pages.core.workspaces.clickNewWorkspace();
});

When(
  "I fill in the workspace name with {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.fillName(name);
  },
);

When(
  "I fill in the workspace description with {string}",
  async function (this: OmedWorld, description: string) {
    await this.pages.core.workspaces.fillDescription(description);
  },
);

When("I submit the workspace form", async function (this: OmedWorld) {
  await this.pages.core.workspaces.submit();
});

Then(
  "I see {string} in the workspaces table",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.expectWorkspaceInTable(name);
  },
);

Then(
  "the active workspace is {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.expectActiveWorkspace(name);
  },
);

When(
  "I click the switch button for workspace {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.clickSwitchWorkspace(name);
  },
);

When(
  "I click the workspace row {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.clickWorkspaceRow(name);
  },
);

Then(
  "workspace {string} has the active tag",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.expectActiveTagInRow(name);
  },
);

Then(
  "workspace {string} has the switch button",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.workspaces.expectSwitchButtonInRow(name);
  },
);

Then("I am on workspace edit page", async function (this: OmedWorld) {
  await this.pages.core.workspaces.expectEditPageLoaded();
});
