import type { Page } from "playwright";
import { HomePage } from "./dashboard/home.page";
import { LoginPage } from "./dashboard/login.page";

/**
 * Page objects grouped by area. `dashboard` is the shell itself (login and the
 * launcher, `apps/web/app/routes`), which is not a module. Business modules
 * (`apps/web/app/modules/<module>`) get a group of their own: add `sys`,
 * `user`, `fin` or `cms` next to it as they get covered.
 */
export const createPages = (page: Page) => ({
  dashboard: {
    login: new LoginPage(page),
    home: new HomePage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
