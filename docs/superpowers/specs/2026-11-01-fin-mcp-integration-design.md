# Design Spec: Finance (`fin`) Integration with the Omed MCP Server

- **Date:** 2026-11-01
- **Status:** Draft / Under Review
- **Module:** `fin` (Financial Accounting), with a platform contract towards `platform/mcpx`
- **Author:** Claude (pair programming with the repository owner)
- **Related:** [`2026-10-30-agent-provider-design.md`](./2026-10-30-agent-provider-design.md) lists "MCP server" as out of scope for the `agent` module. This document does not change that: the MCP server described here belongs to `platform/mcpx` and the modules, not to `agent`.

---

## 1. Intent & The Why

Omed exposes its features to AI clients through the Model Context Protocol (MCP), so that end users can work with their own data from Claude or any other MCP client. The two directions must not be confused:

- `agent` module: Omed calls an LLM using a provider supplied by the workspace (BYOK).
- MCP server: an LLM client calls Omed on behalf of an authenticated user.

The MCP effort is delivered as three ordered sub-projects. Each has its own spec, plan, and implementation cycle:

| # | Sub-project | Output |
| --- | --- | --- |
| 1 | Spike: Streamable HTTP transport of the chosen SDK (go-sdk) on Fiber v3 | A written recommendation, throwaway code only |
| 2 | Foundation: `platform/mcpx` plus personal access tokens (PAT) in `core` | Transport, token auth, registry, permission wrapper |
| 3 | **This spec:** `fin` tools | Read-only financial Q&A first, draft input from receipts later |

The first user-visible outcome is that a user can ask a client questions such as "how is my cash position this month", "what is the balance of my bank accounts", or "find the expenses paid from Cash on Hand last week", and receive accurate answers drawn from the workspace ledger, under the same permissions as the dashboard.

Decisions already taken by the owner and inherited by this spec:

- The MCP server is a product feature for end users, not developer tooling.
- Omed remains a modular monolith. Tools are defined per module; one shared server exposes them at a single endpoint.
- Authentication for external clients uses personal access tokens first. An OAuth authorization server is deferred.
- The workspace is an explicit tool argument.
- Tools start read-only. Writes are limited to drafts and never post automatically.
- Tools are intent-oriented (built around user questions), not one-to-one wrappers of REST endpoints.
- The MCP SDK is the official Go SDK, `github.com/modelcontextprotocol/go-sdk`. It is imported only from `platform/mcpx` (and from tests); modules never import it.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **Three read-only intent tools in the first batch:** `fin_period_overview`, `fin_account_balance`, `fin_find_cashflow`.
- **Same authorization as REST.** Every tool declares the union of the permissions that the corresponding REST routes require, and the shared wrapper enforces it before any handler runs.
- **LLM-friendly output.** Monetary values are decimal strings accompanied by a currency code, accounts are shown with code and name, internal fields are omitted, and result size is bounded.
- **Thin adapters.** Orchestration lives in one new service, `InsightService`, so the same insight can later serve REST or the dashboard.
- **Phase B (outline only):** draft income and expense creation from a receipt that the client has already read.

### Non-Goals
- **Posting, reversing, voiding, closing periods, or any ledger-mutating action.**
- **Tools for periods, settings, exchange rates, currencies, ledger, and single-document detail** in the first batch. They are candidates for later batches.
- **Top categories** (largest expense or income by account). It may require a new aggregation query and is deferred.
- **Amount-range filters** in `fin_find_cashflow`. `SearchCashFlowRequest` has no such field and adding one changes the repository.
- **Receipt storage.** In the first release of phase B the receipt image is not stored in Omed.
- **Server-side receipt extraction** with the `agent` module providers. The MCP client is already an LLM and reads the receipt itself.
- **Exposing the `agent` module or membership administration** through MCP.
- **New environment variables.** Limits in this spec are constants in code.

### Anti-Goals
- Do not let a tool bypass `Authorizer`. A tool without a declared permission set must fail at registration.
- Do not leak the existence of workspaces: a non-member receives `NOT_FOUND`, as defined by the `MembershipResolver` contract.
- Do not return raw internal errors, stack traces, or `err.Error()` of unknown errors to the client.
- Do not import `controller` packages from `mcp/`, and do not reach repositories from `mcp/`.
- Do not return floating-point money.

---

## 3. Constraints

- **Layering (AGENTS.md section 3).** Dependency direction stays `mcp` -> `service` -> `repository`. A module reaches another module through that module's service. Only repositories talk to the database.
- **Module contract.** Modules implement `config.Module` (`Boot`) and optionally `HasPreBoot`. This spec adds an optional `HasMCP` interface next to them in `apps/api/internal/config/contracts.go`.
- **Workspace through context.** Services read the workspace with `base.WorkspaceIDFromContext`. The wrapper must populate it with `authz.ContextWithMembership`.
- **Permissions.** Permission objects are `fin:account`, `fin:transaction`, and the others in `fin/domain/authz.go`. The built-in `fin:viewer` role already holds all read permissions used here.
- **Errors.** Error codes and friendly messages come from the registry in `base.AddToErrorMap`, resolved through `httpx.MapError`.
- **Money.** Amounts use `shopspring/decimal`, which serializes to JSON strings.
- **Dependencies.** Implementation of this spec is blocked by sub-projects 1 and 2. Unit tests can run earlier against a fake registrar.
- **Documentation.** This spec follows the conventions of the existing documents in `docs/superpowers/specs/`, and is written in English, in Markdown (AGENTS.md section 1).

---

## 4. Approaches Considered

### Approach 1: Thin wrappers, one tool per service method
Each service method becomes a tool and returns the `model.*Response` as is. Least code and no new business logic, but output is noisy for an LLM (ids without names, internal fields) and list results waste tokens.

### Approach 2: Intent-oriented tools on top of a new service (Selected by the owner)
Tools are built around user questions and combine existing services, resolving account names and computing comparisons. The orchestration is placed in `fin/service/insight.go` so that the `mcp/` layer stays an adapter. This keeps the layering rules intact and makes the result reusable outside MCP.

### Approach 3: One generic `fin_query` tool
Rejected. Permissions per `(object, action)` become ambiguous and the input schema cannot be validated meaningfully.

---

## 5. Detailed Design

### 5.1 Components

```
apps/api/internal/fin/
├── model/insight.go        # request/response types for the insight tools
├── service/insight.go      # InsightService: the only place with new logic
├── mcp/
│   ├── register.go         # RegisterMCP(): registers the tools with their permissions
│   ├── overview.go         # fin_period_overview
│   ├── balance.go          # fin_account_balance
│   └── cashflow.go         # fin_find_cashflow
└── module.go               # FinModule implements HasMCP and builds InsightService in New()

apps/api/internal/platform/mcpx/   # delivered by sub-project 2
```

- `InsightService` depends on `ReportService`, `AccountService`, `CashFlowService`, and the report repository. It does not use Ent directly.
- `fin/mcp/` decodes arguments, calls `InsightService`, and serializes the result. It contains no business logic.
- `FinModule.RegisterMCP(r mcpx.Registrar) error` is invoked during `Boot`, after the services exist.

### 5.2 Contract required from `platform/mcpx`

The final shape of `mcpx` is decided by the sub-project 2 spec. `fin` requires the following behavior:

1. **Registration** with a tool name, description, a typed input (JSON schema derived from a Go struct), a read-only hint, and a non-empty list of required `(object, action)` permissions. Registration **panics** when the permission list is empty or references an unknown object, mirroring `authz.AddRole`.
2. **Wrapper** executed before the handler:
   - authenticate the caller (PAT, outside this spec) and read `workspace_id` from the arguments;
   - resolve membership with `MembershipResolver` (non-member returns `NOT_FOUND`);
   - build the context with `authz.ContextWithMembership`;
   - check every declared permission with `Authorizer.Can` (missing permission returns `FORBIDDEN`);
   - map errors with `httpx.MapError`;
   - enforce the response size cap (section 5.7).
3. **Naming.** Tool names are prefixed with the module (`fin_`) so they align with permission objects.
4. **Mapping onto go-sdk.** Registration wraps `mcp.AddTool[In, Out]` and sets `ToolAnnotations.ReadOnlyHint` from the read-only hint; the input schema is inferred from the Go struct tags (`json`, `jsonschema`), which also carry the argument descriptions read by the client model. Errors listed in section 5.6 are returned as a `CallToolResult` with `IsError` set, not as Go errors. The caller identity comes from the token verifier (`TokenInfo.UserID`, read from the request extra). The recommended transport options are `Stateless` and `JSONResponse`, so that no event stream has to pass through the Fiber adapter; sub-project 1 confirms this. Symbol names are taken from the SDK documentation and must be confirmed with `go doc` during implementation.

### 5.3 Tool: `fin_period_overview`

Answers "how are we doing this month or over these months".

| | |
| --- | --- |
| Arguments | `workspace_id` (required), `from_month` and `to_month` (`YYYY-MM`, default the current month), `compare_previous` (default `true`) |
| Permissions | `fin:account` `read` |
| Returns | `base_currency`, `from`, `to`, `revenue`, `expense`, `net`, a `monthly` list (`month`, `revenue`, `expense`, `net`), and when requested a `previous` block with the same totals plus `delta` |

Rules:
- Granularity is the month. Arbitrary day ranges are not supported in the first batch.
- `to_month` must not be before `from_month`. The maximum range is 24 months.
- Revenue and expense classification must be identical to `ReportService.Summary`. For months covered by both, the figures must be equal (see section 7).
- The previous period is the range of the same length that ends the month before `from_month`.

### 5.4 Tool: `fin_account_balance`

Answers "what is the balance of account X, or of the accounts starting with 11".

| | |
| --- | --- |
| Arguments | `workspace_id` (required), `as_of_date` (`YYYY-MM-DD`, default today), `account` (optional: code prefix or case-insensitive name fragment) |
| Permissions | `fin:account` `read` |
| Returns | `as_of_date`, `base_currency`, `accounts` (`code`, `name`, `type`, `debit`, `credit`, `net`), `totals` over all matched accounts, `is_balanced` of the whole ledger, `truncated` |

Rules:
- Built on `ReportService.TrialBalance`. Filtering happens in `InsightService`.
- At most 50 accounts are returned. If more matched, `truncated` is `true` and a hint asks the caller to narrow `account`.
- A code prefix is used instead of a tree query so that a branch of the chart of accounts can be requested without assumptions about parent fields.
- No per-currency breakdown in the first batch.

### 5.5 Tool: `fin_find_cashflow`

Answers "find the expenses paid from Cash on Hand last week" and "show draft incomes".

| | |
| --- | --- |
| Arguments | `workspace_id` (required), `type` (required: `income` or `expense`), `date_from`, `date_to` (`YYYY-MM-DD`), `status` (`draft` or `posted`), `query` (text), `payment_account` and `category_account` (code or name), `limit` (default 20, 1 to 50), `cursor` |
| Permissions | `fin:transaction` `read` **and** `fin:account` `read` (account references are resolved) |
| Returns | `items` and `next_cursor` |

Item fields: `id`, `number`, `date`, `type`, `status`, `payment_account` (`code`, `name`), `description`, `reference`, `amount`, `currency`, `base_amount`, and `lines` (`account` with `code` and `name`, `description`, `quantity`, `unit_price`, `amount`). Attachments, audit metadata, workspace id, and entry identifiers are not returned.

Rules:
- `type` maps to `CashFlowService.SearchIncomes` or `SearchExpenses`. A combined `both` is not offered because the two searches use separate cursors.
- Account references resolve first by exact code, then by case-insensitive name fragment. No match returns the existing `ACCOUNT_NOT_FOUND`. More than one match returns a new `ACCOUNT_REFERENCE_AMBIGUOUS` error whose message lists up to five candidates (code and name), registered with `base.AddToErrorMap`.
- `limit` above 50 is rejected with an actionable validation message, not silently clamped. REST allows 100; MCP is stricter to protect client context.
- `cursor` is opaque and passed through. An invalid cursor returns `INVALID_CURSOR`.

### 5.6 Error handling

- Tool errors (validation, not found, forbidden, domain errors such as `PERIOD_NOT_FOUND`) are returned as MCP tool results with `isError: true` and `{code, message}` taken from `httpx.MapError`. This lets the client correct itself.
- Malformed arguments or unknown tool names remain protocol errors raised by the SDK.
- Validation messages must say what is expected, for example `from_month must use the format YYYY-MM`.
- `UNKNOWN` errors are logged with `slog`, following the REST `ErrorHandler`, and returned as a generic message with the request id.
- An invalid or revoked PAT is rejected at the HTTP transport before MCP processing and belongs to sub-project 2.

### 5.7 Output size

- Per-tool bounds: 50 accounts, 50 cashflow items, 24 months.
- The wrapper enforces a serialized size cap. The initial value is **64 KiB** as a constant in code. If exceeded, the result is truncated, `truncated: true` is set, and a hint explains how to narrow the filters. Truncation is never silent.
- Free-text fields (descriptions) are cut to a fixed maximum length.

### 5.8 Phase B (outline): draft input from a receipt

Delivered after the read-only batch and specified in detail in its own document.

- The client reads the receipt (vision) and sends structured data. **In the first release the receipt image is not stored in Omed.** Storing it as an attachment may follow, using the existing `FileService` attachment support.
- Candidate tools: `fin_find_accounts` (helper to find valid category and payment accounts, so the client never guesses ids) and `fin_create_cashflow_draft` (income or expense, multi-item).
- Requirements: always created as `DRAFT`, never posted; permissions `fin:transaction` `create` and `fin:account` `read`; validation of period, currency, and accounts through `CashFlowService`; protection against duplicate drafts when a client repeats a call.
- The duplicate protection mechanism is not decided here (see section 9, item 7).

### 5.9 Impact on existing code, tests, and docs

- New: `fin/model/insight.go`, `fin/service/insight.go`, `fin/mcp/*`, tests for each.
- Changed: `fin/module.go` (build `InsightService`, implement `HasMCP`), `config/contracts.go` (`HasMCP`), `fin/domain/error.go` (`ErrAccountReferenceAmbiguous`).
- Docs: AGENTS.md section 3 gains the `mcp/` layer in the module layout; the README documents the MCP endpoint and the first tools. These updates ship with the implementation, not with this spec.
- Deployment: no new environment variables in this spec. Any enable flag introduced by sub-project 2 must follow AGENTS.md section 6 (Helm chart).

---

## 6. Testing Strategy

- **Go unit tests** (next to the files): `service/insight_test.go` covers defaults, month range validation, previous-period computation, account reference resolution (exact code, name fragment, none, ambiguous), prefix filtering, truncation, and equality with `ReportService.Summary` figures. `mcp/*_test.go` covers argument decoding, defaults, validation messages, and monetary values serialized as strings with a currency code. The mocking pattern follows the existing `fin` service tests; `fin/service/` has no `mock` directory today, so the pattern is confirmed during implementation.
- **Registration tests:** a tool without permissions panics at registration, and every declared permission exists in the role registry (catches typos).
- **Go integration tests** (`internal/fin/test/`), through the `/mcp` endpoint with a PAT, once sub-projects 1 and 2 exist: success for each tool, workspace isolation (non-member receives `NOT_FOUND`), denial for a role that lacks `fin:account` `read` while holding `fin:transaction` `read`, cursor round trip, truncation flag.
- **E2E (Cucumber and Playwright)** is not used because MCP has no UI.

---

## 7. Acceptance Criteria

1. Each of the three tools is registered with a non-empty permission set; registering a tool without one panics.
2. `fin_period_overview` for a range of months returns revenue, expense, and net equal to the corresponding `ReportService.Summary` monthly figures, and `previous` and `delta` are correct for a seeded workspace.
3. `fin_account_balance` with `account = "11"` returns only accounts whose code starts with `11`, with correct totals; without a filter and more than 50 accounts it sets `truncated`.
4. `fin_find_cashflow` returns the expected documents for each combination of `type`, `status`, date range, text query, and account references, and following `next_cursor` yields the remaining items without duplicates.
5. An ambiguous account reference returns `ACCOUNT_REFERENCE_AMBIGUOUS` with candidates; an unknown reference returns `ACCOUNT_NOT_FOUND`; `limit = 51` is rejected.
6. A user who is not a member of the workspace receives `NOT_FOUND`; a member with `fin:transaction` `read` but without `fin:account` `read` receives `FORBIDDEN` from `fin_find_cashflow`.
7. All monetary values are strings, each result carries a currency code, and no result contains attachments, audit metadata, or internal entry ids.
8. A result larger than the size cap is truncated with `truncated: true`.
9. `bun run lint` and `cd apps/api && go test ./...` pass.

---

## 8. What's Left to the Builder

- Whether the monthly revenue and expense computation is shared with `ReportService.Summary` by extracting a helper or by calling it, provided criterion 2 holds.
- The Go names and JSON tags of the insight request and response types, within the conventions of `fin/model`.
- The wording of tool descriptions and argument descriptions. These are read by the client model, so they should be reviewed for clarity and for stating units, defaults, and limits.
- The exact value and enforcement point of the maximum free-text length.
- Date handling for date-only arguments: no time zone is involved, and month boundaries follow the calendar.

---

## 9. Open Items to Verify Before Implementation

1. **Transport (sub-project 1):** the SDK is decided (`github.com/modelcontextprotocol/go-sdk`). Still to confirm is whether its Streamable HTTP handler, an `http.Handler`, integrates with Fiber v3 without a heavy adapter, and whether `Stateless` with `JSONResponse` is enough. `go.mod` currently contains no MCP dependency.
2. **PAT scope (sub-project 2):** whether a PAT is valid only for `/mcp` or also for REST. The middleware in `core/middleware/auth.go` already accepts a Bearer token, so this must be decided deliberately.
3. **`AccountService.Search`:** it accepts an empty `SearchAccountRequest` and returns a cursor. Confirm whether it returns the whole chart of accounts or a page, since account reference resolution depends on it.
4. **`TrialBalanceAccountItem` fields:** confirm that code, name, type, debit, credit, and net are available for the output mapping in section 5.4.
5. **`GetMonthlyAccountFlows`:** confirm that its output supports the same revenue and expense classification as `ReportService.Summary`.
6. **`SaveCashFlowRequest`:** confirm the draft or posted field and its validation rules for phase B.
7. **Duplicate protection for phase B:** an idempotency key may need persistence, which would require a schema change and migration; decide in the phase B spec.
8. **Ledger access:** `GET /workspaces/{wid}/fin/accounts/{id}/ledger` exists in REST. Whether and how it becomes a tool is left to a later batch.
