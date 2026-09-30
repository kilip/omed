import type { Page } from "playwright";
import { e2eConfig } from "../support/config";

export class BasePage {
  constructor(protected readonly page: Page) {}

  async navigate(path = "/") {
    const targetUrl = path.startsWith("http")
      ? path
      : `${e2eConfig.baseUrl}${path.startsWith("/") ? path : `/${path}`}`;
    await this.page.goto(targetUrl);
  }

  async getPathname(): Promise<string> {
    const url = new URL(this.page.url());
    return url.pathname;
  }

  async waitForUrl(urlOrPattern: string | RegExp) {
    await this.page.waitForURL(urlOrPattern);
  }
}
