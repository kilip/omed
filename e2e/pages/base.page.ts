import type { Page } from "playwright";

/**
 * Base class for every page object. Page objects own the locators and the
 * `expect*` checks for one screen; steps only call them.
 */
export abstract class BasePage {
  /** Path of the page, relative to the dashboard URL. */
  abstract readonly path: string;

  constructor(protected readonly page: Page) {}
}
