import type { Page } from "playwright";
import { HomePage } from "./core/home.page";
import { LoginPage } from "./core/login.page";

/**
 * Page objects grouped by module: `core`, `fin`, etc.
 */
export const createPages = (page: Page) => ({
  core: {
    login: new LoginPage(page),
    home: new HomePage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
