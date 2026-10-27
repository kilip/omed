import {
  type IWorldOptions,
  setDefaultTimeout,
  setWorldConstructor,
  World,
} from "@cucumber/cucumber";
import type { BrowserContext, Page } from "playwright";
import { createPages, type Pages } from "../pages";
import { config } from "./config";
import { type TestUser, type TestWorkspace, testClient } from "./test-client";

setDefaultTimeout(30_000);

export class OmedWorld extends World {
  context!: BrowserContext;
  page!: Page;
  pages!: Pages;
  /** Set by the `@authenticated` hook. */
  user?: TestUser;
  sessionToken?: string;
  workspace?: TestWorkspace;
  recipientUser?: TestUser;
  invitationLink?: string;
  testTxId?: number;
  testEntryId?: number;
  testPostedEntryId?: number;
  publicApiResponse?: { status: number; data?: unknown };
  publishedPostTitle?: string;
  draftPostTitle?: string;
  scheduledPostTitle?: string;
  consoleLogs: string[] = [];

  get client() {
    return testClient;
  }

  constructor(options: IWorldOptions) {
    super(options);
  }

  /** Called by the `Before` hook once the browser context exists. */
  init(context: BrowserContext, page: Page): void {
    this.context = context;
    this.page = page;
    this.pages = createPages(page);
    this.consoleLogs = [];
  }

  /** Absolute dashboard URL for a path such as `/login`. */
  web(path: string): string {
    return new URL(path, config.webUrl).toString();
  }
}

setWorldConstructor(OmedWorld);
