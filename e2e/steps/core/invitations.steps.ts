import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

Given("I have an active workspace", async function (this: OmedWorld) {
  assert.ok(this.user, "Tag the scenario with @authenticated");
  const ws = await testClient.createWorkspace({
    name: "Acme Corp",
    slug: `acme-${Date.now()}`,
    ownerId: this.user.id,
  });
  this.workspace = ws;
  await this.context.addInitScript((wid) => {
    localStorage.setItem("omed.workspace", String(wid));
  }, ws.id);
  await testClient.resetEmails();
});

When(
  "I create an invitation for {string} with role {string}",
  async function (this: OmedWorld, email: string, role: string) {
    await this.pages.core.invitations.openModal();
    await this.pages.core.invitations.selectRole(role);
    await this.pages.core.invitations.fillEmail(email);
    await this.pages.core.invitations.submit();
  },
);

Then(
  "an invitation email should be sent to {string}",
  async function (this: OmedWorld, recipientEmail: string) {
    const email = await testClient.getLastEmail();
    assert.ok(email, "No email was sent");
    const recipients = email.To?.map((t) => t.Address);
    assert.ok(
      recipients?.includes(recipientEmail),
      `Expected email to ${recipientEmail}, got ${recipients?.join(", ")}`,
    );

    const match =
      email.HTMLBody?.match(/href="([^"]*\/invite\/[^"]*)"/) ||
      email.TextBody?.match(/(https?:\/\/[^\s]+\/invite\/[^\s]+)/);
    assert.ok(match, "Invitation link not found in email body");
    this.invitationLink = match[1];
  },
);

When(
  "the recipient opens the invitation link",
  async function (this: OmedWorld) {
    assert.ok(this.invitationLink, "No invitation link stored");

    const { user, cookies } = await testClient.setupAuthenticatedUser({
      name: "Invited Colleague",
      email: "colleague@omed.test",
    });
    this.recipientUser = user;

    await this.context.clearCookies();
    await this.context.addCookies(cookies);

    await this.page.goto(this.invitationLink);
  },
);

Then(
  "the recipient can accept the invitation",
  async function (this: OmedWorld) {
    await this.pages.core.invite.expectCanAccept();
    await this.pages.core.invite.accept();
    await this.page.waitForURL((url) => url.pathname === "/", {
      timeout: 15_000,
    });
  },
);
