import assert from "node:assert/strict";
import { Given, Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

// --- Posts Steps ---

When("I click the new post button", async function (this: OmedWorld) {
  await this.pages.cms.posts.clickNewPost();
});

When(
  "I fill in the post with title {string} and body {string}",
  async function (this: OmedWorld, title: string, body: string) {
    await this.pages.cms.posts.fillPost({ title, body });
  },
);

When("I save the post", async function (this: OmedWorld) {
  await this.pages.cms.posts.savePost();
});

Then(
  "I see the post status {string}",
  async function (this: OmedWorld, status: string) {
    await this.pages.cms.posts.expectStatus(status);
  },
);

Then(
  "I see {string} in the posts table",
  async function (this: OmedWorld, title: string) {
    await this.pages.cms.posts.expectPostInTable(title);
  },
);

Then(
  "I do not see {string} in the posts table",
  async function (this: OmedWorld, title: string) {
    await this.pages.cms.posts.expectPostNotInTable(title);
  },
);

When("I open post {string}", async function (this: OmedWorld, title: string) {
  await this.pages.cms.posts.openPost(title);
});

When("I delete the post", async function (this: OmedWorld) {
  await this.pages.cms.posts.deletePost();
});

Then("the new post button is hidden", async function (this: OmedWorld) {
  await this.pages.cms.posts.expectNewButtonHidden();
});

Then("the new post button is visible", async function (this: OmedWorld) {
  await this.pages.cms.posts.expectNewButtonVisible();
});

// --- Pages Steps ---

When("I click the new page button", async function (this: OmedWorld) {
  await this.pages.cms.pages.clickNewPage();
});

When(
  "I fill in the page with title {string} and body {string}",
  async function (this: OmedWorld, title: string, body: string) {
    await this.pages.cms.pages.fillPage({ title, body });
  },
);

When("I save the page", async function (this: OmedWorld) {
  await this.pages.cms.pages.savePage();
});

Then(
  "I see the page status {string}",
  async function (this: OmedWorld, status: string) {
    await this.pages.cms.pages.expectStatus(status);
  },
);

Then(
  "I see {string} in the pages table",
  async function (this: OmedWorld, title: string) {
    await this.pages.cms.pages.expectPageInTable(title);
  },
);

Then(
  "I do not see {string} in the pages table",
  async function (this: OmedWorld, title: string) {
    await this.pages.cms.pages.expectPageNotInTable(title);
  },
);

When("I open page {string}", async function (this: OmedWorld, title: string) {
  await this.pages.cms.pages.openPage(title);
});

When("I delete the page", async function (this: OmedWorld) {
  await this.pages.cms.pages.deletePage();
});

Then("the new page button is hidden", async function (this: OmedWorld) {
  await this.pages.cms.pages.expectNewButtonHidden();
});

Then("the new page button is visible", async function (this: OmedWorld) {
  await this.pages.cms.pages.expectNewButtonVisible();
});

// --- Media Steps ---

When(
  "I upload a media file named {string}",
  async function (this: OmedWorld, filename: string) {
    await this.pages.cms.media.uploadMedia(filename);
  },
);

Then(
  "I see {string} in the media library",
  async function (this: OmedWorld, filename: string) {
    await this.pages.cms.media.expectMediaVisible(filename);
  },
);

When(
  "I edit the alt text of {string} to {string}",
  async function (this: OmedWorld, filename: string, altText: string) {
    await this.pages.cms.media.openEditDrawer(filename);
    await this.pages.cms.media.editAltText(altText);
  },
);

When("I select the uploaded cover image", async function (this: OmedWorld) {
  await this.pages.cms.posts.selectCoverImage();
});

When(
  "I attempt to delete the media {string}",
  async function (this: OmedWorld, filename: string) {
    await this.pages.cms.media.deleteMedia(filename);
  },
);

Then(
  "the media delete is blocked with a message that it is in use",
  async function (this: OmedWorld) {
    await this.pages.cms.media.expectDeleteBlocked();
  },
);

Then("the upload media button is hidden", async function (this: OmedWorld) {
  await this.pages.cms.media.expectUploadButtonHidden();
});

Then("the upload media button is visible", async function (this: OmedWorld) {
  await this.pages.cms.media.expectUploadButtonVisible();
});

// --- Publishing Steps ---

When("I publish the post", async function (this: OmedWorld) {
  await this.pages.cms.posts.publishPost();
});

When("I unpublish the post", async function (this: OmedWorld) {
  await this.pages.cms.posts.unpublishPost();
});

When("I schedule the post for publication", async function (this: OmedWorld) {
  await this.pages.cms.posts.schedulePost();
});

// --- Permissions Steps ---

Given(
  "a workspace exists where I have role {string}",
  async function (this: OmedWorld, roleId: string) {
    assert.ok(this.user, "Tag the scenario with @authenticated");

    const owner = await this.client.createUser({
      name: "CMS Workspace Owner",
      email: `owner+${crypto.randomUUID()}@omed.test`,
      emailVerified: true,
    });
    this.recipientUser = owner;

    const ws = await this.client.createWorkspace({
      name: "CMS Permissions Test",
      slug: `cms-perm-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      ownerId: owner.id,
      personal: false,
    });
    this.workspace = ws;

    await this.client.addMember({
      workspaceId: ws.id,
      userId: this.user.id,
      role: roleId,
    });

    await this.context.addInitScript((wid) => {
      localStorage.setItem("omed.workspace", String(wid));
    }, ws.id);
  },
);

// --- Public API Steps ---

Given(
  "a workspace exists with a published post, a draft post, and a future-scheduled post",
  async function (this: OmedWorld) {
    assert.ok(this.user, "Tag the scenario with @authenticated");
    assert.ok(this.sessionToken, "Expected sessionToken");

    const ws = await this.client.createWorkspace({
      name: "Public API Workspace",
      slug: `public-ws-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      ownerId: this.user.id,
      personal: true,
    });
    this.workspace = ws;
    const wid = ws.id;

    this.publishedPostTitle = "Public Live Post";
    this.draftPostTitle = "Secret Draft Post";
    this.scheduledPostTitle = "Future Scheduled Post";

    // 1. Create and publish post
    const pubRes = await this.client.authenticatedRequest<{
      data?: { id: number };
    }>(this.sessionToken, `/workspaces/${wid}/cms/posts`, {
      method: "POST",
      body: JSON.stringify({
        title: this.publishedPostTitle,
        slug: `public-live-${Date.now()}`,
        body: "# Public Post Content",
      }),
    });
    const pubId = pubRes.data?.id;
    assert.ok(pubId, "Expected created post id");

    await this.client.authenticatedRequest(
      this.sessionToken,
      `/workspaces/${wid}/cms/posts/${pubId}/publish`,
      { method: "POST" },
    );

    // 2. Create draft post
    await this.client.authenticatedRequest(
      this.sessionToken,
      `/workspaces/${wid}/cms/posts`,
      {
        method: "POST",
        body: JSON.stringify({
          title: this.draftPostTitle,
          slug: `secret-draft-${Date.now()}`,
          body: "# Secret Draft Content",
        }),
      },
    );

    // 3. Create future scheduled post
    const schedRes = await this.client.authenticatedRequest<{
      data?: { id: number };
    }>(this.sessionToken, `/workspaces/${wid}/cms/posts`, {
      method: "POST",
      body: JSON.stringify({
        title: this.scheduledPostTitle,
        slug: `future-sched-${Date.now()}`,
        body: "# Future Scheduled Content",
      }),
    });
    const schedId = schedRes.data?.id;
    assert.ok(schedId, "Expected scheduled post id");

    await this.client.authenticatedRequest(
      this.sessionToken,
      `/workspaces/${wid}/cms/posts/${schedId}/schedule`,
      {
        method: "POST",
        body: JSON.stringify({
          publishAt: new Date(Date.now() + 86400000).toISOString(),
        }),
      },
    );
  },
);

When(
  "an unauthenticated request is made for the public posts",
  async function (this: OmedWorld) {
    assert.ok(this.workspace, "Expected workspace");
    const wid = this.workspace.id;

    const res = await this.client.publicGet<{
      data?: Array<{ title?: string; slug?: string }>;
    }>(`/public/workspaces/${wid}/cms/posts`);

    this.publicApiResponse = res;
  },
);

Then(
  "the public response contains the published post",
  async function (this: OmedWorld) {
    assert.ok(this.publicApiResponse, "Expected publicApiResponse");
    assert.equal(
      this.publicApiResponse.status,
      200,
      `Expected status 200, got ${this.publicApiResponse.status}`,
    );

    const data = this.publicApiResponse.data as {
      data?: Array<{ title?: string }>;
    };
    const items = data?.data ?? [];
    const titles = items.map((i) => i.title);

    assert.ok(
      titles.includes(this.publishedPostTitle),
      `Expected published post "${this.publishedPostTitle}" in public list: ${JSON.stringify(titles)}`,
    );
  },
);

Then(
  "the public response does not contain the draft post",
  async function (this: OmedWorld) {
    assert.ok(this.publicApiResponse, "Expected publicApiResponse");
    const data = this.publicApiResponse.data as {
      data?: Array<{ title?: string }>;
    };
    const items = data?.data ?? [];
    const titles = items.map((i) => i.title);

    assert.ok(
      !titles.includes(this.draftPostTitle),
      `Did not expect draft post "${this.draftPostTitle}" in public list: ${JSON.stringify(titles)}`,
    );
  },
);

Then(
  "the public response does not contain the future-scheduled post",
  async function (this: OmedWorld) {
    assert.ok(this.publicApiResponse, "Expected publicApiResponse");
    const data = this.publicApiResponse.data as {
      data?: Array<{ title?: string }>;
    };
    const items = data?.data ?? [];
    const titles = items.map((i) => i.title);

    assert.ok(
      !titles.includes(this.scheduledPostTitle),
      `Did not expect future-scheduled post "${this.scheduledPostTitle}" in public list: ${JSON.stringify(titles)}`,
    );
  },
);
