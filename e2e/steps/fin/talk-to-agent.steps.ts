import { Given, Then, When } from "@cucumber/cucumber";
import { FAKE_LLM_URL } from "../../support/fake-llm-server";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

async function ensureAgentAndAccounts(world: OmedWorld): Promise<void> {
  const ws = world.workspace;
  if (!ws) throw new Error("Workspace must exist");
  const token = world.sessionToken;
  if (!token) throw new Error("Session token must exist");

  // Ensure accounts seeded
  await testClient.authenticatedRequest(
    token,
    `/workspaces/${ws.id}/fin/accounts/seed`,
    {
      method: "POST",
      body: JSON.stringify({ template: "freelancer", lang: "en" }),
    },
  );

  // Ensure open period
  await testClient.ensureOpenPeriod(ws.id, token);

  // Ensure provider exists and is default
  await testClient.authenticatedRequest(
    token,
    `/workspaces/${ws.id}/agent/providers`,
    {
      method: "POST",
      body: JSON.stringify({
        name: "Test LLM Provider",
        kind: "openai_compat",
        model: "gpt-4o",
        apiKey: "sk-test-chat-key",
        baseUrl: FAKE_LLM_URL,
        isDefault: true,
      }),
    },
  );

  // Enable financial agent in settings
  await testClient.authenticatedRequest(
    token,
    `/workspaces/${ws.id}/agent/settings`,
    {
      method: "PATCH",
      body: JSON.stringify({
        financialEnabled: true,
      }),
    },
  );

  // Mark workspace as onboarded so navigation to /fin/* does not redirect to /fin/onboarding
  await testClient.authenticatedRequest(
    token,
    `/workspaces/${ws.id}/fin/settings`,
    {
      method: "PUT",
      body: JSON.stringify({ onboarding: true, decimalStyle: "dot" }),
    },
  );
}

Given(
  "the workspace has financial agent and accounts configured",
  async function (this: OmedWorld) {
    await ensureAgentAndAccounts(this);
  },
);

When("I navigate to the Talk to Agent page", async function (this: OmedWorld) {
  await this.pages.fin.talkToAgent.navigate();
});

When(
  "I send chat message {string}",
  async function (this: OmedWorld, content: string) {
    await this.pages.fin.talkToAgent.sendText(content);
  },
);

When(
  "I attach file {string} to the chat",
  async function (this: OmedWorld, fileName: string) {
    await this.pages.fin.talkToAgent.attachFile(fileName);
  },
);

When("I confirm the draft proposal", async function (this: OmedWorld) {
  await this.pages.fin.talkToAgent.confirmDraft();
});

When("I discard the draft proposal", async function (this: OmedWorld) {
  await this.pages.fin.talkToAgent.discardDraft();
});

Then(
  "I see message {string} in the chat",
  async function (this: OmedWorld, text: string) {
    await this.pages.fin.talkToAgent.expectMessage(text);
  },
);

Then(
  "I see a draft card of kind {string}",
  async function (this: OmedWorld, kind: string) {
    await this.pages.fin.talkToAgent.expectDraftCard(kind);
  },
);

Then(
  "I see attachment thumbnail {string}",
  async function (this: OmedWorld, fileName: string) {
    await this.pages.fin.talkToAgent.expectAttachmentThumbnail(fileName);
  },
);

Then("I see the confirmed draft state", async function (this: OmedWorld) {
  await this.pages.fin.talkToAgent.expectConfirmedState();
});

Then("I see the discarded draft state", async function (this: OmedWorld) {
  await this.pages.fin.talkToAgent.expectDiscardedState();
});

Then(
  "I see forbidden access denied for chat",
  async function (this: OmedWorld) {
    await this.pages.fin.talkToAgent.expectForbidden();
  },
);
