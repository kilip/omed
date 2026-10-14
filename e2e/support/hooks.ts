import { After, AfterAll, Before, BeforeAll, Status } from "@cucumber/cucumber";
import { type Browser, chromium } from "playwright";
import { config } from "./config";
import { startServers, stopServers } from "./servers";
import { testClient } from "./test-client";
import type { OmedWorld } from "./world";

let browser: Browser;

BeforeAll({ timeout: config.startTimeoutMs * 2 }, async () => {
  await startServers();
  browser = await chromium.launch({ headless: !config.headed });
});

AfterAll({ timeout: 30_000 }, async () => {
  await browser?.close();
  await stopServers();
});

Before(async function (this: OmedWorld) {
  const context = await browser.newContext({
    baseURL: config.webUrl,
    // The dashboard picks its language from the browser: pin it so every
    // assertion can use the English copy.
    locale: "en-US",
    viewport: { width: 1280, height: 800 },
  });
  this.init(context, await context.newPage());
});

/** Signs in through testClient: real user, real session, cookies for dashboard. */
Before({ tags: "@authenticated" }, async function (this: OmedWorld) {
  const { user, cookies } = await testClient.setupAuthenticatedUser();
  await this.context.addCookies(cookies);
  this.user = user;
});

After(async function (this: OmedWorld, scenario) {
  if (scenario.result?.status === Status.FAILED) {
    const png = await this.page.screenshot({ fullPage: true });
    this.attach(png, "image/png");
  }
  await this.context.close();
  if (this.user) {
    await testClient.deleteUser(this.user.id);
  }
});
