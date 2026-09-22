import * as fs from "node:fs";
import * as path from "node:path";
import "dotenv/config";
import {
  type IWorldOptions,
  setWorldConstructor,
  World,
} from "@cucumber/cucumber";
import {
  type Browser,
  type BrowserContext,
  chromium,
  type Page,
  type Response,
} from "@playwright/test";
import { auth } from "@/auth";
import { type OmedDatabase, serverDB } from "@/database";

export type AuthContext = Awaited<typeof auth.$context>;
export type AuthUtils = AuthContext["test"];

export interface TestContext {
  [key: string]: unknown;
  consoleErrors: string[];
  jsErrors: Error[];
  lastResponse?: Response | null;
  previousUrl?: string;
  userIds: string[];
}

let sharedBrowser: Browser | undefined;

/**
 * Default timeout for waiting operations (e.g., waitForURL, toBeVisible)
 */
export const WAIT_TIMEOUT = 13_000;

async function getSharedBrowser(): Promise<Browser> {
  if (!sharedBrowser) {
    sharedBrowser = await chromium.launch({
      headless: process.env.HEADLESS !== "false",
    });
  }

  return sharedBrowser;
}

export async function closeSharedBrowser(): Promise<void> {
  await sharedBrowser?.close();
  sharedBrowser = undefined;
}

export class CustomWorld extends World {
  browser!: Browser;
  browserContext!: BrowserContext;
  page!: Page;
  testContext!: TestContext;
  auth!: AuthUtils;
  db!: OmedDatabase;

  constructor(options: IWorldOptions) {
    super(options);
    this.testContext = {
      consoleErrors: [],
      jsErrors: [],
      userIds: [],
    };

    this.db = serverDB;
  }

  async init() {
    const ctx = await auth.$context;
    this.auth = ctx.test;

    const PORT = process.env.PORT ? Number(process.env.PORT) : 3000;
    const baseURL = process.env.BASE_URL || `http://localhost:${PORT}`;

    this.browser = await getSharedBrowser();
    this.browserContext = await this.browser.newContext({
      baseURL,
      viewport: { height: 720, width: 1200 },
    });
    this.browserContext.setDefaultTimeout(30_000);
    this.page = await this.browserContext.newPage();

    // Set up error listeners
    this.page.on("pageerror", (error) => {
      this.testContext.jsErrors.push(error);
      console.error("Page error:", error.message);
    });

    this.page.on("console", (msg) => {
      if (msg.type() === "error") {
        this.testContext.consoleErrors.push(msg.text());
      }
    });

    this.page.setDefaultTimeout(30_000);
  }

  get context(): TestContext {
    return this.testContext;
  }

  async cleanup() {
    await this.page?.close();
    await this.browserContext?.close();
  }

  async takeScreenshot(name: string): Promise<Buffer> {
    const screenshot = await this.page.screenshot({ fullPage: true });

    // Save screenshot to file
    const screenshotsDir = path.join(process.cwd(), "test-reports/screenshots");
    if (!fs.existsSync(screenshotsDir)) {
      fs.mkdirSync(screenshotsDir, { recursive: true });
    }
    const filepath = path.join(screenshotsDir, `${name}.png`);
    fs.writeFileSync(filepath, screenshot);
    console.log(`📸 Screenshot saved: ${filepath}`);

    return screenshot;
  }
}

setWorldConstructor(CustomWorld);
