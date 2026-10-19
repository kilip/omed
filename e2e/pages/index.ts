import type { Page } from "playwright";
import { HomePage } from "./core/home.page";
import { InvitationsPage } from "./core/invitations.page";
import { InvitePage } from "./core/invite.page";
import { LoginPage } from "./core/login.page";
import { MembersPage } from "./core/members.page";
import { WorkspacesPage } from "./core/workspaces.page";

import { FinAccountsPage } from "./fin/accounts.page";
import { FinDashboardPage } from "./fin/dashboard.page";
import { FinEntriesPage } from "./fin/entries.page";
import { FinEntryDetailPage } from "./fin/entry-detail.page";
import { FinLedgerPage } from "./fin/ledger.page";
import { FinSetupPage } from "./fin/setup.page";
import { FinTransactionDetailPage } from "./fin/transaction-detail.page";
import { FinTransactionsPage } from "./fin/transactions.page";

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
    workspaces: new WorkspacesPage(page),
  },
  fin: {
    dashboard: new FinDashboardPage(page),
    setup: new FinSetupPage(page),
    accounts: new FinAccountsPage(page),
    entries: new FinEntriesPage(page),
    entryDetail: new FinEntryDetailPage(page),
    ledger: new FinLedgerPage(page),
    transactions: new FinTransactionsPage(page),
    transactionDetail: new FinTransactionDetailPage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
