import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

Given("I have a personal workspace", async function (this: OmedWorld) {
  assert.ok(this.user, "Tag the scenario with @authenticated");
  const ws = await testClient.createWorkspace({
    name: "Personal Workspace",
    slug: `personal-${Date.now()}`,
    ownerId: this.user.id,
    personal: true,
  });
  this.workspace = ws;
  await testClient.resetEmails();
});

When(
  "I select the role {string}",
  async function (this: OmedWorld, roleId: string) {
    await this.pages.core.members.selectRole(roleId);
  },
);

Then(
  "the role {string} should be selected",
  async function (this: OmedWorld, roleId: string) {
    await this.pages.core.members.expectRoleChecked(roleId);
  },
);

Then(
  "the role {string} should not be selected",
  async function (this: OmedWorld, roleId: string) {
    await this.pages.core.members.expectRoleUnchecked(roleId);
  },
);

When(
  "I fill the member email {string}",
  async function (this: OmedWorld, email: string) {
    await this.pages.core.members.fillEmail(email);
  },
);

When("I submit the member invitation form", async function (this: OmedWorld) {
  await this.pages.core.members.submit();
});

Then(
  "{string} should be listed with role {string}",
  async function (this: OmedWorld, email: string, roleName: string) {
    await this.pages.core.members.expectMemberWithRole(email, roleName);
  },
);

Then(
  "the {string} menu should not be visible",
  async function (this: OmedWorld, name: string) {
    await this.pages.core.members.expectMenuHidden(name);
  },
);

Then(
  "I should see the personal workspace notice",
  async function (this: OmedWorld) {
    await this.pages.core.members.expectPersonalNotice();
  },
);
