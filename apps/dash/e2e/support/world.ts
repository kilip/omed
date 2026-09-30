import {
  type IWorldOptions,
  setWorldConstructor,
  World,
} from "@cucumber/cucumber";
import type { Browser, BrowserContext, Page } from "playwright";
import type { TestAuthUser } from "./auth-helper";

export class CustomWorld extends World {
  browser?: Browser;
  context?: BrowserContext;
  page?: Page;
  currentUser?: TestAuthUser;
  private cleanupTasks: (() => Promise<void>)[] = [];

  constructor(options: IWorldOptions) {
    super(options);
  }

  addCleanupTask(task: () => Promise<void>) {
    this.cleanupTasks.push(task);
  }

  async runCleanup() {
    while (this.cleanupTasks.length > 0) {
      const task = this.cleanupTasks.pop();
      if (task) {
        await task();
      }
    }
  }
}

setWorldConstructor(CustomWorld);
