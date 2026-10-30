import assert from "node:assert/strict";
import { Then, When } from "@cucumber/cucumber";
import type { OmedWorld } from "../../support/world";

When("I open {string}", async function (this: OmedWorld, path: string) {
  await this.page.goto(this.web(path));
});

When("I reload the page", async function (this: OmedWorld) {
  await this.page.reload();
});

Then("I am on {string}", async function (this: OmedWorld, path: string) {
  // Auth redirects happen client side, so wait for the final URL.
  const isMatch = (urlPath: string) => {
    if (urlPath === path || urlPath === `${path}/`) return true;
    if (path === "/" || path === "/fin") return false;
    return urlPath.startsWith(path.endsWith("/") ? path : `${path}/`);
  };

  await this.page.waitForURL((url) => isMatch(url.pathname), {
    timeout: 15_000,
  });
  const pathname = new URL(this.page.url()).pathname;
  assert.ok(
    isMatch(pathname),
    `Expected URL pathname to match "${path}", got "${pathname}"`,
  );
});
