import { After, AfterAll, Before, BeforeAll, Status } from "@cucumber/cucumber";
import { type Browser, chromium } from "playwright";
import { getTestHelpers } from "./auth";
import { config } from "./config";
import { startServers, stopServers } from "./servers";
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

/** Signs in through Better Auth `testUtils`: real user, real session, signed cookie. */
Before({ tags: "@authenticated" }, async function (this: OmedWorld) {
  const test = await getTestHelpers();
  const user = await test.saveUser(
    test.createUser({
      name: "E2E Tester",
      email: `e2e+${crypto.randomUUID()}@omed.test`,
    }),
  );
  const { cookies } = await test.login({ userId: user.id });
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
    const test = await getTestHelpers();
    await test.deleteUser(this.user.id);
  }
});
