import assert from "node:assert/strict";
import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I open {string}", async function (this: OmedWorld, path: string) {
  await this.page.goto(this.web(path));
});

Then("I am on {string}", async function (this: OmedWorld, path: string) {
  // Auth redirects happen client side, so wait for the final URL.
  await this.page.waitForURL(
    (url) =>
      url.pathname === path ||
      url.pathname.startsWith(path.endsWith("/") ? path : `${path}/`),
    {
      timeout: 15_000,
    },
  );
  const pathname = new URL(this.page.url()).pathname;
  assert.ok(
    pathname === path ||
      pathname.startsWith(path.endsWith("/") ? path : `${path}/`),
    `Expected URL pathname to be or start with "${path}", got "${pathname}"`,
  );
});
