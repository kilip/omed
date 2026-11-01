import type { Page } from "playwright";
import { AgentProvidersPage } from "./agent/providers.page";
import { CmsMediaPage } from "./cms/media.page";
import { CmsPagesPage } from "./cms/pages.page";
import { CmsPostsPage } from "./cms/posts.page";
import { AccessTokensPage } from "./core/access-tokens.page";
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
import { FinIncomeExpensePage } from "./fin/income-expense.page";
import { FinLedgerPage } from "./fin/ledger.page";
import { FinOpeningBalancePage } from "./fin/opening-balance.page";
import { FinPeriodsPage } from "./fin/periods.page";
import { FinSetupPage } from "./fin/setup.page";
import { FinTransactionDetailPage } from "./fin/transaction-detail.page";
import { FinTransactionsPage } from "./fin/transactions.page";
import { FinTrialBalancePage } from "./fin/trial-balance.page";

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
    accessTokens: new AccessTokensPage(page),
  },
  fin: {
    dashboard: new FinDashboardPage(page),
    setup: new FinSetupPage(page),
    accounts: new FinAccountsPage(page),
    periods: new FinPeriodsPage(page),
    openingBalance: new FinOpeningBalancePage(page),
    incomeExpense: new FinIncomeExpensePage(page),
    entries: new FinEntriesPage(page),
    entryDetail: new FinEntryDetailPage(page),
    ledger: new FinLedgerPage(page),
    transactions: new FinTransactionsPage(page),
    transactionDetail: new FinTransactionDetailPage(page),
    trialBalance: new FinTrialBalancePage(page),
  },
  cms: {
    posts: new CmsPostsPage(page),
    pages: new CmsPagesPage(page),
    media: new CmsMediaPage(page),
  },
  agent: {
    providers: new AgentProvidersPage(page),
  },
});

export type Pages = ReturnType<typeof createPages>;
