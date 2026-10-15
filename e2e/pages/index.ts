import type { Page } from "playwright";
import { HomePage } from "./core/home.page";
import { InvitationsPage } from "./core/invitations.page";
import { InvitePage } from "./core/invite.page";
import { LoginPage } from "./core/login.page";
import { MembersPage } from "./core/members.page";

/**
 * Page objects grouped by module: `core`, `fin`, etc.
 */
export const createPages = (page: Page) => ({
  core: {
    login: new LoginPage(page),
    home: new HomePage(page),
    invitations: new InvitationsPage(page),
    invite: new InvitePage(page),
    members: new MembersPage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
