import {
  After,
  AfterAll,
  Before,
  BeforeAll,
  type IWorldOptions,
  setDefaultTimeout,
  setWorldConstructor,
  World,
} from "@cucumber/cucumber";
import {
  type Browser,
  type BrowserContext,
  chromium,
  type Page,
} from "playwright";
import { e2eEnv } from "../env";
import { startServers, stopServers } from "../servers";

setDefaultTimeout(60 * 1000);

let globalBrowser: Browser | null = null;

export class CustomWorld extends World {
  browser!: Browser;
  context!: BrowserContext;
  page!: Page;

  constructor(options: IWorldOptions) {
    super(options);
  }

  async signInUser(
    opts: { name?: string; locale?: "en" | "id"; currency?: string } = {},
  ) {
    const { auth } = await import("@omed/better-auth");
    const authContext = await auth.$context;
    const testUtils = authContext.test;

    const user = await testUtils.saveUser(
      testUtils.createUser({
        name: opts.name ?? "E2E Test User",
        email: `e2e+${crypto.randomUUID()}@omed.test`,
        locale: opts.locale ?? null,
        defaultCurrency: opts.currency ?? "IDR",
      }),
    );

    const cookies = await testUtils.getCookies({
      userId: user.id,
      domain: "localhost",
    });

    if (this.context) {
      await this.context.addCookies(cookies);
    }

    return {
      id: user.id,
      name: user.name,
      email: user.email,
    };
  }
}

setWorldConstructor(CustomWorld);

BeforeAll(async () => {
  await startServers();
  globalBrowser = await chromium.launch({
    headless: true,
  });
  const warmPage = await globalBrowser.newPage();
  try {
    await warmPage.goto(e2eEnv.DASH_URL);
    await warmPage.waitForURL("**/login", { timeout: 45000 });
  } catch {}
  await warmPage.close();
});

AfterAll(async () => {
  if (globalBrowser) {
    await globalBrowser.close();
    globalBrowser = null;
  }
  await stopServers();
});

Before(async function (this: CustomWorld) {
  if (!globalBrowser) {
    throw new Error("globalBrowser is not initialized");
  }
  this.browser = globalBrowser;
  this.context = await this.browser.newContext({
    baseURL: e2eEnv.DASH_URL,
    locale: "en-US",
  });
  this.page = await this.context.newPage();
});

After(async function (this: CustomWorld) {
  if (this.context) {
    await this.context.close();
  }
});
