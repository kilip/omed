import path from "node:path";
import { After, AfterAll, Before, BeforeAll, Status } from "@cucumber/cucumber";
import { type CoverageMapData, createCoverageMap } from "istanbul-lib-coverage";
import { createContext } from "istanbul-lib-report";
import reports from "istanbul-reports";
import { type Browser, chromium } from "playwright";
import { config, repoRoot } from "./config";
import { startServers, stopServers } from "./servers";
import { testClient } from "./test-client";
import type { OmedWorld } from "./world";

let browser: Browser;
const coverageMap = createCoverageMap();

BeforeAll({ timeout: config.startTimeoutMs * 2 }, async () => {
  await startServers();
  browser = await chromium.launch({ headless: !config.headed });
});

AfterAll({ timeout: 30_000 }, async () => {
  await browser?.close();
  await stopServers();

  if (coverageMap.files().length > 0) {
    const rootReportDir = path.join(repoRoot, "coverage");
    const e2eReportDir = path.join(repoRoot, "e2e", "reports", "coverage");

    const rootCtx = createContext({ dir: rootReportDir, coverageMap });
    reports.create("lcov").execute(rootCtx);
    reports.create("text-summary").execute(rootCtx);

    const e2eCtx = createContext({ dir: e2eReportDir, coverageMap });
    reports.create("html").execute(e2eCtx);
  }
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
  try {
    const coverage = await this.page.evaluate(() => {
      return (globalThis as unknown as { __coverage__?: CoverageMapData })
        .__coverage__;
    });
    if (coverage) {
      coverageMap.merge(coverage);
    }
  } catch {
    // Page may already be closed or redirected.
  }
  await this.context.close();
  if (this.user) {
    await testClient.deleteUser(this.user.id);
  }
});
