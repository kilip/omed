# Finance MCP Integration (Read-Only Insight Tools) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose three read-only financial Q&A tools (`fin_period_overview`, `fin_account_balance`, `fin_find_cashflow`) through the Omed MCP server, under the same permissions as the dashboard.

**Architecture:** One new service, `InsightService`, orchestrates the existing `ReportService`, `AccountService`, and `CashFlowService` into intent-oriented results. `internal/platform/mcpx` is the only package that imports the MCP SDK; it registers tools, enforces membership and permissions, maps errors, and caps result size. `fin/mcp` is a thin adapter that declares each tool and its permission set.

**Tech Stack:** Go 1.27 (Fiber v3, Ent, `shopspring/decimal`, testify suites), `github.com/modelcontextprotocol/go-sdk` (owner decision, 2026-11-01).

**Spec:** `docs/superpowers/specs/2026-11-01-fin-mcp-integration-design.md`.

## Global Constraints

- SDK is `github.com/modelcontextprotocol/go-sdk`, imported **only** from `apps/api/internal/platform/mcpx` and from `_test.go` files. `fin/mcp` and `fin/service` never import it.
- Layering: `mcp -> service -> repository`. `mcp/` never imports `controller` packages and never reaches repositories (spec §2).
- Every tool declares a non-empty `(object, action)` permission set; registration panics otherwise (spec §2).
- A non-member of the workspace receives `NOT_FOUND`; a member missing a permission receives `FORBIDDEN` (spec §2, §5.2).
- Money is `shopspring/decimal`, serialized as a JSON string, and every result carries a currency code. No floating point (spec §2).
- No raw internal errors: `UNKNOWN` errors are logged with `slog` and returned as a generic message with a reference id (spec §5.6).
- Limits are constants in code: 50 accounts, 50 cashflow items, 24 months, 64 KiB result cap. No new environment variables, so no Helm change (spec §2, §5.9).
- Tools are read-only. No posting, reversing, voiding, or period actions (spec §2).
- Out of scope: phase B (`fin_find_accounts`, `fin_create_cashflow_draft`, receipt input) gets its own spec and plan; top categories; amount-range filters; receipt storage; exposing `agent` or membership administration (spec §2, §5.8).
- Test pattern: service tests are testify suites on `testutil.CreateTestDB` (real in-memory DB), next to the file under test. MCP-level tests use the SDK in-memory transport.
- Commits: `type(scope): <emoji> subject`, scope `api` or `docs`, one logical change each, ending with the attribution trailers the executing session requires.

## Resolved Open Items (spec §9, checked against the code)

- **#1 SDK:** go-sdk. `mcp.AddTool[In, Out](server, *mcp.Tool, handler)` with handler `func(ctx, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)`; `mcp.NewStreamableHTTPHandler(getServer, *mcp.StreamableHTTPOptions)` returns an `http.Handler`. Verify exact names with `go doc` in Task 1; this plan was written from the SDK docs, not from compiled code.
- **#3** `AccountService.Search` returns the whole chart ordered by code with an empty cursor (`repository/account.go`), so account references resolve in memory.
- **#4** `TrialBalanceAccountItem` has `AccountCode`, `AccountName`, `Type`, `Debit`, `Credit`, `NetBalance` (base currency).
- **#5** `GetMonthlyAccountFlows` feeds `ReportService.Summary` with `BaseCredit - BaseDebit` for revenue and `BaseDebit - BaseCredit` for expense. `Summary` only supports trailing months ending today, so Task 2 extracts that logic for arbitrary ranges.
- **#6** `SaveCashFlowRequest` already has `Status` (`DRAFT`/`POSTED`). Phase B is out of scope here.
- **Still open, owned by sub-project 2:** #2 (PAT scope), #7, #8.

## Prerequisite from sub-project 2 (not in this plan)

Sub-project 2 mounts `/mcp` and authenticates PATs. This plan assumes it will: build `mcpx.NewRegistrar(...)` once at boot, call `RegisterMCP` on every module that implements `config.HasMCP`, serve `registrar.Server()` with `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})` behind a Fiber adapter, and have its token verifier set `TokenInfo.UserID`. Stateless plus JSON responses is recommended so no SSE stream has to pass through Fiber's adapter; confirm in the spike. Tasks 1 to 6 do not need any of this. Task 7 does.

## Review Focus

- RF1: `workspace_id` missing or `0` returns a validation message saying it is required, not `NOT_FOUND`. Pinned in Task 1.
- RF2: an empty or whitespace-only `account` filter means "no filter", not "match everything by name". Pinned in Task 4.
- RF3: a workspace with no accounts (not onboarded) returns an empty list, `is_balanced: true`, and a base currency, not an error. Pinned in Task 4.
- RF4: an account reference equal to one exact code never becomes ambiguous because other names contain the same text. Pinned in Task 5.
- RF5: description text with multi-byte characters (Indonesian text, emoji) is cut on a rune boundary and stays valid UTF-8. Pinned in Task 5.

---

### Task 1: `mcpx` registrar, permission wrapper, error mapping, size cap

**Files:**
- Create: `apps/api/internal/platform/mcpx/registrar.go`, `tool.go`, `errors.go`
- Test: `apps/api/internal/platform/mcpx/registrar_test.go`, `tool_test.go`
- Modify: `apps/api/go.mod`, `apps/api/go.sum`

**Interfaces:**
- Produces:
  - `type Permission struct{ Object, Action string }`
  - `type ToolSpec struct{ Name, Description string; ReadOnly bool; Permissions []Permission }`
  - `type WorkspaceScoped interface{ GetWorkspaceID() int64 }`
  - `type Shrinkable interface{ Shrink() bool }` (implemented on pointer receivers; drops trailing items, sets a `Truncated` flag and a hint, returns `false` when nothing is left to drop)
  - `type ErrorDetailer interface{ ErrorDetail() string }` (appended to the mapped message)
  - `const DefaultMaxResultBytes = 64 << 10`
  - `type Config struct{ Authorizer *authz.Authorizer; Resolver authz.MembershipResolver; UserID func(*mcp.CallToolRequest) (int64, bool); Logger *slog.Logger; MaxResultBytes int }` (`UserID` defaults to parsing `req.Extra.TokenInfo.UserID`; `MaxResultBytes` defaults to `DefaultMaxResultBytes`)
  - `func NewRegistrar(cfg Config) *Registrar` and `func (r *Registrar) Server() *mcp.Server`
  - `func AddTool[In WorkspaceScoped, Out any](r *Registrar, spec ToolSpec, h func(ctx context.Context, in In) (Out, error))`
- Tool errors are returned as `*mcp.CallToolResult{IsError: true}` with one text content holding `{"code":"...","message":"..."}`, never as a Go error, so the client can self-correct.

- [x] **Step 1: Add the dependency**: `cd apps/api && go get github.com/modelcontextprotocol/go-sdk@latest && go mod tidy`. Run `go doc github.com/modelcontextprotocol/go-sdk/mcp AddTool`, `... NewInMemoryTransports`, `... ToolAnnotations`, and `... CallToolRequest`, and fix any name in this task that differs.
- [x] **Step 2: Write failing tests** with a harness that builds an `authz.Authorizer` from `authz.NewWithRegistry` (test roles holding `fin:account read` and `fin:transaction read`), a stub `MembershipResolver`, a `UserID` stub, a `bytes.Buffer` logger, and an in-memory client session:
  - `Test_AddTool_PanicsWithoutPermissions`; `Test_AddTool_PanicsOnUnknownPermissionObject` (`fin:acount`); `Test_AddTool_PanicsWhenNameLacksModulePrefix` (`balance` with a `fin:account` permission). The known-object set comes from `cfg.Authorizer.Roles()`, ignoring wildcard objects.
  - `Test_Call_NonMember_ReturnsNotFoundToolError` (`IsError` true, code `NOT_FOUND`); `Test_Call_MissingOnePermission_ReturnsForbidden` (tool needs two, role holds one).
  - `Test_Call_WorkspaceIDMissing_ReturnsValidationMessage` (RF1): message is `workspace_id is required and must be a positive integer`, code `VALIDATION_FAILED`.
  - `Test_Call_NoUser_ReturnsUnauthorized`.
  - `Test_Call_Success_ContextCarriesMembership`: the handler sees `base.WorkspaceIDFromContext(ctx)` equal to the argument and `authz.MembershipFromContext(ctx)` ok.
  - `Test_Call_UnknownError_IsGenericAndLogged`: handler returns `errors.New("pq: secret")`; the client text has code `UNKNOWN` and a reference id but not `secret`; the log buffer contains `secret` and the same reference id.
  - `Test_Call_ErrorDetailer_AppendedToMessage`.
  - `Test_Call_OversizeResult_IsShrunkAndFlagged` (cap set to 512 bytes; output type implements `Shrinkable`) and `Test_Call_OversizeNotShrinkable_ReturnsResultTooLarge`.
  - `Test_ListTools_ReadOnlyHintAndRequiredWorkspaceID`: `annotations.readOnlyHint` is true and the input schema requires `workspace_id`.
- [x] **Step 3: Run** `cd apps/api && go test ./internal/platform/mcpx/ -v`. Expected: FAIL (package does not compile).
- [x] **Step 4: Implement** per the Interfaces block. `AddTool` validates the spec (panic messages name the tool), builds `&mcp.Tool{Name, Description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: spec.ReadOnly}}`, and registers a wrapped handler on `r.Server()` in this order: read `in.GetWorkspaceID()` (≤ 0 is a validation error) -> `cfg.UserID` -> `cfg.Resolver.ResolveMembership` -> `authz.ContextWithMembership` -> `cfg.Authorizer.Can` for every permission (`base.ErrForbidden`) -> handler -> size cap. Map errors with `httpx.MapError`; for code `UNKNOWN`, log with `slog` at error level and answer `internal error (ref: <uuid>)`. The size cap marshals the output and calls `Shrink()` until it fits or returns `false`.
- [x] **Step 5: Run** the command from Step 3. Expected: PASS.
- [x] **Step 6: Commit**
```bash
git add apps/api/go.mod apps/api/go.sum apps/api/internal/platform/mcpx
git commit -m "feat(api): ✨ add mcp tool registrar on the go sdk"
```

### Task 2: Share the monthly revenue and expense computation

**Files:**
- Modify: `apps/api/internal/fin/service/report.go`
- Test: `apps/api/internal/fin/service/report_test.go` (suite `ReportServiceSuite`)

**Interfaces:**
- Produces: `func (s ReportService) MonthlyRevenueExpense(ctx context.Context, firstMonth time.Time, months int) ([]model.MonthlyRevenueExpenseItem, error)`. `firstMonth` is any instant inside the first month; the result has exactly `months` ordered rows, zero rows for months without flows; no range clamp (callers validate). `Summary` keeps its 1 to 24 clamp and calls this method for `Monthly`.

- [x] **Step 1: Write failing tests** reusing the suite's seeded revenue and expense entries across three months:
  - `Test_MonthlyRevenueExpense_EqualsSummaryMonthly`: for the same three months, every row equals `Summary(ctx, 3).Monthly`.
  - `Test_MonthlyRevenueExpense_EmptyMonthsAreZeroRows`: a range before any entry returns the requested number of rows with zero amounts.
  - `Test_MonthlyRevenueExpense_RangeNotEndingToday`: a range ending two months ago contains only entries from that range.
- [x] **Step 2: Run** `cd apps/api && go test ./internal/fin/service/ -run Report -v`. Expected: FAIL (`MonthlyRevenueExpense` undefined).
- [x] **Step 3: Implement** `MonthlyRevenueExpense` by moving the account map, `GetMonthlyAccountFlows` call, and per-month classification out of `Summary` unchanged, then make `Summary` call it with `startMonthTime` and `months`.
- [x] **Step 4: Run** the command from Step 2. Expected: PASS, including every pre-existing `Summary` test.
- [x] **Step 5: Commit**
```bash
git add apps/api/internal/fin/service/report.go apps/api/internal/fin/service/report_test.go
git commit -m "refactor(api): ♻️ extract monthly revenue and expense for reuse"
```

### Task 3: `InsightService.PeriodOverview`

**Files:**
- Create: `apps/api/internal/fin/model/insight.go`, `apps/api/internal/fin/service/insight.go`
- Modify: `apps/api/internal/fin/domain/error.go`
- Test: `apps/api/internal/fin/service/insight_test.go` (suite `InsightServiceSuite`)

**Interfaces:**
- Consumes: Task 2 `MonthlyRevenueExpense`. This task also adds `func (s ReportService) BaseCurrency(ctx context.Context) string` to `report.go`, a thin wrapper over the existing private `getBaseCurrencyAndDecimals`, so `InsightService` can report the currency without a repository. Spec §5.1 lists the report repository as a dependency; it is not needed because Task 2 exposes the monthly computation on `ReportService`.
- Produces (domain): `var ErrInvalidArgument = errors.New("INVALID_ARGUMENT")` mapped to 422 `invalid argument`; `type ArgumentError struct{ Field, Expect string }` with `Error() string`, `Is(target error) bool` (matches `ErrInvalidArgument`) and `ErrorDetail() string` (for example `from_month must use the format YYYY-MM`).
- Produces (model, JSON names snake_case, `jsonschema` tags carry the argument descriptions the client model reads): `PeriodOverviewRequest{ WorkspaceID int64 "workspace_id"; FromMonth, ToMonth string "from_month,omitempty","to_month,omitempty"; ComparePrevious *bool "compare_previous,omitempty" }` with `GetWorkspaceID() int64`; `MonthTotals{ Month string; Revenue, Expense, Net decimal.Decimal }`; `TotalsDelta{ Revenue, Expense, Net decimal.Decimal }`; `PeriodComparison{ From, To string; Revenue, Expense, Net decimal.Decimal; Monthly []MonthTotals; Delta TotalsDelta }` (`Delta` is current minus previous); `PeriodOverviewResponse{ BaseCurrency, From, To string; Revenue, Expense, Net decimal.Decimal; Monthly []MonthTotals; Previous *PeriodComparison }` (`Previous` is nil when `compare_previous` is false).
- Produces (service): `func NewInsightService(report ReportService, accounts AccountService, cashflow CashFlowService, opts ...InsightOption) InsightService`; `type InsightOption func(*InsightService)`; `func WithInsightClock(now func() time.Time) InsightOption`; `func (s InsightService) PeriodOverview(c context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error)`.

- [x] **Step 1: Write failing tests** (seed like `ReportServiceSuite`; clock fixed at `2026-05-15`):
  - `Test_PeriodOverview_DefaultsToCurrentMonthWithPrevious`: no months given returns `from == to == "2026-05"` and a `Previous` of `2026-04`.
  - `Test_PeriodOverview_RangeEqualsSummaryMonthly` (acceptance 2): each `Monthly` row equals the `ReportService` figure for that month.
  - `Test_PeriodOverview_PreviousIsSameLengthEndingBeforeFrom`: `2026-03` to `2026-05` gives previous `2025-12` to `2026-02`, and `Delta` equals current minus previous.
  - `Test_PeriodOverview_CompareFalse_OmitsPrevious`.
  - `Test_PeriodOverview_EmptyMonthsStillListed`: a month with no entries appears with zero amounts that marshal to JSON strings.
  - `Test_PeriodOverview_InvalidInput`: table of `2026-13`, `2026/03`, `to` before `from`, and a 25-month range, each `errors.Is(err, domain.ErrInvalidArgument)` with the exact `ErrorDetail()` text (`from_month must use the format YYYY-MM`, `to_month must not be before from_month`, `range must not exceed 24 months`).
- [x] **Step 2: Run** `cd apps/api && go test ./internal/fin/service/ -run Insight -v`. Expected: FAIL (does not compile).
- [x] **Step 3: Implement** the types and `PeriodOverview`: parse months with layout `2006-01`, default both to the clock's current month, `compare_previous` defaults to `true`, compute `Net = Revenue - Expense` per month and total, and the previous range as the same length ending the month before `from`.
- [x] **Step 4: Run** the command from Step 2. Expected: PASS.
- [x] **Step 5: Commit**
```bash
git add apps/api/internal/fin/domain apps/api/internal/fin/model/insight.go apps/api/internal/fin/service
git commit -m "feat(api): ✨ add period overview insight"
```

### Task 4: `InsightService.AccountBalance`

**Files:**
- Modify: `apps/api/internal/fin/model/insight.go`, `apps/api/internal/fin/service/insight.go`
- Test: `apps/api/internal/fin/service/insight_test.go`

**Interfaces:**
- Consumes: `ReportService.TrialBalance(ctx, model.TrialBalanceRequest{AsOfDate: model.Date(t)})`; `InsightService` and `WithInsightClock` from Task 3.
- Produces (model): `AccountBalanceRequest{ WorkspaceID int64 "workspace_id"; AsOfDate string "as_of_date,omitempty"; Account string "account,omitempty" }` with `GetWorkspaceID()`; `AccountBalanceItem{ Code, Name, Type string; Debit, Credit, Net decimal.Decimal }`; `BalanceTotals{ Debit, Credit decimal.Decimal }`; `AccountBalanceResponse{ AsOfDate, BaseCurrency string; Accounts []AccountBalanceItem; Totals BalanceTotals; IsBalanced bool; Truncated bool; Hint string }` with `Shrink() bool` on the pointer receiver (drops the last half of `Accounts`, sets `Truncated` and `Hint`).
- Produces (service): `const maxInsightAccounts = 50`; `func (s InsightService) AccountBalance(c context.Context, req model.AccountBalanceRequest) (model.AccountBalanceResponse, error)`.

- [ ] **Step 1: Write failing tests**:
  - `Test_AccountBalance_CodePrefix`: `account: "11"` returns only codes starting with `11`; totals equal the sum over those accounts (acceptance 3).
  - `Test_AccountBalance_DigitsOnlyNeverMatchesNames`: an account named `Fund 11` with code `5000` is not returned for `"11"`.
  - `Test_AccountBalance_NameFragmentCaseInsensitive`: `"BANK"` matches `Bank Kaltim`.
  - `Test_AccountBalance_EmptyFilterMeansAll` (RF2): `""` and `"   "` return the full chart.
  - `Test_AccountBalance_MoreThan50_TruncatedWithTotalsOverAll`: 60 matched accounts return 50, `Truncated` true, a hint mentioning `account`, and totals over all 60.
  - `Test_AccountBalance_NoAccounts_EmptyAndBalanced` (RF3).
  - `Test_AccountBalance_IsBalancedIgnoresFilter`: `IsBalanced` reflects the whole ledger even when the filter matches one account.
  - `Test_AccountBalance_InvalidDate`: `as_of_date must use the format YYYY-MM-DD` as `ArgumentError`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/service/ -run Insight -v`. Expected: FAIL.
- [ ] **Step 3: Implement** `AccountBalance`: the date defaults to the clock's today; a trimmed filter that is all digits matches by code prefix only, otherwise it matches a code prefix or a case-insensitive name fragment (this disambiguation is a plan decision; the spec only names both forms). Per-currency breakdown is not requested.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/api/internal/fin/model/insight.go apps/api/internal/fin/service
git commit -m "feat(api): ✨ add account balance insight"
```

### Task 5: `InsightService.FindCashflow` and account references

**Files:**
- Modify: `apps/api/internal/fin/domain/error.go`, `apps/api/internal/fin/model/insight.go`, `apps/api/internal/fin/service/insight.go`
- Test: `apps/api/internal/fin/service/insight_test.go`

**Interfaces:**
- Consumes: `CashFlowService.SearchIncomes/SearchExpenses(c, model.SearchCashFlowRequest)`, `AccountService.Search`, `domain.ErrAccountNotFound`, `base.ErrInvalidCursor`, `ArgumentError` from Task 3.
- Produces (domain): `var ErrAccountReferenceAmbiguous = errors.New("ACCOUNT_REFERENCE_AMBIGUOUS")` mapped to 422 `account reference matches more than one account`; `type AmbiguousAccountError struct{ Reference string; Candidates []AccountRef }` whose `Is` matches the sentinel and whose `ErrorDetail()` lists at most five `code name` pairs.
- Produces (model): `AccountRef{ Code, Name string }`; `FindCashflowRequest{ WorkspaceID int64 "workspace_id"; Type string "type"; DateFrom, DateTo, Status, Query, PaymentAccount, CategoryAccount, Cursor string; Limit int }` (all optional except the first two, snake_case JSON) with `GetWorkspaceID()`; `CashflowLine{ Account AccountRef; Description string; Quantity, UnitPrice, Amount decimal.Decimal }`; `CashflowDoc{ ID int64; Number, Date, Type, Status string; PaymentAccount AccountRef; Description, Reference string; Amount decimal.Decimal; Currency string; BaseAmount decimal.Decimal; Lines []CashflowLine }` (no attachments, audit metadata, workspace id, or entry id); `FindCashflowResponse{ Items []CashflowDoc; NextCursor string "next_cursor"; Truncated bool; Hint string }` with `Shrink() bool`.
- Produces (service): `const maxInsightItems = 50`, `const maxInsightTextRunes = 200`; `func (s InsightService) FindCashflow(c context.Context, req model.FindCashflowRequest) (model.FindCashflowResponse, error)`; `func resolveAccountRef(accounts []domain.Account, ref string) (domain.Account, error)`; `func truncateText(s string, maxRunes int) string`.

- [ ] **Step 1: Write failing tests**:
  - `Test_FindCashflow_TypeStatusDatesQueryAndAccounts`: table over `income`/`expense`, `draft`/`posted`, a date range, a text query, and payment and category references, asserting the expected document numbers (acceptance 4). `Status` and `Type` are lowercase in the output.
  - `Test_FindCashflow_CursorRoundTrip_NoDuplicates`: five documents, `limit: 2`, following `NextCursor` until empty returns every document exactly once.
  - `Test_FindCashflow_OutputHasNoInternalFields`: the marshaled item contains none of the keys `attachments`, `auditMeta`, `entryId`, `entryNumber`, `workspaceId` (acceptance 7).
  - `Test_ResolveAccountRef_ExactCodeThenNameFragment`; `Test_ResolveAccountRef_Unknown_ReturnsAccountNotFound`; `Test_ResolveAccountRef_Ambiguous_ListsUpToFiveCandidates` (seven matches, five listed in `ErrorDetail()`); `Test_ResolveAccountRef_ExactCodeWinsOverNameMatches` (RF4: code `6710` also appears inside other account names).
  - `Test_FindCashflow_LimitAbove50_Rejected`: `limit = 51` returns `ArgumentError` with `limit must be between 1 and 50; use cursor to read more`; `limit = 0` defaults to 20.
  - `Test_FindCashflow_InvalidTypeOrStatus`: `type must be "income" or "expense"`, `status must be "draft" or "posted"`.
  - `Test_FindCashflow_InvalidCursor_ReturnsErrInvalidCursor`.
  - `Test_TruncateText_KeepsValidUTF8` (RF5): 250 runes of `é` and emoji cut to 200 runes, `utf8.ValidString` true.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/service/ -run Insight -v`. Expected: FAIL.
- [ ] **Step 3: Implement** `FindCashflow`. Load the chart once with `AccountService.Search` only when a reference is present; map `type` to the matching search; map `date_from`/`date_to` with `model.Date`; uppercase `status`; default `limit` 20; map each `domain.CashFlowItem` to `CashflowDoc` (document `ID`, not entry id) and cut descriptions with `truncateText`.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/api/internal/fin/domain apps/api/internal/fin/model/insight.go apps/api/internal/fin/service
git commit -m "feat(api): ✨ add cashflow search insight"
```

### Task 6: `fin/mcp` tools, module registration, `HasMCP`

**Files:**
- Create: `apps/api/internal/fin/mcp/register.go`, `overview.go`, `balance.go`, `cashflow.go`
- Test: `apps/api/internal/fin/mcp/register_test.go`
- Modify: `apps/api/internal/fin/module.go`, `apps/api/internal/config/contracts.go`
- Test: `apps/api/internal/config/contracts_test.go`

**Interfaces:**
- Consumes: Task 1 `mcpx.AddTool`, `mcpx.ToolSpec`, `*mcpx.Registrar`; Tasks 3 to 5 request, response, and service types.
- Produces: `type Insights interface{ PeriodOverview(context.Context, model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error); AccountBalance(context.Context, model.AccountBalanceRequest) (model.AccountBalanceResponse, error); FindCashflow(context.Context, model.FindCashflowRequest) (model.FindCashflowResponse, error) }`; `func RegisterTools(r *mcpx.Registrar, in Insights)`; `FinModule.InsightService service.InsightService` built in `New()` as `service.NewInsightService(reportSvc, accountSvc, cashFlowSvc)`; `func (m *FinModule) RegisterMCP(r *mcpx.Registrar) error`; `type HasMCP interface{ RegisterMCP(r *mcpx.Registrar) error }` in `config/contracts.go`.
- Tool permissions: `fin_period_overview` and `fin_account_balance` require `fin:account read`; `fin_find_cashflow` requires `fin:transaction read` and `fin:account read`. All three are `ReadOnly: true`. Descriptions are read by the client model, so each states its units, defaults, and limits (months `YYYY-MM`, max 24; accounts max 50; `limit` 1 to 50).

- [ ] **Step 1: Write failing tests** against a real `mcpx.Registrar` (stub resolver and roles as in Task 1) and a fake `Insights`, called through the SDK in-memory client (import the SDK as `sdk` in test files only):
  - `Test_RegisterTools_ThreeReadOnlyTools`: tool names, `readOnlyHint`, and a required `workspace_id` property.
  - `Test_FindCashflow_RoleWithoutAccountRead_Forbidden` (acceptance 6): a role holding only `fin:transaction read` gets `FORBIDDEN`.
  - `Test_Tool_NonMember_NotFound`.
  - `Test_Tools_MoneyIsStringWithCurrency`: each tool's structured content has string amounts and a currency field (acceptance 7).
  - `Test_ArgumentError_ReachesClientWithExpectedFormat`: the fake returns `ArgumentError`, the client text contains `from_month must use the format YYYY-MM`.
  - In `config/contracts_test.go`: `var _ HasMCP = (*fin.FinModule)(nil)` (the assertion lives in `config`, because `fin` importing `config` would be an import cycle).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/mcp/ ./internal/config/ -v`. Expected: FAIL.
- [ ] **Step 3: Implement** `RegisterTools` (one `mcpx.AddTool` call per tool, each in its own file with its description constant), then `RegisterMCP` on `FinModule` calling `RegisterTools(r, m.InsightService)`, and `HasMCP` next to `HasPreBoot`.
- [ ] **Step 4: Run** the command from Step 2. Expected: PASS.
- [ ] **Step 5: Commit**
```bash
git add apps/api/internal/fin apps/api/internal/config
git commit -m "feat(api): ✨ register fin insight tools with the mcp server"
```

### Task 7: Integration tests through `/mcp` and documentation

**Blocked until sub-project 2 mounts `/mcp` and issues PATs.**

**Files:**
- Create: `apps/api/internal/fin/test/mcp_test.go`
- Modify: `AGENTS.md` (section 3 module layout gains `mcp/`), `README.md` (MCP endpoint and the three tools)

**Interfaces:**
- Consumes: the `/mcp` endpoint and PAT creation helper from sub-project 2 (names fixed in its plan); `testutil.ApiTestSuite`.

- [ ] **Step 1: Write integration tests** (spec §6) using the owner's seeded data: success for each tool; `Test_Mcp_NonMember_NotFound`; `Test_Mcp_RoleWithoutAccountRead_ForbiddenOnFindCashflow`; `Test_Mcp_FindCashflow_CursorRoundTrip`; `Test_Mcp_AccountBalance_TruncatedFlag` (60 accounts); `Test_Mcp_PeriodOverview_EqualsReportSummary`; `Test_Mcp_RevokedToken_Rejected` (transport-level, asserts only that no tool runs).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/fin/test/ -run Mcp -v`. Expected: FAIL before the mount exists, then PASS.
- [ ] **Step 3: Update docs**: add `mcp/` to the layer list in `AGENTS.md` section 3 (`mcp/` — MCP tool adapters, registered through `mcpx`, never import controllers or repositories), and add a README section naming the endpoint, authentication by PAT, and the three tools with their arguments.
- [ ] **Step 4: Verify everything**: `cd apps/api && gofmt -l . && go vet ./... && go test ./...` then `bun run lint`. Expected: no gofmt output, no vet errors, PASS (acceptance 9).
- [ ] **Step 5: Commit** (two commits)
```bash
git add apps/api/internal/fin/test/mcp_test.go
git commit -m "test(api): ✅ cover fin mcp tools through the mcp endpoint"
git add AGENTS.md README.md
git commit -m "docs(docs): 📝 document the mcp layer and fin insight tools"
```
