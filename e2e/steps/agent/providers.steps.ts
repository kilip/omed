import assert from "node:assert/strict";
import { type DataTable, Given, Then, When } from "@cucumber/cucumber";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

Given(
  "a workspace exists with a sample agent provider where I have role {string}",
  async function (this: OmedWorld, roleId: string) {
    assert.ok(this.user, "Tag the scenario with @authenticated");

    const owner = await testClient.createUser({
      name: "Workspace Owner",
      email: `owner+${crypto.randomUUID()}@omed.test`,
      emailVerified: true,
    });
    this.recipientUser = owner;

    const ws = await testClient.createWorkspace({
      name: "Agent Permissions Test",
      slug: `agent-perm-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      ownerId: owner.id,
      personal: false,
    });
    this.workspace = ws;

    const ownerSession = await testClient.createSession({ userId: owner.id });

    // Seed sample provider
    await testClient.authenticatedRequest(
      ownerSession.token,
      `/workspaces/${ws.id}/agent/providers`,
      {
        method: "POST",
        body: JSON.stringify({
          name: "Sample Provider",
          kind: "openai_compat",
          model: "gpt-4o",
          apiKey: "sk-test-sample-key",
          baseUrl: "http://127.0.0.1:4010/v1",
        }),
      },
    );

    await testClient.addMember({
      workspaceId: ws.id,
      userId: this.user.id,
      role: roleId,
    });

    await this.context.addInitScript((wid) => {
      localStorage.setItem("omed.workspace", String(wid));
    }, ws.id);
  },
);

Then("I see the providers page", async function (this: OmedWorld) {
  await this.pages.agent.providers.expectProvidersPageLoaded();
});

When("I click the add provider button", async function (this: OmedWorld) {
  await this.pages.agent.providers.clickAddProvider();
});

When(
  "I fill the provider form with:",
  async function (this: OmedWorld, dataTable: DataTable) {
    const rows = dataTable.rowsHash();
    if (rows.name) await this.pages.agent.providers.fillName(rows.name);
    if (rows.kind) await this.pages.agent.providers.selectKind(rows.kind);
    if (rows.model) await this.pages.agent.providers.fillModel(rows.model);
    if (rows.baseUrl)
      await this.pages.agent.providers.fillBaseUrl(rows.baseUrl);
    if (rows.apiKey) await this.pages.agent.providers.fillApiKey(rows.apiKey);
    if (rows.isDefault === "true") {
      await this.pages.agent.providers.toggleDefault();
    }
  },
);

When("I submit the provider form", async function (this: OmedWorld) {
  await this.pages.agent.providers.submitForm();
});

Then(
  "I see {string} in the providers table",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectProviderInTable(name);
  },
);

Then(
  "I do not see {string} in the providers table",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectProviderNotInTable(name);
  },
);

When(
  "I test the connection for provider {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.clickTestConnection(name);
  },
);

Then(
  "the connection test for {string} succeeds",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectTestSuccess(name);
  },
);

When(
  "I set {string} as default provider",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.clickSetDefault(name);
  },
);

Then(
  "{string} is marked as default",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectProviderIsDefault(name);
  },
);

When(
  "I delete the provider {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.clickDelete(name);
  },
);

Then("the add provider button is hidden", async function (this: OmedWorld) {
  await this.pages.agent.providers.expectAddButtonHidden();
});

Then(
  "the test connection button is hidden for {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectTestButtonHidden(name);
  },
);

Then(
  "the set default button is hidden for {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectSetDefaultButtonHidden(name);
  },
);

Then(
  "the edit provider button is hidden for {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectEditButtonHidden(name);
  },
);

Then(
  "the delete provider button is hidden for {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.providers.expectDeleteButtonHidden(name);
  },
);
