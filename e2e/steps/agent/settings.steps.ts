import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import { FAKE_LLM_URL } from "../../support/fake-llm-server";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

async function seedProvider(
  world: OmedWorld,
  name: string,
  isDefault: boolean,
): Promise<void> {
  assert.ok(world.workspace, "A workspace must exist first");
  assert.ok(world.sessionToken, "Tag the scenario with @authenticated");
  await testClient.authenticatedRequest(
    world.sessionToken,
    `/workspaces/${world.workspace.id}/agent/providers`,
    {
      method: "POST",
      body: JSON.stringify({
        name,
        kind: "openai_compat",
        model: "gpt-4o",
        apiKey: "sk-test-settings-key",
        baseUrl: FAKE_LLM_URL,
        isDefault,
      }),
    },
  );
}

Given(
  "the workspace has a default agent provider {string}",
  async function (this: OmedWorld, name: string) {
    await seedProvider(this, name, true);
  },
);

Given(
  "the workspace has an agent provider {string}",
  async function (this: OmedWorld, name: string) {
    await seedProvider(this, name, false);
  },
);

Then("I see the agent settings page", async function (this: OmedWorld) {
  await this.pages.agent.settings.expectLoaded();
});

When("I enable the financial agent", async function (this: OmedWorld) {
  await this.pages.agent.settings.toggleEnabled();
});

When(
  "I select the financial provider {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.settings.selectProvider(name);
  },
);

When("I save the agent settings", async function (this: OmedWorld) {
  await this.pages.agent.settings.save();
});

Then(
  "I see the agent settings saved message",
  async function (this: OmedWorld) {
    await this.pages.agent.settings.expectSavedMessage();
  },
);

Then(
  "I see the agent settings error {string}",
  async function (this: OmedWorld, text: string) {
    await this.pages.agent.settings.expectError(text);
  },
);

Then("the financial agent is enabled", async function (this: OmedWorld) {
  await this.pages.agent.settings.expectEnabled(true);
});

Then("the financial agent is disabled", async function (this: OmedWorld) {
  await this.pages.agent.settings.expectEnabled(false);
});

Then(
  "the financial provider is {string}",
  async function (this: OmedWorld, name: string) {
    await this.pages.agent.settings.expectProvider(name);
  },
);

Then(
  "the financial agent switch is disabled",
  async function (this: OmedWorld) {
    await this.pages.agent.settings.expectSwitchDisabled();
  },
);

Then(
  "the financial provider select is disabled",
  async function (this: OmedWorld) {
    await this.pages.agent.settings.expectProviderSelectDisabled();
  },
);

Then(
  "the save agent settings button is hidden",
  async function (this: OmedWorld) {
    await this.pages.agent.settings.expectSaveButtonHidden();
  },
);
