# E2E coverage backlog

Inventory of web routes in `apps/web` that have no end-to-end coverage in `e2e/`.

Sources:

- Route definitions: `apps/web/app/routes.ts` and `apps/web/app/modules/*/route.ts`.
- Coverage: `e2e/features/**/*.feature` and the matching steps and page objects.

A route counts as **covered** when a scenario opens it or navigates to it through the UI and asserts on it. A route counts as **partial** when only part of its behavior is exercised, for example only a "button hidden" check, or only the sibling route that shares the same flow.

Follow the conventions in `AGENTS.md` section 5: Gherkin feature in `e2e/features/{module}/`, page object in `e2e/pages/{module}/`, steps in `e2e/steps/{module}/`. Do not put selectors in steps.

## Not covered

Routes with no E2E test.

- [x] 1. `/agent/settings` (`modules/agent/routes/settings.tsx`)
  - Covered by `e2e/features/agent/settings.feature`: enable with a selected provider and persist after reload, error when no provider exists, viewer sees read-only controls.
  - Page object: `e2e/pages/agent/settings.page.ts`. Steps: `e2e/steps/agent/settings.steps.ts`.
  - Verified against a live stack (`mise run e2e:local`).
  - Follow-up: analyze a document on the income and expense forms with the financial agent enabled. This needs a fake LLM that returns structured JSON.
- [x] 2. `/agent` (`modules/agent/routes/index.tsx`)
  - Covered by `e2e/features/agent/launcher.feature`: open `/agent` and navigate to `/agent/providers` and `/agent/settings` from the launcher tiles.
  - Page object: `e2e/pages/agent/launcher.page.ts`. Steps: `e2e/steps/agent/launcher.steps.ts`.
- [x] 3. `/fin/settings` (`modules/fin/routes/settings.tsx`)
  - Covered by `e2e/features/fin/settings.feature`: update base currency, decimal format, and FX account mappings and persist after reload, plus create a fiscal period from finance settings.
  - Page object: `e2e/pages/fin/settings.page.ts`. Steps: `e2e/steps/fin/settings.steps.ts`.
- [ ] 4. `/cms` (`modules/cms/routes/index.tsx`)
  - CMS launcher page. Every CMS feature opens `/cms/posts`, `/cms/pages`, or `/cms/media` directly.
  - Suggested scenario: open `/cms` and navigate to posts, pages, and media from the tiles.

## Partial coverage

Routes with some coverage that is missing important flows.

- [ ] 5. `/agent/providers/:id/edit` (`modules/agent/routes/provider-edit.tsx`)
  - `agent/permissions.feature` only checks that the edit button is hidden for a viewer.
  - Missing: an admin edits a provider (name, model, rotate API key) and sees the change in the table.
  - Add to `e2e/features/agent/providers.feature`.
- [ ] 6. `/core/members/:id` (`modules/core/routes/member-edit.tsx`)
  - `core/members.feature` covers `/core/members/new` and the members list only.
  - Missing: open an existing member, change roles, save, and see the updated roles in the list. Also member removal, if the page supports it.
- [ ] 7. `/fin/income/:id` and `/fin/income/:id/edit` (`income-detail.tsx`, `income-edit.tsx`)
  - `fin/income-expense.feature` creates and posts an income. Only the expense scenario opens the detail page and edits it.
  - Missing: open an income draft, edit it, post it. Add a scenario next to the expense one.
- [ ] 8. `/fin/expense/:id` (`expense-detail.tsx`)
  - Covered indirectly by the expense draft, edit, and post scenario. It has no assertions on the detail view itself, such as items, totals, and attachments.
- [ ] 9. `/fin/accounts` (`modules/fin/routes/accounts.tsx`)
  - Opened in `fin/ledger.feature` and `fin/setup.feature` for navigation and for checking seeded accounts.
  - Missing: create, edit, and deactivate an account.

## Covered (for reference)

| Route | Feature |
| --- | --- |
| `/login` | `core/otp-login.feature` |
| `/invite/:token` | `core/invitations.feature` |
| `/` (launcher) | `core/home.feature`, `core/preferences.feature` |
| `/core` | `core/workspaces.feature`, `core/members.feature` |
| `/core/workspaces/new` | `core/workspaces.feature` |
| `/core/workspaces/:id` | `core/workspaces.feature` |
| `/core/members`, `/core/members/new` | `core/members.feature` |
| `/core/invitations` | `core/invitations.feature` |
| `/core/access-tokens` | `core/access-tokens.feature` |
| `/agent` (launcher) | `agent/launcher.feature` |
| `/agent/providers`, `/agent/providers/new` | `agent/providers.feature`, `agent/permissions.feature` |
| `/agent/settings` | `agent/settings.feature` |
| `/cms/posts`, `/cms/posts/new`, `/cms/posts/:id` | `cms/posts.feature`, `cms/publishing.feature`, `cms/permissions.feature` |
| `/cms/pages`, `/cms/pages/new`, `/cms/pages/:id` | `cms/pages.feature`, `cms/permissions.feature` |
| `/cms/media` | `cms/media.feature`, `cms/permissions.feature` |
| `/fin` (launcher) | `fin/dashboard.feature`, `fin/setup.feature` |
| `/fin/onboarding` | `fin/setup.feature` |
| `/fin/overview` | `fin/dashboard.feature` |
| `/fin/income`, `/fin/income/new` | `fin/income-expense.feature` |
| `/fin/expense`, `/fin/expense/new`, `/fin/expense/:id/edit` | `fin/income-expense.feature` |
| `/fin/periods` | `fin/periods.feature` |
| `/fin/opening-balance` | `fin/opening-balance.feature` |
| `/fin/transactions`, `/new`, `/:id`, `/:id/edit` | `fin/transactions.feature`, `fin/permissions.feature` |
| `/fin/journal`, `/fin/journal/new`, `/fin/journal/:id` | `fin/entries.feature`, `fin/permissions.feature` |
| `/fin/ledger` | `fin/ledger.feature` |
| `/fin/trial-balance` | `fin/trial-balance.feature` |
| `/fin/settings` | `fin/settings.feature` |
