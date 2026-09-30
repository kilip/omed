import {
  After,
  AfterAll,
  Before,
  BeforeAll,
  Status,
  setDefaultTimeout,
} from "@cucumber/cucumber";
import { type Browser, chromium } from "playwright";
import { e2eConfig } from "./config";
import type { CustomWorld } from "./world";

setDefaultTimeout(60 * 1000);

let globalBrowser: Browser | undefined;

BeforeAll(async () => {
  globalBrowser = await chromium.launch({
    headless: e2eConfig.browser.headless,
    slowMo: e2eConfig.browser.slowMo,
  });
});

AfterAll(async () => {
  if (globalBrowser) {
    await globalBrowser.close();
  }
});

Before(async function (this: CustomWorld) {
  if (!globalBrowser) {
    globalBrowser = await chromium.launch({
      headless: e2eConfig.browser.headless,
      slowMo: e2eConfig.browser.slowMo,
    });
  }

  this.browser = globalBrowser;
  this.context = await this.browser.newContext({
    viewport: e2eConfig.viewport,
  });
  this.page = await this.context.newPage();
  this.page.setDefaultTimeout(e2eConfig.browser.timeout);
});

After(async function (this: CustomWorld, scenario) {
  if (scenario.result?.status === Status.FAILED && this.page) {
    const screenshot = await this.page.screenshot({
      fullPage: true,
    });
    this.attach(screenshot, "image/png");
  }

  // Run all registered user/data cleanup tasks
  await this.runCleanup();

  if (this.page) {
    await this.page.close();
  }
  if (this.context) {
    await this.context.close();
  }
});
