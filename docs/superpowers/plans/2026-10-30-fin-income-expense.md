# Finance Income & Expense Quick Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add dedicated 1-to-1 Income (`/fin/income`, `/fin/income/new`) and Expense (`/fin/expense`, `/fin/expense/new`) quick-entry workflows backed by Go API endpoints that atomically create balanced `POSTED` double-entry transactions with optional attachments, relocate `/fin` dashboard to `/fin/overview`, and make `/fin` an Odoo-style module sub-launcher.

**Architecture:** The Go backend introduces `CashFlowRepository`, `CashFlowService`, and `CashFlowController` in `apps/api/internal/fin/` exposing `POST /workspaces/{wid}/fin/expense`, `POST /workspaces/{wid}/fin/expense/search`, `POST /workspaces/{wid}/fin/income`, and `POST /workspaces/{wid}/fin/income/search`. It reuses `fin_transaction`, `fin_entry`, `fin_posting`, and `core.files` with zero database schema migrations. The React Router frontend updates `apps/web/app/modules/fin/route.ts` and `module.ts`, moving the overview dashboard to `/fin/overview`, mounting a tile launcher at `/fin`, and adding list + creation routes for `/fin/income` and `/fin/expense`.

**Tech Stack:** Go 1.23+, Fiber v3, Ent ORM, `shopspring/decimal`, Swaggo OpenAPI, React Router v7, React 19, TypeScript, Ant Design 5, TanStack Query, Vitest, Playwright + Cucumber E2E.

**Spec:** `docs/superpowers/specs/2026-10-30-fin-cash-income-expense-design.md`

## Global Constraints

- Dependency direction in `apps/api/internal/fin/` is strictly `controller -> service -> repository`.
- Zero Ent schema changes or database migrations; reuse `fin_account`, `fin_transaction`, `fin_entry`, `fin_posting`, and `core.files`.
- Monetary amounts in DTOs use `decimal.Decimal` serialized as strings (`swaggertype:"string"`).
- Quick-entry transactions are created with `fin_entry.status = POSTED` and must respect `OPEN` fiscal periods, account active/postable checks, and multi-currency exchange rate resolution.
- Uploaded receipt files are linked to `fin_transaction` via `FileService.Update` (`EntityType: "fin:transaction"`, `EntityID: tx.ID`) so existing `ErrTransactionHasPostedEntries` invariants on `TransactionService.UploadAttachment` remain untouched.
- Regenerate OpenAPI (`bunx turbo run oa:gen` and `bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts`) after adding controller annotations.
- Formatting and linting must pass `gofmt`, `go vet ./...`, `bun run typecheck`, and `bun run lint`.
- Commit messages follow `<type>(<scope>): <emoji> <subject>` with allowed scopes (`api`, `web`, `e2e`, `docs`).

## Review Focus

- Creating an expense with a `REVENUE` or `LIABILITY` account as `expenseAccountId` (or non-`ASSET` as `paymentAccountId`) must fail with `422 Unprocessable Entity` (`ErrAccountInvalidType`).
- Creating an income or expense on a date that falls outside any `OPEN` period (or inside a `CLOSED` period) must fail with `404`/`409`/`422` (`ErrPeriodNotFound` or `ErrPeriodClosed`).
- Creating an income or expense using a parent/header account (`is_postable == false`) or inactive account (`is_active == false`) must fail with `422` (`ErrPostingAccountNotPostable` / `ErrPostingAccountInactive`).
- Attaching a `fileId` from another workspace or a non-existent file ID in `attachmentFileIds` must fail cleanly without leaving an orphaned unlinked transaction.
- Navigating to `/fin` must render the sub-launcher tiles and navigating to `/fin/overview` must render the financial overview dashboard without breaking existing E2E or unit tests.

---

### Task 1: Backend Domain, Models & Repository for Income and Expense (`apps/api`)

**Files:**
- Create: `apps/api/internal/fin/domain/cashflow.go`
- Create: `apps/api/internal/fin/model/cashflow.go`
- Create: `apps/api/internal/fin/repository/cashflow.go`
- Test: `apps/api/internal/fin/repository/cashflow_test.go`

**Interfaces:**
- Consumes: `ent.Client`, `domain.AccountTypeAsset`, `domain.AccountTypeExpense`, `domain.AccountTypeRevenue`, `domain.EntryStatusPosted`, `domain.PostingDirectionDebit`, `domain.PostingDirectionCredit`, `repository.GenerateTransactionNumber`, `repository.GenerateEntryNumber`
- Produces:
  - `domain.CashFlowType` (`"INCOME"`, `"EXPENSE"`)
  - `domain.CashFlowAccountSummary` (`ID int64`, `Code string`, `Name string`, `Type string`)
  - `domain.CashFlowItem` (`ID int64`, `WorkspaceID int64`, `Number string`, `TransactionDate time.Time`, `Type string`, `PaymentAccount domain.CashFlowAccountSummary`, `CategoryAccount domain.CashFlowAccountSummary`, `Amount decimal.Decimal`, `BaseAmount decimal.Decimal`, `Currency string`, `ExchangeRate decimal.Decimal`, `Description string`, `Reference string`, `EntryID int64`, `EntryNumber string`, `EntryStatus string`, `Attachments []coredomain.File`, `AuditMeta base.AuditMeta`)
  - `model.CreateExpenseRequest` (`PaymentAccountID int64`, `ExpenseAccountID int64`, `Amount decimal.Decimal`, `TransactionDate Date`, `Description string`, `Reference string`, `Currency string`, `ExchangeRate *decimal.Decimal`, `AttachmentFileIDs []int64`)
  - `model.CreateIncomeRequest` (`PaymentAccountID int64`, `RevenueAccountID int64`, `Amount decimal.Decimal`, `TransactionDate Date`, `Description string`, `Reference string`, `Currency string`, `ExchangeRate *decimal.Decimal`, `AttachmentFileIDs []int64`)
  - `model.SearchCashFlowRequest` (`DateFrom *Date`, `DateTo *Date`, `PaymentAccountID *int64`, `CategoryAccountID *int64`, `Query *string`, `Cursor *string`, `Limit *int`)
  - `model.CashFlowAccountSummaryResponse`, `model.CashFlowItemResponse`
  - `repository.CashFlowRepository`:
    - `NewCashFlowRepository(cli *ent.Client) CashFlowRepository`
    - `CreatePosted(c context.Context, params repository.CreateCashFlowParams) (domain.CashFlowItem, error)`
    - `Search(c context.Context, flowType string, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)`

- [ ] **Step 1: Write the failing repository unit tests in `apps/api/internal/fin/repository/cashflow_test.go`**

```go
func (s *CashFlowRepoTestSuite) TestCreatePostedExpenseAndSearch() {
	item, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
		FlowType:          domain.CashFlowTypeExpense,
		PeriodID:          s.period.ID,
		PaymentAccountID:  s.cashAcc.ID,
		CategoryAccountID: s.expenseAcc.ID,
		Amount:            decimal.NewFromInt(75000),
		BaseAmount:        decimal.NewFromInt(75000),
		TransactionDate:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		Description:       "Belanja di Indomaret",
		Reference:         "STRUK-001",
		Currency:          "IDR",
		ExchangeRate:      decimal.NewFromInt(1),
		PostedBy:          s.user.ID,
	})
	s.Require().NoError(err)
	s.Equal(domain.CashFlowTypeExpense, item.Type)
	s.Equal(s.cashAcc.ID, item.PaymentAccount.ID)
	s.Equal(s.expenseAcc.ID, item.CategoryAccount.ID)
	s.True(decimal.NewFromInt(75000).Equal(item.Amount))
	s.Equal(domain.EntryStatusPosted, item.EntryStatus)

	items, cursor, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(cursor)
	s.Len(items, 1)
	s.Equal(item.ID, items[0].ID)

	incItems, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(incItems)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/api && go test ./internal/fin/repository -run TestCashFlowRepoTestSuite -v`
Expected: FAIL (undefined types/methods)

- [ ] **Step 3: Implement `domain/cashflow.go`, `model/cashflow.go`, and `repository/cashflow.go`**

In `repository/cashflow.go`:
- `CreatePosted` runs inside `entx.WithTx` with retry on unique number conflict (matching `TransactionRepository.Create`), generates transaction number and entry number, creates `fin_transaction`, creates `fin_entry` with `Status = POSTED`, `PostedAt = now`, `PostedBy = params.PostedBy`, and inserts the 2 `fin_posting` rows:
  - For `EXPENSE`: position 0 is `DEBIT` `CategoryAccountID` (`expenseAccountId`), position 1 is `CREDIT` `PaymentAccountID`.
  - For `INCOME`: position 0 is `DEBIT` `PaymentAccountID`, position 1 is `CREDIT` `CategoryAccountID` (`revenueAccountId`).
- `Search` queries `Transaction` with `DeletedAtIsNil()`, filters by `DateFrom`, `DateTo`, `Query` (number/description/reference), `Cursor`, and `HasEntriesWith` where `entry.DeletedAtIsNil()`, `entry.StatusEQ(entry.StatusPOSTED)`, and matching `HasPostingsWith` for the payment account (`ASSET` + direction) and category account (`EXPENSE`/`REVENUE` + direction). Loads `WithEntries` -> `WithPostings` -> `WithAccount` and maps each transaction into `domain.CashFlowItem`.

- [ ] **Step 4: Run the repository tests to verify they pass**

Run: `cd apps/api && go test ./internal/fin/repository -run TestCashFlowRepoTestSuite -v`
Expected: PASS

---

### Task 2: Backend Service, Controller, Routes & OpenAPI Generation (`apps/api` & `packages/openapi`)

**Files:**
- Create: `apps/api/internal/fin/service/cashflow.go`
- Create: `apps/api/internal/fin/service/cashflow_test.go`
- Create: `apps/api/internal/fin/controller/cashflow.go`
- Modify: `apps/api/internal/fin/module.go`
- Modify: `apps/api/internal/fin/routes.go`
- Create: `apps/api/internal/fin/test/cashflow_test.go`
- Modify: `apps/api/docs/*` (via `oa:gen`)
- Modify: `packages/openapi/src/api.ts` (via `openapi-typescript`)

**Interfaces:**
- Consumes: `repository.CashFlowRepository`, `service.PeriodRepository`, `service.AccountRepository`, `service.ExchangeRateRepository`, `service.CurrencyRepository`, `service.FileServiceContract`
- Produces:
  - `service.CashFlowService`:
    - `NewCashFlowService(repo CashFlowRepositoryContract, periodRepo PeriodRepository, accountRepo AccountRepository, rateRepo ExchangeRateRepository, currencyRepo CurrencyRepository, fileSvc FileServiceContract) CashFlowService`
    - `CreateExpense(c context.Context, req model.CreateExpenseRequest) (domain.CashFlowItem, error)`
    - `CreateIncome(c context.Context, req model.CreateIncomeRequest) (domain.CashFlowItem, error)`
    - `SearchExpenses(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)`
    - `SearchIncomes(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)`
  - `controller.CashFlowController`:
    - `CreateExpense(c fiber.Ctx) error` (`POST /workspaces/{wid}/fin/expense`)
    - `SearchExpenses(c fiber.Ctx) error` (`POST /workspaces/{wid}/fin/expense/search`)
    - `CreateIncome(c fiber.Ctx) error` (`POST /workspaces/{wid}/fin/income`)
    - `SearchIncomes(c fiber.Ctx) error` (`POST /workspaces/{wid}/fin/income/search`)

- [ ] **Step 1: Write failing unit and integration tests in `service/cashflow_test.go` and `test/cashflow_test.go`**

Cover:
1. Valid `CreateExpense` creates a `POSTED` entry with Debit Expense and Credit Asset, links `attachmentFileIds` via `fileSvc.Update`, and appears in `SearchExpenses`.
2. Valid `CreateIncome` creates a `POSTED` entry with Debit Asset and Credit Revenue, and appears in `SearchIncomes`.
3. Passing a non-`ASSET` account as `paymentAccountId` or a non-`EXPENSE` account as `expenseAccountId` returns `domain.ErrAccountInvalidType` (HTTP 422).
4. Passing a non-postable (`is_postable == false`) or inactive (`is_active == false`) account returns `domain.ErrPostingAccountNotPostable` / `domain.ErrPostingAccountInactive` (HTTP 422).
5. Passing a `transactionDate` outside any open period returns `domain.ErrPeriodNotFound` or `domain.ErrPeriodClosed`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/api && go test ./internal/fin/service ./internal/fin/test -run "TestCashFlow" -v`
Expected: FAIL

- [ ] **Step 3: Implement `service/cashflow.go`, `controller/cashflow.go`, and wire into `module.go` and `routes.go`**

In `service/cashflow.go`:
- Validate `req.Amount.IsPositive()` (else `domain.ErrPostingInvalidAmount`).
- Resolve period via `periodRepo.FindByDate(c, req.TransactionDate.Time())`; if `errors.Is(err, base.ErrNotFound)` return `domain.ErrPeriodNotFound`; if `period.Status != domain.PeriodStatusOpen` return `domain.ErrPeriodClosed`.
- Fetch `paymentAcc` and `categoryAcc` via `accountRepo.GetByID`:
  - Check `IsActive` (`domain.ErrPostingAccountInactive`) and `IsPostable` (`domain.ErrPostingAccountNotPostable`).
  - Check `paymentAcc.Type == domain.AccountTypeAsset` and `categoryAcc.Type == expectedCategoryType` (else `domain.ErrAccountInvalidType`).
  - Check currency restriction on accounts if `acc.Currency != nil && *acc.Currency != ""` against transaction currency (`domain.ErrPostingCurrencyMismatch`).
- Validate any `AttachmentFileIDs` by calling `fileSvc.GetByID(c, fileID)` before creating the transaction so invalid file IDs fail fast.
- Call `repo.CreatePosted(c, params)` and then link each file via `fileSvc.Update(c, f.ID, coremodel.UpdateFileRequest{Name: f.Name, EntityType: domain.TransactionEntityType, EntityID: item.ID})`, attaching the resulting files to `item.Attachments`.
- On `SearchExpenses` and `SearchIncomes`, enrich returned items with attachments from `fileSvc.Search` where `EntityType == domain.TransactionEntityType`.

In `routes.go`:
```go
	cashFlowCtl := controller.NewCashFlowController(mod.CashFlowService)
	expense := r.Group("/workspaces/:wid/fin/expense")
	expense.Post("/", az.Require(domain.ObjTransaction, "create"), cashFlowCtl.CreateExpense)
	expense.Post("/search", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.SearchExpenses)

	income := r.Group("/workspaces/:wid/fin/income")
	income.Post("/", az.Require(domain.ObjTransaction, "create"), cashFlowCtl.CreateIncome)
	income.Post("/search", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.SearchIncomes)
```

- [ ] **Step 4: Regenerate OpenAPI spec and TypeScript types**

Run:
```bash
bunx turbo run oa:gen
bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts
```
Expected: `apps/api/docs/swagger.json` and `packages/openapi/src/api.ts` contain `/workspaces/{wid}/fin/expense`, `/workspaces/{wid}/fin/expense/search`, `/workspaces/{wid}/fin/income`, and `/workspaces/{wid}/fin/income/search`.

- [ ] **Step 5: Run all Go unit and integration tests**

Run: `cd apps/api && go test ./internal/fin/... -v`
Expected: PASS

---

### Task 3: Frontend Sub-Launcher (`/fin`), Overview Relocation (`/fin/overview`), Navbar & Queries (`apps/web`)

**Files:**
- Create: `apps/web/app/modules/fin/routes/launcher.tsx`
- Create: `apps/web/app/modules/fin/routes/launcher.test.tsx`
- Modify: `apps/web/app/modules/fin/route.ts`
- Modify: `apps/web/app/modules/fin/module.ts`
- Modify: `apps/web/app/modules/fin/queries.ts`
- Modify: `apps/web/app/i18n/en.ts`
- Modify: `apps/web/app/i18n/id.ts`
- Modify: `apps/web/app/routes/shell.test.tsx`

**Interfaces:**
- Consumes: `@omed/openapi` generated types (`model.CashFlowItemResponse`, `model.CreateExpenseRequest`, `model.CreateIncomeRequest`, `model.SearchCashFlowRequest`)
- Produces:
  - Routes in `apps/web/app/modules/fin/route.ts`:
    - `index("modules/fin/routes/launcher.tsx")` (`/fin`)
    - `route("overview", "modules/fin/routes/dashboard.tsx")` (`/fin/overview`)
    - `route("income", "modules/fin/routes/income.tsx")` (`/fin/income`)
    - `route("income/new", "modules/fin/routes/income-new.tsx")` (`/fin/income/new`)
    - `route("expense", "modules/fin/routes/expense.tsx")` (`/fin/expense`)
    - `route("expense/new", "modules/fin/routes/expense-new.tsx")` (`/fin/expense/new`)
  - Query helpers in `apps/web/app/modules/fin/queries.ts`:
    - `incomeKeys`, `expenseKeys`
    - `incomesInfiniteQueryOptions(wid: number, filters?: CashFlowFilters)`
    - `expensesInfiniteQueryOptions(wid: number, filters?: CashFlowFilters)`

- [ ] **Step 1: Write unit test for `FinLauncher` in `apps/web/app/modules/fin/routes/launcher.test.tsx`**

Verify that `/fin` renders tiles linking to `/fin/overview`, `/fin/income`, `/fin/expense`, `/fin/accounts`, `/fin/periods`, `/fin/opening-balance`, `/fin/transactions`, `/fin/journal`, `/fin/ledger`, and `/fin/trial-balance`.

- [ ] **Step 2: Run test to verify it fails**

Run: `bun --cwd apps/web run test app/modules/fin/routes/launcher.test.tsx`
Expected: FAIL (file does not exist)

- [ ] **Step 3: Implement `launcher.tsx`, update `module.ts`, `route.ts`, `queries.ts`, and `i18n/{en,id}.ts`**

In `launcher.tsx`:
- Match the visual layout of `apps/web/app/routes/launcher.tsx` (`home-backdrop`, `.app-tile`, `.app-tile--1..3`), rendering tiles for Overview, Income, Expense, and each item under Accounting.
In `module.ts`:
- Set top-level menu items:
  - `overview` (`path: "overview"`, `labelKey: "fin.menu.overview"`, `icon: DashboardOutlined`)
  - `income` (`path: "income"`, `labelKey: "fin.menu.income"`, `icon: ArrowDownOutlined`)
  - `expense` (`path: "expense"`, `labelKey: "fin.menu.expense"`, `icon: ArrowUpOutlined`)
  - `accounting` (`labelKey: "fin.menu.accounting"`, `icon: AuditOutlined`, `children`: `accounts`, `periods`, `opening-balance`, `transactions`, `journal`, `ledger`, `trial-balance`)

- [ ] **Step 4: Run web tests for launcher and shell**

Run: `bun --cwd apps/web run test app/modules/fin/routes/launcher.test.tsx app/routes/shell.test.tsx`
Expected: PASS

---

### Task 4: Frontend Income & Expense History and Quick Entry Forms (`apps/web`)

**Files:**
- Create: `apps/web/app/modules/fin/components/cashflow-helpers.ts`
- Create: `apps/web/app/modules/fin/routes/income.tsx`
- Create: `apps/web/app/modules/fin/routes/income-new.tsx`
- Create: `apps/web/app/modules/fin/routes/expense.tsx`
- Create: `apps/web/app/modules/fin/routes/expense-new.tsx`
- Create: `apps/web/app/modules/fin/routes/income-expense.test.tsx`
- Modify: `apps/web/app/modules/fin/components/index.ts`

**Interfaces:**
- Consumes: `accountsQueryOptions`, `currenciesQueryOptions`, `periodsQueryOptions`, `incomesInfiniteQueryOptions`, `expensesInfiniteQueryOptions`, `PendingAttachmentsPicker`, `FormView`, `Page`
- Produces:
  - `groupPaymentAccounts(accounts: AccountResponse[])`: filters active postable `ASSET` accounts and groups them by parent account (`1110 Kas & E-Wallet`, `1120 Bank Operasional`, `1130 Platform & Gateway`, plus any other postable liquid asset accounts excluding `114x` AR, `115x` Prepaid, `12xx` Fixed Assets unless fallback is needed when custom CoA has no `111x/112x/113x`).
  - `uploadWorkspaceFile(wid: number, file: File): Promise<number>`: uploads a file to `POST /workspaces/{wid}/files` and returns `res.id`.
  - Route components `IncomeListRoute`, `IncomeNewRoute`, `ExpenseListRoute`, `ExpenseNewRoute`.

- [ ] **Step 1: Write failing component and route tests in `apps/web/app/modules/fin/routes/income-expense.test.tsx`**

Test cases:
1. `groupPaymentAccounts` groups `1111 Kas Tunai` under `Kas & E-Wallet` and `1121 Bank BCA` under `Bank Operasional`, excluding `1141 Piutang Klien` and `1211 Laptop`.
2. `ExpenseNewRoute` renders payment account select, expense category select, amount, date, description, receipt picker, and live double-entry preview `(D) Expense / (K) Payment`, and submits to `POST /workspaces/{wid}/fin/expense`.
3. `IncomeNewRoute` renders deposit account select, revenue category select, amount, date, description, attachment picker, and live double-entry preview `(D) Payment / (K) Revenue`, and submits to `POST /workspaces/{wid}/fin/income`.
4. `ExpenseListRoute` and `IncomeListRoute` display history rows and navigate to `/fin/expense/new` and `/fin/income/new`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun --cwd apps/web run test app/modules/fin/routes/income-expense.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement `cashflow-helpers.ts`, `income.tsx`, `income-new.tsx`, `expense.tsx`, and `expense-new.tsx`**

- In `income-new.tsx` and `expense-new.tsx`:
  - Upload any `pendingFiles` via `POST /workspaces/{wid}/files` to collect `attachmentFileIds`.
  - Submit payload to `POST /workspaces/{wid}/fin/income` or `POST /workspaces/{wid}/fin/expense`.
  - Invalidate `incomeKeys.all(wid)` / `expenseKeys.all(wid)`, `transactionKeys.all(wid)`, `entryKeys.all(wid)`, and navigate back to `/fin/income` or `/fin/expense`.
- In `income.tsx` and `expense.tsx`:
  - Render filter bar (search input, payment account filter, category account filter, date range picker).
  - Render table with columns: Date, Number (link to `/fin/transactions/${record.id}`), Payment Account, Category Account, Description, Reference, Amount, Attachments count/preview.

- [ ] **Step 4: Run web unit tests, typecheck, and linter**

Run:
```bash
bun --cwd apps/web run test
bun run typecheck
bun run lint
```
Expected: PASS

---

### Task 5: E2E Tests & `/fin/overview` Route Updates (`e2e`)

**Files:**
- Modify: `e2e/pages/fin/dashboard.page.ts`
- Modify: `e2e/features/fin/dashboard.feature`
- Create: `e2e/pages/fin/income-expense.page.ts`
- Modify: `e2e/pages/index.ts`
- Create: `e2e/features/fin/income-expense.feature`
- Create: `e2e/steps/fin/income-expense.steps.ts`

**Interfaces:**
- Consumes: `BasePage`, `createPages` in `e2e/pages/index.ts`
- Produces: E2E coverage for `/fin` launcher, `/fin/overview` dashboard, `/fin/expense` + `/fin/expense/new`, and `/fin/income` + `/fin/income/new`.

- [ ] **Step 1: Update `e2e/pages/fin/dashboard.page.ts` and `e2e/features/fin/dashboard.feature` for `/fin/overview` and `/fin` launcher**

Check any E2E steps opening `/fin` expecting the dashboard directly vs clicking through the navbar/launcher, and update `dashboard.page.ts` path to `/fin/overview` where appropriate.

- [ ] **Step 2: Add E2E feature and page object for Income & Expense quick entry**

Create `e2e/features/fin/income-expense.feature` testing:
1. Recording an expense at Indomaret using Kas Tunai and seeing it in `/fin/expense`.
2. Recording an income from Company X into a Bank account and seeing it in `/fin/income`.

- [ ] **Step 3: Run typecheck and lint across the monorepo**

Run:
```bash
bun run typecheck
bun run lint
```
Expected: PASS with 0 errors.
