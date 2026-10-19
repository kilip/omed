import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import { testClient } from "../../support/test-client";
import type { OmedWorld } from "../../support/world";

Given(
  "a workspace exists with sample finance data where I have role {string}",
  async function (this: OmedWorld, roleId: string) {
    assert.ok(this.user, "Tag the scenario with @authenticated");

    // 1. Create a workspace owner
    const owner = await testClient.createUser({
      name: "Workspace Owner",
      email: `owner+${crypto.randomUUID()}@omed.test`,
      emailVerified: true,
    });
    this.recipientUser = owner;

    // 2. Create the workspace owned by that owner
    const ws = await testClient.createWorkspace({
      name: "Finance Permissions Test",
      slug: `fin-perm-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      ownerId: owner.id,
      personal: false,
    });
    this.workspace = ws;

    // 3. Create session for owner to seed data via API
    const ownerSession = await testClient.createSession({ userId: owner.id });

    // Seed COA
    await testClient.authenticatedRequest(
      ownerSession.token,
      `/workspaces/${ws.id}/fin/accounts/seed`,
      {
        method: "POST",
        body: JSON.stringify({ template: "freelancer", lang: "en" }),
      },
    );

    // Get accounts to find valid postable debit & credit accounts
    const accountsRes = await testClient.authenticatedRequest<{
      data?: Array<{ id?: number; code?: string; isPostable?: boolean }>;
    }>(ownerSession.token, `/workspaces/${ws.id}/fin/accounts/search`, {
      method: "POST",
      body: JSON.stringify({ limit: 100 }),
    });

    const accounts = accountsRes.data ?? [];
    const debitAcc =
      accounts.find((a) => a.code === "5100") ??
      accounts.find((a) => a.isPostable);
    const creditAcc =
      accounts.find((a) => a.code === "1111") ??
      accounts.find((a) => a.isPostable && a.id !== debitAcc?.id);

    // Create a transaction
    const txRes = await testClient.authenticatedRequest<{
      data?: { id?: number };
    }>(ownerSession.token, `/workspaces/${ws.id}/fin/transactions`, {
      method: "POST",
      body: JSON.stringify({
        description: "Permission Sample Transaction",
        reference: "PERM-001",
      }),
    });
    this.testTxId = txRes.data?.id;

    // Create a draft journal entry
    const entryRes = await testClient.authenticatedRequest<{
      data?: { id?: number };
    }>(ownerSession.token, `/workspaces/${ws.id}/fin/entries`, {
      method: "POST",
      body: JSON.stringify({
        transactionId: this.testTxId,
        description: "Permission Sample Draft Entry",
        entryDate: new Date().toISOString(),
        postings: [
          { accountId: debitAcc?.id, amount: 50000, direction: "DEBIT" },
          { accountId: creditAcc?.id, amount: 50000, direction: "CREDIT" },
        ],
      }),
    });
    this.testEntryId = entryRes.data?.id;

    // Create and post a posted journal entry
    const postedEntryRes = await testClient.authenticatedRequest<{
      data?: { id?: number };
    }>(ownerSession.token, `/workspaces/${ws.id}/fin/entries`, {
      method: "POST",
      body: JSON.stringify({
        transactionId: this.testTxId,
        description: "Permission Sample Posted Entry",
        entryDate: new Date().toISOString(),
        postings: [
          { accountId: debitAcc?.id, amount: 75000, direction: "DEBIT" },
          { accountId: creditAcc?.id, amount: 75000, direction: "CREDIT" },
        ],
      }),
    });
    this.testPostedEntryId = postedEntryRes.data?.id;

    if (this.testPostedEntryId) {
      await testClient.authenticatedRequest(
        ownerSession.token,
        `/workspaces/${ws.id}/fin/entries/${this.testPostedEntryId}/post`,
        { method: "POST" },
      );
    }

    // 4. Add the authenticated test user as a member with the requested role
    await testClient.addMember({
      workspaceId: ws.id,
      userId: this.user.id,
      role: roleId,
    });

    // 5. Direct Playwright browser to use this workspace
    await this.context.addInitScript((wid) => {
      localStorage.setItem("omed.workspace", String(wid));
    }, ws.id);
  },
);

When(
  "I open the sample transaction detail page",
  async function (this: OmedWorld) {
    assert.ok(this.testTxId, "No test transaction ID available");
    await this.page.goto(this.web(`/fin/transactions/${this.testTxId}`));
  },
);

When(
  "I open the sample journal entry detail page",
  async function (this: OmedWorld) {
    assert.ok(this.testEntryId, "No test entry ID available");
    await this.page.goto(this.web(`/fin/journal/${this.testEntryId}`));
  },
);

When(
  "I open the sample posted journal entry detail page",
  async function (this: OmedWorld) {
    assert.ok(this.testPostedEntryId, "No test posted entry ID available");
    await this.page.goto(this.web(`/fin/journal/${this.testPostedEntryId}`));
  },
);

Then("the new transaction button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.transactions.expectNewTransactionHidden();
});

Then("the new transaction button is visible", async function (this: OmedWorld) {
  await this.pages.fin.transactions.expectNewTransactionVisible();
});

Then("the edit transaction button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.expectNoEditButton();
});

Then(
  "the edit transaction button is visible",
  async function (this: OmedWorld) {
    await this.pages.fin.transactionDetail.expectEditButtonVisible();
  },
);

Then(
  "the delete transaction button is hidden",
  async function (this: OmedWorld) {
    await this.pages.fin.transactionDetail.expectNoDeleteButton();
  },
);

Then(
  "the delete transaction button is visible",
  async function (this: OmedWorld) {
    await this.pages.fin.transactionDetail.expectDeleteButtonVisible();
  },
);

Then("the add entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.expectNoAddEntryButton();
});

Then("the add entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.transactionDetail.expectAddEntryButtonVisible();
});

Then("the new entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entries.expectNewEntryHidden();
});

Then("the new entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entries.expectNewEntryVisible();
});

Then("the post entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectNoPostButton();
});

Then("the post entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectPostButtonVisible();
});

Then("the edit entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectNoEditButton();
});

Then("the edit entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectEditButtonVisible();
});

Then("the delete entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectNoDeleteButton();
});

Then("the delete entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectDeleteButtonVisible();
});

Then("the void entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectNoVoidButton();
});

Then("the void entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectVoidButtonVisible();
});

Then("the reverse entry button is hidden", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectNoReverseButton();
});

Then("the reverse entry button is visible", async function (this: OmedWorld) {
  await this.pages.fin.entryDetail.expectReverseButtonVisible();
});
