# Multi-Item Income & Expense with Posting / Draft Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let income and expense documents carry multiple items (account, description, quantity, unit price), be saved as `DRAFT` or `POSTED`, and have drafts edited later, with a wider money input.

**Architecture:** Items are `fin_posting` rows that already have `quantity` and `unit_price`; the header is `fin_transaction` (`reference` holds the invoice number) and one `fin_entry` whose status is `DRAFT` or `POSTED`. The existing cashflow controller, service, and repository are reworked (create, read, replace-draft, search with status). The web app replaces its two near-identical forms with one shared `CashflowForm`.

**Tech Stack:** Go (Fiber v3, Ent, `shopspring/decimal`, testify suites, Swaggo), React 19 + React Router (SPA), Ant Design, `openapi-fetch`, Vitest, Cucumber + Playwright, Biome.

**Spec:** `docs/superpowers/specs/2026-10-31-fin-cashflow-items-design.md`. It supersedes the 1-to-1 parts of `docs/superpowers/plans/2026-10-30-fin-income-expense.md` and the single-category contract of its spec.

## Global Constraints

- Zero schema changes: no Ent regeneration, no migration. `fin_posting.quantity`, `fin_posting.unit_price` (`numeric(20,4)`, nullable) and `fin_transaction.reference` already exist.
- Layering: `controller -> service -> repository`. Controllers never touch Ent. Only repositories talk to the database.
- Money: `shopspring/decimal` in Go, decimal strings on the wire. No floating point on amounts in the backend.
- Items: 1 to 100 per document. `quantity` > 0, `unitPrice` not negative, line amount `quantity × unitPrice` > 0, rounded to the document currency decimals (capped at 4, fallback 4).
- `status` is `DRAFT` or `POSTED`, default `POSTED`. Editing (`PUT`) is allowed only while the entry is `DRAFT` (`ErrEntryNotDraft`, 409).
- Frontend: types from `@omed/openapi` only (no hand-written response types); design tokens from `BRANDING.md`, no hardcoded hex; Biome; tests next to the file under test; i18n keys in both `i18n/en.ts` and `i18n/id.ts`; no unit-of-measure field; no `500k`/`1jt` parsing.
- Money inputs show at least 9 integer digits with grouping and the currency decimals (e.g. `999.999.999`).
- Commits: Conventional Commits with emoji (`feat(api): ✨ ...`, `test(...)`: ✅, `docs`: 📝), required scope (`api`, `web`, `e2e`, `docs`), one logical change per commit, lowercase subject, and the attribution trailers the executing session requires.
- Tasks 1 to 3 change one Go contract. Each task verifies its own package (`go test ./internal/fin/<pkg>/`); `go build ./...` is expected green only after Task 3. Web tasks need the regenerated types from Task 3.

## Review Focus

- RF1: the same account on several items (the owner's expense uses `6710` twice) must appear once in search results, keep item order, and sum correctly. Pinned in Tasks 1 and 2.
- RF2: fractional quantity times price in a zero-decimal currency (`2.5 × 3333` in IDR) rounds to the currency decimals, not to 4. Pinned in Task 2.
- RF3: a foreign-currency document with a non-trivial rate stays balanced in the base currency (payment `base_amount` equals the sum of item `base_amount`s). Pinned in Tasks 1 and 2.
- RF4: a draft whose period was closed after saving, or whose edited date falls in another period, must fail posting cleanly and stay `DRAFT`. Pinned in Task 2.
- RF5: double-clicking Posting or Draft must not create two documents; both buttons are disabled while a request is pending. Pinned in Task 5.

---

### Task 1: Domain, models, and repository

**Files:**
- Modify: `apps/api/internal/fin/domain/cashflow.go`, `apps/api/internal/fin/model/cashflow.go`, `apps/api/internal/fin/repository/cashflow.go`
- Test: `apps/api/internal/fin/repository/cashflow_test.go` (suite `CashFlowRepoTestSuite`)

**Interfaces:**
- Produces (domain): `type CashFlowLine struct { ID int64; Account CashFlowAccountSummary; Description string; Quantity, UnitPrice, Amount, BaseAmount decimal.Decimal }`; `CashFlowItem` loses `CategoryAccount` and gains `Items []CashFlowLine`.
- Produces (model): `type CashFlowItemRequest struct { AccountID int64; Description string; Quantity, UnitPrice decimal.Decimal }`; `type SaveCashFlowRequest struct { PaymentAccountID int64; TransactionDate Date; Reference, Description, Currency, Status string; ExchangeRate *decimal.Decimal; Items []CashFlowItemRequest; AttachmentFileIDs []int64 }` (JSON names per the spec §5.2; `items` is `validate:"required,min=1,max=100,dive"`, `status` is `omitempty,oneof=DRAFT POSTED`) replacing `CreateIncomeRequest` and `CreateExpenseRequest`; `SearchCashFlowRequest` gains `Status *string` (`omitempty,oneof=DRAFT POSTED`); `CashFlowItemResponse` loses `CategoryAccount` and gains `Items []CashFlowLineResponse`, where `type CashFlowLineResponse struct { ID int64; Account CashFlowAccountSummaryResponse; Description string; Quantity, UnitPrice, Amount, BaseAmount decimal.Decimal }` (decimals as swagger strings).
- Produces (repository): `type CashFlowItemParams struct { AccountID int64; Description string; Quantity, UnitPrice, Amount, BaseAmount decimal.Decimal }`; `CreateCashFlowParams` replaces `CategoryAccountID`/`Amount`/`BaseAmount` with `Items []CashFlowItemParams`, `TotalAmount`, `TotalBaseAmount`, `Status string`; methods `Create(c, CreateCashFlowParams) (domain.CashFlowItem, error)` (renamed from `CreatePosted`), `Update(c, id int64, CreateCashFlowParams) (domain.CashFlowItem, error)`, `GetByID(c, flowType string, id int64) (domain.CashFlowItem, error)`; `Search` keeps its signature.

- [ ] **Step 1: Write failing tests** in `cashflow_test.go`, each building params with the owner's data (income: payment `Bank Kaltim`, items `4130` 1 × 1,000,000 and `4110` 6 × 500,000):
  - `Test_Create_MultiItemPosted_PostingsAndPositions`: payment posting `position 0`, `DEBIT`, amount `4000000`, quantity/unit price nil; items at positions 1 and 2, `CREDIT`, amounts `1000000` and `3000000`, quantities `1` and `6`, unit prices `1000000` and `500000`; entry `POSTED` with `posted_at` set.
  - `Test_Create_Draft_HasNoPostedAt`: entry `DRAFT`, `posted_at` and `posted_by` nil.
  - `Test_Create_FXBaseAmountsBalance` (RF3): USD document, rate `16000.5`, three items with fractional amounts; sum of item `base_amount` equals the payment posting `base_amount`.
  - `Test_Search_IncludesDraftAndFiltersByStatus`: one draft and one posted document; nil status returns both, `DRAFT` returns one, a `VOID` entry is never returned.
  - `Test_Search_SameAccountOnTwoItems_ReturnsDocumentOnce` (RF1): expense with `6710` twice (2 × 55,000 and 5 × 3,000), filter by that `categoryAccountId`; exactly one result, items in position order, total `125000`.
  - `Test_Update_ReplacesPostings_PreservesNumbers`: after `Update` the transaction and entry numbers are unchanged and postings are exactly the new set at positions `0..n`.
  - `Test_Update_PostedEntry_ReturnsErrEntryNotDraft`.
  - `Test_GetByID_WrongFlowType_ReturnsErrTransactionNotFound`.
  - `Test_GetByID_LegacyPostingWithoutQuantity`: a posting with nil quantity/unit price reads back as quantity `1`, unit price equal to its amount.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/repository/ -run CashFlow -v`. Expected: FAIL (does not compile: new types and methods are missing).
- [ ] **Step 3: Implement** domain and model types above, then the repository: `Create` writes transaction, entry (status per params; set `posted_at`/`posted_by` only for `POSTED`), payment posting at position 0 (`DEBIT` income, `CREDIT` expense, amount `TotalAmount`, base `TotalBaseAmount`), and one posting per item at positions `1..n` with `SetQuantity`/`SetUnitPrice`, all in one `entx.WithTx` keeping the existing number-collision retry. `Update` runs in one `entx.WithTx`: reject non-`DRAFT` entry with `domain.ErrEntryNotDraft`, update header, entry date and period, delete the entry's postings, recreate them, set `posted_at`/`posted_by` when `Status` is `POSTED`. `GetByID` passes an empty flow type to `toCashFlowDomain` so the type is detected from item postings, and returns `domain.ErrTransactionNotFound` when it differs from `flowType`. In `Search`, replace the `POSTED` filter with `entry.StatusIn` over the requested status or `{DRAFT, POSTED}`. `toCashFlowDomain` fills `Items` from the item-side postings ordered by position (nil quantity/unit price fall back to `1` and the amount), takes `PaymentAccount` and totals from the payment posting.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/api/internal/fin/domain apps/api/internal/fin/model apps/api/internal/fin/repository
git commit -m "feat(api): ✨ store income and expense as multi-item documents"
```

### Task 2: Service

**Files:**
- Modify: `apps/api/internal/fin/service/cashflow.go`
- Test: `apps/api/internal/fin/service/cashflow_test.go` (suite `CashFlowServiceSuite`)

**Interfaces:**
- Consumes: Task 1 repository methods and `model.SaveCashFlowRequest`.
- Produces: `CashFlowRepositoryContract` with `Create`, `Update`, `GetByID`, `Search`; `CreateIncome`, `CreateExpense`, `UpdateIncome(c, id int64, req)`, `UpdateExpense(c, id int64, req)`, `GetIncome(c, id int64)`, `GetExpense(c, id int64)`, all taking `model.SaveCashFlowRequest` where applicable and returning `(domain.CashFlowItem, error)`; `SearchIncomes`/`SearchExpenses` unchanged. Internal: `saveCashFlow(c, flowType string, id int64, req) (domain.CashFlowItem, error)` (`id == 0` means create) and `currencyDecimals(c, currency string) int32`.

- [ ] **Step 1: Write failing tests** (the suite already seeds asset, revenue, expense, closed-period fixtures):
  - `Test_CreateIncome_TwoItems_PostsBalancedDocument`: owner's income data; total `4000000`, `EntryStatus "POSTED"`, two items with amounts `1000000` and `3000000`, `Reference "inv-123456"`.
  - `Test_CreateExpense_SameAccountTwice_Total125000` (RF1): `6710` 2 × 55,000 and 5 × 3,000; total `125000`, items `110000` and `15000`.
  - `Test_CreateIncome_Draft_EntryIsDraft`.
  - `Test_Create_InvalidItems_ReturnErrors`: table of empty items, 101 items, quantity `0` (`ErrPostingInvalidQuantity`), unit price `-1` (`ErrPostingInvalidUnitPrice`), line amount `0` (`ErrPostingInvalidAmount`).
  - `Test_Create_ItemAccountWrongType_ReturnsErrAccountInvalidType`: expense account on an income document.
  - `Test_Create_FractionalQuantityRoundsToCurrencyDecimals` (RF2): IDR (0 decimals), `2.5 × 3333` gives line amount `8333`.
  - `Test_Create_ForeignCurrency_BaseAmountsBalance` (RF3): USD, rate `16000.5`; payment base equals the sum of item bases.
  - `Test_Update_Draft_ReplacesItemsAndPostsWhenRequested`, `Test_Update_PostedDocument_ReturnsErrEntryNotDraft`, `Test_Update_LinksAdditionalAttachments` (existing attachments stay).
  - `Test_Update_PeriodClosedAfterSave_StaysDraft` and `Test_Update_DateMovesToOtherPeriod_RecomputesPeriod` (RF4): posting into a closed period returns `ErrPeriodClosed` and the document is still `DRAFT`.
  - `Test_GetExpense_OnIncomeDocument_ReturnsErrTransactionNotFound`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/service/ -run CashFlow -v`. Expected: FAIL (does not compile).
- [ ] **Step 3: Implement** `saveCashFlow` following spec §5.3 rules 1 to 9 in that order. Compute each line amount with `quantity.Mul(unitPrice).Round(currencyDecimals)`; per-item `BaseAmount = amount.Mul(rate).Round(baseDecimals)`; `TotalAmount` and `TotalBaseAmount` are the sums. On update, load with `repo.GetByID`, return `domain.ErrEntryNotDraft` unless `EntryStatus == "DRAFT"`, then call `repo.Update`; link `attachmentFileIds` exactly as `createCashFlow` does today (`fileSvc.Update` with `domain.TransactionEntityType`). Remove `createCashFlow`.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/api/internal/fin/service
git commit -m "feat(api): ✨ add item validation and draft editing to cashflow service"
```

### Task 3: Controller, routes, integration tests, regenerated contract

**Files:**
- Modify: `apps/api/internal/fin/controller/cashflow.go`, `apps/api/internal/fin/routes.go`, `apps/api/internal/fin/test/cashflow_test.go`
- Regenerate: `apps/api/docs/*`, `packages/openapi/src/api.ts`

**Interfaces:**
- Consumes: Task 2 service methods.
- Produces: handlers `GetIncome`, `GetExpense`, `UpdateIncome`, `UpdateExpense` on `CashFlowController` (Swaggo-annotated, tags `Finance Incomes` / `Finance Expenses`); routes `GET /workspaces/:wid/fin/{income,expense}/:id` (`ObjTransaction` `read`) and `PUT .../:id` (`ObjTransaction` `update`); `toCashFlowItemResponse` fills `Items`. OpenAPI schemas `model.SaveCashFlowRequest`, `model.CashFlowLineResponse`.

- [ ] **Step 1: Update and add integration tests** in `CashFlowApiSuite`, rewriting the existing payloads to `SaveCashFlowRequest`:
  - `Test_CreateIncome_TwoItemsPosted_Success`: owner's income data returns 201; `GET /entries/{entryId}` shows three postings; the trial-balance endpoint reflects `4000000` on the payment account.
  - `Test_CreateExpense_Draft_LeavesTrialBalanceUnchanged`.
  - `Test_UpdateIncome_DraftToPosted_Success`, `Test_UpdateIncome_Posted_Returns409EntryNotDraft`, `Test_GetExpense_OnIncomeID_Returns404`.
  - `Test_Search_StatusFilter`, `Test_DeleteDraftViaTransactions_Success`, `Test_DeletePostedViaTransactions_Returns409TransactionHasPostedEntries`.
  - `Test_CreateIncome_NoItems_ReturnsValidationError`: asserts the status the existing invalid-payload test in this file asserts.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/test/ -run CashFlow -v`. Expected: FAIL (does not compile).
- [ ] **Step 3: Implement** the handlers and routes (register `Get("/:id")` and `Put("/:id")` next to the existing `Post("/")` and `Post("/search")` of both groups). Bind create and update to `model.SaveCashFlowRequest`; parse `:id` as the existing controllers do.
- [ ] **Step 4: Run** `cd apps/api && gofmt -l . && go vet ./... && go test ./...`. Expected: no gofmt output, no vet errors, PASS.
- [ ] **Step 5: Regenerate the contract**: `bunx turbo run oa:gen` then `bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts`. Expected: `packages/openapi/src/api.ts` contains `model.SaveCashFlowRequest`.
- [ ] **Step 6: Commit**
```bash
git add apps/api/internal/fin apps/api/docs packages/openapi/src/api.ts
git commit -m "feat(api): ✨ expose income and expense get and update endpoints"
```

### Task 4: `MoneyInput` and cashflow helpers

**Files:**
- Create: `apps/web/app/modules/fin/components/money-input.tsx`, `money-input.test.tsx`, `cashflow-helpers.test.ts`
- Modify: `apps/web/app/modules/fin/components/cashflow-helpers.ts`, `components/index.ts`

**Interfaces:**
- Produces: `formatMoneyInputValue(value: number | string, style: DecimalStyle, decimals: number): string`; `parseMoneyInputValue(text: string, style: DecimalStyle): number | null`; `MONEY_INPUT_MIN_WIDTH = 168`; `MoneyInput(props: Omit<InputNumberProps<number>, "formatter" | "parser"> & { currency?: string })` using `getActiveDecimalStyle()` and `getCurrencyDecimals(currency)` from `~/lib/format`; `type CashflowItemValue = { accountId?: number; description?: string; quantity?: number; unitPrice?: number }`; `computeLineAmount(quantity?: number, unitPrice?: number, decimals?: number): number` and `computeTotal(items: CashflowItemValue[], decimals?: number): number` (display only, the server is authoritative).

- [ ] **Step 1: Write failing tests**: `formatMoneyInputValue(999999999, "comma", 2)` is `"999.999.999"`; `formatMoneyInputValue(1234.5, "dot", 2)` is `"1,234.5"`; `parseMoneyInputValue("999.999.999,5", "comma")` is `999999999.5`; `parseMoneyInputValue("1,234.5", "dot")` is `1234.5`; parse of `"abc"` is `null`; `renderToString(<MoneyInput value={999999999} />)` contains `min-width:168px`; `computeLineAmount(6, 500000, 0)` is `3000000`; `computeLineAmount(2.5, 3333, 0)` is `8333`; `computeTotal` of the owner's expense items is `125000`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin/components/money-input.test.tsx app/modules/fin/components/cashflow-helpers.test.ts`. Expected: FAIL (modules missing).
- [ ] **Step 3: Implement** the helpers and the component (wire the two pure functions into `InputNumber`'s `formatter` and `parser`, apply `style={{ minWidth: MONEY_INPUT_MIN_WIDTH }}`), export from `components/index.ts`.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/web/app/modules/fin/components
git commit -m "feat(web): ✨ add money input and cashflow line helpers"
```

### Task 5: Shared form, create routes, queries, i18n

**Files:**
- Create: `apps/web/app/modules/fin/components/cashflow-form.tsx`, `cashflow-form.test.tsx`
- Modify: `apps/web/app/modules/fin/queries.ts`, `routes/income-new.tsx`, `routes/expense-new.tsx`, `routes/income-expense.test.tsx`, `components/index.ts`, `apps/web/app/i18n/en.ts`, `apps/web/app/i18n/id.ts`

**Interfaces:**
- Consumes: Task 4 exports; generated `model.SaveCashFlowRequest`.
- Produces: `type CashflowFlow = "INCOME" | "EXPENSE"`; `type CashflowFormValues = { transactionDate: Dayjs; paymentAccountId: number; currency?: string; exchangeRate?: number; reference?: string; description?: string; items: CashflowItemValue[] }`; `buildCashflowPayload(values, status: "DRAFT" | "POSTED", attachmentFileIds: number[]): SaveCashFlowRequest` (quantity and unit price as strings); `submitCashflowForm(args: { wid: number; flow: CashflowFlow; documentId?: number; status: "DRAFT" | "POSTED"; values: CashflowFormValues; pendingFiles: File[]; queryClient: Pick<QueryClient, "invalidateQueries">; navigate: (to: string) => void }): Promise<void>` (`POST` without `documentId`, `PUT` with it; invalidates income or expense, transaction, and entry keys; navigates to `/fin/income` or `/fin/expense`); `CashflowActions({ pendingStatus: "POSTED" | "DRAFT" | null; onPosting; onDraft; onCancel })`; `CashflowForm({ wid, flow, documentId?, initialValues? })`; `incomeKeys.detail(wid, id)`, `expenseKeys.detail(wid, id)`, `incomeDetailQueryOptions(wid, id)`, `expenseDetailQueryOptions(wid, id)`.

- [ ] **Step 1: Write failing tests**: `submitCashflowForm` posts to `/workspaces/{wid}/fin/income` with `status: "POSTED"`, `reference: "inv-123456"`, and items quantity `"6"` and unit price `"500000"`; the Draft path sends `status: "DRAFT"`; with `documentId` it calls `PUT /workspaces/{wid}/fin/expense/{id}`; it navigates to `/fin/income`. `renderToString(<CashflowActions pendingStatus={null} … />)` contains `Posting` and `Draft`. With `pendingStatus="POSTED"` both buttons carry `disabled` and only Posting carries `ant-btn-loading` (RF5). Update `income-expense.test.tsx` to the new payloads and drop `submitIncomeForm`/`submitExpenseForm`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin`. Expected: FAIL.
- [ ] **Step 3: Implement** per spec §5.5: header (payment account via `groupPaymentAccounts`, date, currency, invoice number for income or reference for expense, description), `Form.List` items table (searchable account select limited to active postable `REVENUE` or `EXPENSE`, description, quantity, `MoneyInput` unit price, read-only line total, remove button; one initial row, last row not removable; **Add item**), document total, the accounting preview (`data-testid="cashflow-double-entry-preview"`, one payment line and one line per item), attachment picker, container wider than `max-w-3xl`. **Posting** is primary, **Draft** default style, **Cancel**. Validate with `form.validateFields()` before submitting either status. Reduce `income-new.tsx` and `expense-new.tsx` to loaders plus `<CashflowForm>`. Add the i18n keys used (item columns, Posting, Draft, status filter, add and remove item, totals).
- [ ] **Step 4: Run** the command from Step 2 plus `bun run typecheck`. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/web/app
git commit -m "feat(web): ✨ add multi-item cashflow form with posting and draft"
```

### Task 6: List, detail, and edit routes

**Files:**
- Create: `apps/web/app/modules/fin/components/cashflow-detail.tsx`, `cashflow-detail.test.tsx`, `routes/income-detail.tsx`, `routes/income-edit.tsx`, `routes/expense-detail.tsx`, `routes/expense-edit.tsx`
- Modify: `routes/income.tsx`, `routes/expense.tsx`, `route.ts`, `hooks/use-fin-permissions.ts` (only if update or delete flags are missing), `i18n/en.ts`, `i18n/id.ts`

**Interfaces:**
- Consumes: Task 5 `CashflowForm`, `incomeDetailQueryOptions`, `expenseDetailQueryOptions`.
- Produces: `summarizeCategories(items: CashFlowLine[]): { first?: string; more: number }` (`first` is `"code - name"` of the first item, `more` is `items.length - 1`); `CashflowDetail({ wid, flow, item })` with Edit (navigate to `/fin/{flow}/{id}/edit`), Posting (`POST /workspaces/{wid}/fin/entries/{entryId}/post`), Delete (`DELETE /workspaces/{wid}/fin/transactions/{id}`) shown only for `DRAFT` and gated by permissions; routes `income/:id`, `income/:id/edit`, `expense/:id`, `expense/:id/edit` registered after `income/new` and `expense/new`.

- [ ] **Step 1: Write failing tests**: `summarizeCategories` for the owner's income items returns first `"4130 - Consulting & Advisory Income"` and `more` `1`; for `6710` twice returns `more` `1` (RF1). `renderToString(<CashflowDetail … item={draft} />)` contains Edit, Posting, and Delete; for a posted item it contains none of them. The list column renders `EntryStatusTag` for `entryStatus`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin`. Expected: FAIL.
- [ ] **Step 3: Implement**: list tables get a status column (`EntryStatusTag`), a status `Select` filter (All, Draft, Posted) feeding `CashFlowFilters.status`, and a category column using `summarizeCategories`; rows link to `:id`. Edit routes load the document and redirect to `:id` when it is not `DRAFT`; both edit and detail loaders use the detail query options.
- [ ] **Step 4: Run** `cd apps/web && bunx vitest run app/modules/fin && cd ../.. && bun run typecheck && bun run lint`. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/web/app
git commit -m "feat(web): ✨ add cashflow detail, edit, and status filter"
```

### Task 7: End-to-end scenarios

**Files:**
- Modify: `e2e/features/fin/income-expense.feature`, `e2e/pages/fin/income-expense.page.ts`, `e2e/steps/fin/income-expense.steps.ts`

**Interfaces:**
- Consumes: the form field ids Ant Design generates (`items_0_accountId`, `items_0_description`, `items_0_quantity`, `items_0_unitPrice`), buttons named `Posting` and `Draft`, tags `Draft` and `Posted`.
- Produces: page methods `addItem()`, `fillItem(index, { account, description, quantity, unitPrice })`, `clickPosting()`, `clickDraft()`, `expectStatusTag(text)`; steps `When I add cashflow item {int} with account {string}, description {string}, quantity {int} and unit price {int}`, `When I click the cashflow posting button`, `When I click the cashflow draft button`, `Then I see cashflow status {string}`.

- [ ] **Step 1: Rewrite the feature** with two scenarios against the freelancer CoA: (1) income to `1121` with reference `inv-123456`, items `4130` "Konsultasi Improvement" 1 × 1000000 and `4110` "Scaffold base project structure and ci/cd" 6 × 500000, click Posting, expect `/fin/income`, the list row `inv-123456`, amount `4,000,000`, status `Posted`; (2) expense from `1111`, items `6710` "telur" 2 × 55000 and `6710` "indomie" 5 × 3000, click Draft, expect status `Draft` and amount `125,000`, open the row, Edit, click Posting, expect status `Posted`.
- [ ] **Step 2: Run** `bun run e2e -- --name "Finance Income and Expense"` (after `bunx playwright install chromium`). Expected: FAIL (undefined steps).
- [ ] **Step 3: Implement** the page methods and steps, delete the obsolete single-amount steps and the `amountInput`, `expenseAccountId`, and `revenueAccountId` locators, and grep `e2e/` for any other feature that used them.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add e2e
git commit -m "test(e2e): ✅ cover multi-item income and draft expense flows"
```

### Task 8: Documentation and final verification

**Files:**
- Modify: `README.md` (Finance routes list under Architecture)

- [ ] **Step 1: Add to the Finance route list**: `/fin/income`, `/fin/income/new`, `/fin/income/:id`, `/fin/income/:id/edit`, and the same four for `/fin/expense`, each with a one-line description (multi-item documents, Posting or Draft, draft editing).
- [ ] **Step 2: Run the full checks**: `bun run lint && bun run typecheck && bun run test`, `cd apps/api && go test ./...`, `bun run e2e`. Expected: all pass.
- [ ] **Step 3: Walk the acceptance criteria** of spec §7 (1 to 5) against the running app or the tests that pin them, and note any that lack a test.
- [ ] **Step 4: Commit**
```bash
git add README.md
git commit -m "docs(docs): 📝 document multi-item income and expense routes"
```
