import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

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
