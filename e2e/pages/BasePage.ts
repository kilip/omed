import type { Page } from "playwright";

export abstract class BasePage {
  constructor(protected page: Page) {}

  async goto(path: string) {
    await this.page.goto(path);
  }

  async expectUrlPath(expectedPath: string, timeout = 60000) {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      const current = new URL(this.page.url()).pathname;
      if (current === expectedPath) {
        return;
      }
      await this.page.waitForTimeout(100);
    }
    throw new Error(
      `Timeout ${timeout}ms exceeded waiting for pathname "${expectedPath}". Current URL is "${this.page.url()}"`,
    );
  }

  async expectHeading(text: string) {
    const heading = this.page.getByRole("heading", { name: text });
    await heading.waitFor({ state: "visible" });
    return heading;
  }

  async expectButton(text: string) {
    const button = this.page.getByRole("button", { name: text });
    await button.waitFor({ state: "visible" });
    return button;
  }
}
