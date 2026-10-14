import type { Page } from "playwright";
import { HomePage } from "./web/home.page";
import { LoginPage } from "./web/login.page";

/**
 * Page objects grouped by dashboard module (`apps/web/app/modules/<module>`).
 * `web` holds the pages of the shell itself: login and the launcher.
 * Add `sys`, `user` and `fin` groups next to it as those modules get covered.
 */
export const createPages = (page: Page) => ({
  web: {
    login: new LoginPage(page),
    home: new HomePage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
