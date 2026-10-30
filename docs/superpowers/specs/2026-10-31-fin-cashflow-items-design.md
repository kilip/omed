# Design Spec: Multi-Item Income & Expense with Posting / Draft Workflow

- **Date:** 2026-10-31
- **Status:** Draft / Under Review
- **Module:** `fin` (Financial Accounting)
- **Author:** Claude (pair programming with the repository owner)
- **Supersedes:** the "strictly 1-to-1" non-goal (§2) and the single-category request/response contract (§5.2) of [`2026-10-30-fin-cash-income-expense-design.md`](./2026-10-30-fin-cash-income-expense-design.md). Everything else in that spec (launcher, `/fin/overview`, payment account grouping, attachment upload flow) stays in force unless this document says otherwise.

---

## 1. Intent & The Why

Income and Expense quick entry currently records exactly one category line per document and always posts immediately. Real documents are richer: a client invoice has several billable items, a shopping receipt has several purchased goods, each with a quantity and a unit price. Users also need to save work in progress without touching the ledger.

Reference data provided by the owner:

**Income** (invoice label `inv-123456`, deposited to account `Bank Kaltim`):

| Account | Description | Quantity | Unit price | Line total |
| --- | --- | --- | --- | --- |
| 4130 | Konsultasi Improvement | 1 | 1,000,000 | 1,000,000 |
| 4110 | Scaffold base project structure and ci/cd | 6 | 500,000 | 3,000,000 |

Document total: **4,000,000**.

**Expense** (paid from account `Cash on Hand`, no invoice number):

| Account | Description | Quantity | Unit price | Line total |
| --- | --- | --- | --- | --- |
| 6710 | telur | 2 | 55,000 | 110,000 |
| 6710 | indomie | 5 | 3,000 | 15,000 |

Document total: **125,000**.

The frontend must also expose two distinct actions: **Posting** (save and post immediately) and **Draft** (save without posting).

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **Multi-item documents**: an income or expense document has a header and one or more item lines. Each item has an account, a description, a quantity, and a unit price. The line amount is `quantity × unitPrice`, computed on the server.
- **Document label**: income carries an invoice number, stored as the transaction `reference`. Expense keeps an optional `reference` (for example a receipt number) and has no invoice number.
- **Status**: a document is saved as `POSTED` or `DRAFT`. `POSTED` behaves as today. `DRAFT` creates a `DRAFT` journal entry that does not affect the ledger.
- **Draft editing**: a draft can be reopened in the same form through a new edit route and updated with a new `PUT` endpoint. Saving the edit as `POSTED` posts it.
- **Lists show drafts**: income and expense lists include `DRAFT` and `POSTED` documents, show a status tag, and filter by status.
- **Frontend buttons**: **Posting** posts directly, **Draft** saves as draft, on both the income and expense forms.
- **Wider money input**: money inputs fit at least 9 integer digits with grouping separators and the currency's decimals (for example `999.999.999,00`).

### Non-Goals
- **Invoice lifecycle**: no accounts receivable, payment status, or settlement. Income is cash basis: the payment account is debited and revenue is credited at once. The design must not prevent a dedicated invoice document later (see §4, Approach 1).
- **Unit of measure**: items have quantity and unit price only, no unit field.
- **Shorthand input**: values such as `500k` or `1jt` are not parsed. Inputs are plain numbers.
- **Other forms**: the buttons of the manual journal (`/fin/journal/new`) and transaction (`/fin/transactions/new`) forms do not change.
- **Item-level tax or discount.**
- **Schema changes or migrations.**

### Anti-Goals
- Do not bypass accounting invariants: closed periods, active and postable accounts, account currency match, `POSTED` entry immutability, soft deletes, decimal precision.
- Do not produce an unbalanced journal because of rounding (see §5.3, rule 6).
- Do not break existing routes outside `/fin/income*` and `/fin/expense*`.

---

## 3. Constraints

- **Zero schema changes.** `fin_posting.quantity` and `fin_posting.unit_price` (`numeric(20,4)`, nullable) and `fin_transaction.reference` (indexed with `workspace_id`) already exist. No Ent regeneration and no migration is required.
- **Layering** (AGENTS.md §3): `controller -> service -> repository`. Controllers never touch Ent. Only repositories talk to the database.
- **API contract**: Swaggo annotations are updated, then `apps/api/docs/*` and `packages/openapi/src/api.ts` are regenerated and committed. This is a breaking change to the `/fin/income` and `/fin/expense` endpoints; the web app is their only consumer in this repository.
- **Money**: `shopspring/decimal` in Go, decimal strings on the wire. No floating point arithmetic on amounts in the backend.
- **Frontend** (AGENTS.md §4): types come from `@omed/openapi`; design tokens from `BRANDING.md`, no hardcoded hex values; Biome formatting; unit tests next to the file under test; i18n keys in both `id` and `en`.
- **Commits**: Conventional Commits with the emoji mapping and a required scope (`api`, `web`, `e2e`, `docs`). One logical change per commit.

---

## 4. Approaches Considered

### Approach 1: Dedicated cashflow document tables
New `fin_cashflow` (header) and `fin_cashflow_item` tables linked to `transaction` and `entry`.
- *Pros*: clean place for header fields; natural base for a later invoice lifecycle.
- *Cons*: two new tables and a migration; draft items would live outside the journal, so the same data exists twice (document items and postings).

### Approach 2: Reuse `transaction` + `entry` + `posting` (Selected by the owner)
Items are stored as postings that carry `quantity` and `unit_price`. The header lives on `transaction` and `entry`.
- *Pros*: no schema change; ledger, trial balance, and reports are unaffected; the draft state is simply a `DRAFT` entry that the existing journal tooling already understands.
- *Cons*: the posting table carries two presentation-oriented columns (already present today); a later invoice lifecycle will need its own table, and this design keeps the data shape compatible with that (header on `transaction`, lines on `posting`).

### Approach 3: Items as JSON on `fin_transaction`
- *Cons*: items cannot be queried or reported, and the JSON would duplicate the postings. Rejected.

---

## 5. Detailed Design

### 5.1 Data mapping

| Concept | Storage |
| --- | --- |
| Document | `fin_transaction` (`source = MANUAL`) |
| Invoice number / reference | `fin_transaction.reference` |
| Description | `fin_transaction.description` and `fin_entry.description` |
| Status | `fin_entry.status` (`DRAFT` or `POSTED`). Exactly one entry per document. |
| Payment line | `fin_posting`, `position = 0`. Income: `DEBIT`. Expense: `CREDIT`. `amount` is the sum of item amounts. `quantity` and `unit_price` are `NULL`. |
| Item line `i` | `fin_posting`, `position = i` (1-based). Income: `CREDIT`. Expense: `DEBIT`. `account_id` is the item account. `quantity`, `unit_price`, `amount = quantity × unit_price`, `description`. |
| Attachments | `core` files linked with `EntityType = "fin:transaction"` and `EntityID = transaction.id` |

The flow type of an existing document is detected from its item postings (`REVENUE` account means income, `EXPENSE` account means expense), as `toCashFlowDomain` does today.

### 5.2 API contract

All endpoints are under `/workspaces/{wid}/fin`.

| Method and path | Purpose | Authz |
| --- | --- | --- |
| `POST /income`, `POST /expense` | Create (`201`) | `ObjTransaction` `create` |
| `GET /income/{id}`, `GET /expense/{id}` | Read one document (`200`). A document of the other type returns `404 TRANSACTION_NOT_FOUND`. | `ObjTransaction` `read` |
| `PUT /income/{id}`, `PUT /expense/{id}` | Replace a draft (`200`). A document whose entry is not `DRAFT` returns `409 ENTRY_NOT_DRAFT`. | `ObjTransaction` `update` |
| `POST /income/search`, `POST /expense/search` | Search (existing, extended) | `ObjTransaction` `read` |
| `DELETE /transactions/{id}` | Delete a draft (existing). Documents with posted entries return `409 TRANSACTION_HAS_POSTED_ENTRIES`. | `ObjTransaction` `delete` |

Saving with `status = POSTED` through `POST` or `PUT` requires only the `ObjTransaction` permission of that endpoint, which matches the current behavior where income and expense creation posts directly.

**Create / replace request** (income shown; expense is identical except the field `reference` is optional and described as a receipt number). IDs are illustrative:

```json
{
  "paymentAccountId": 10002,
  "transactionDate": "2026-10-31",
  "reference": "inv-123456",
  "description": "Consulting and scaffolding for Client X",
  "currency": "IDR",
  "exchangeRate": "1",
  "status": "POSTED",
  "items": [
    { "accountId": 40130, "description": "Konsultasi Improvement", "quantity": "1", "unitPrice": "1000000" },
    { "accountId": 40110, "description": "Scaffold base project structure and ci/cd", "quantity": "6", "unitPrice": "500000" }
  ],
  "attachmentFileIds": []
}
```

| Field | Rules |
| --- | --- |
| `paymentAccountId` | required |
| `transactionDate` | required |
| `status` | optional, `DRAFT` or `POSTED`, default `POSTED` |
| `items` | required, 1 to 100 items |
| `items[].accountId` | required |
| `items[].quantity` | required, greater than 0 |
| `items[].unitPrice` | required, not negative. A line whose computed amount is zero is rejected by service rule 2. |
| `currency`, `exchangeRate`, `description`, `reference` | optional, same semantics as today |
| `attachmentFileIds` | optional. On `PUT` it lists additional files to link; existing attachments are untouched and are removed with the existing `DELETE /transactions/{id}/attachments/{fileId}`. |

The fields `revenueAccountId`, `expenseAccountId`, and `amount` are removed.

**Response** (`CashFlowItemResponse`): `categoryAccount` is removed and `items` is added. `amount` and `baseAmount` become the document totals. `paymentAccount`, `reference`, `entryId`, `entryNumber`, and `entryStatus` are unchanged.

```json
{
  "id": 1,
  "number": "TRX-2026-000001",
  "type": "INCOME",
  "reference": "inv-123456",
  "paymentAccount": { "id": 10002, "code": "1120", "name": "Bank Kaltim", "type": "ASSET" },
  "amount": "4000000",
  "baseAmount": "4000000",
  "entryStatus": "POSTED",
  "items": [
    {
      "id": 11,
      "account": { "id": 40130, "code": "4130", "name": "Consulting & Advisory Income", "type": "REVENUE" },
      "description": "Konsultasi Improvement",
      "quantity": "1",
      "unitPrice": "1000000",
      "amount": "1000000",
      "baseAmount": "1000000"
    }
  ]
}
```

**Search** (`SearchCashFlowRequest`) gains `status` (`DRAFT` or `POSTED`; omitted means both). `VOID` entries stay excluded. `categoryAccountId` matches when any item uses that account.

### 5.3 Service rules (`CashFlowService`)

`createCashFlow` becomes a shared `saveCashFlow` used by create and update. Validation order:

1. `items` has between 1 and 100 entries; each `quantity` is positive (`ErrPostingInvalidQuantity`) and each `unitPrice` is not negative (`ErrPostingInvalidUnitPrice`).
2. The line amount is `quantity × unitPrice`, rounded to the decimals of the document currency (looked up with `currencyRepo.GetByCode`, capped at the 4-decimal column precision, and falling back to 4 when the currency is unknown). It must be positive (`ErrPostingInvalidAmount`).
3. The transaction date resolves to an existing period (`ErrPeriodNotFound`) that is `OPEN` (`ErrPeriodClosed`). This applies to both statuses, because `fin_entry.period_id` is required. Posting an edited draft re-checks it.
4. The payment account is an active, postable `ASSET` account. Every item account is active, postable, and of type `REVENUE` (income) or `EXPENSE` (expense). Violations return `ErrPostingAccountInactive`, `ErrPostingAccountNotPostable`, or `ErrAccountInvalidType`, as today.
5. Account currency, when set, must equal the document currency (`ErrPostingCurrencyMismatch`), checked for the payment account and every item account.
6. **Rounding**: `base_amount` is computed per item as `round(amount × rate, baseDecimals)`. The payment line's `amount` is the sum of item amounts and its `base_amount` is the sum of item `base_amount`s. Debits equal credits in both the document and the base currency by construction.
7. **Status**: `DRAFT` creates the entry with status `DRAFT` and no `posted_at` or `posted_by`. `POSTED` sets both within the same database transaction.
8. **Update**: the service loads the document, verifies its type, and requires the entry to be `DRAFT` (`ErrEntryNotDraft`). It updates the header, recomputes the period from the new date, deletes all postings, and recreates them with positions `0..n`, in one database transaction. Transaction and entry numbers are preserved. When `status = POSTED`, it sets `posted_at` and `posted_by`.
9. **Attachments**: files are validated and linked exactly as today (`fileSvc.GetByID`, then `fileSvc.Update` with `fin:transaction`). Linking happens on `PUT` as well, before the entry is posted.

### 5.4 Repository (`CashFlowRepository`)

- `CreateCashFlowParams` replaces `CategoryAccountID` and `Amount` with `Items []CashFlowItemParams` (`AccountID`, `Description`, `Quantity`, `UnitPrice`, `Amount`, `BaseAmount`) and adds `Status`, `TotalAmount`, and `TotalBaseAmount`.
- `CreatePosted` is renamed `Create`. It writes the transaction, the entry (status per request), the payment posting at position 0, and the item postings, in one `entx.WithTx`, keeping the existing retry on sequence number collision.
- New `Update(ctx, id, params)` and public `GetByID(ctx, flowType, id)`. `getByID` already exists and becomes exported through the contract.
- `Search` replaces `StatusEQ(POSTED)` with `StatusIn(requested or {DRAFT, POSTED})`. The payment and category posting predicates keep working with multiple items.
- `toCashFlowDomain` builds `Items` from the item-side postings (ordered by `position`), takes `paymentAccount` and totals from the payment posting, and keeps the type detection.
- `CashFlowItem` (domain) drops `CategoryAccount` and adds `Items []CashFlowLine`.

### 5.5 Frontend (`apps/web`)

**Routes** (`modules/fin/route.ts`, following the `transactions` convention; `new` is registered before `:id`):

| Route | Content |
| --- | --- |
| `income`, `expense` | List with a status column (`entry-status-tag.tsx`) and a status filter. The category column shows the first item account and a `+N` badge when there are more items. |
| `income/new`, `expense/new` | Create form |
| `income/:id`, `expense/:id` | Read-only detail: header, items table, totals, attachments. A `DRAFT` document shows **Edit**, **Posting**, and **Delete**. **Posting** here calls the existing `POST /entries/{entryId}/post` (`ObjEntry` `update`), because no edited payload is involved; **Delete** calls `DELETE /transactions/{id}`. |
| `income/:id/edit`, `expense/:id/edit` | The same form, prefilled from `GET`. A non-draft document redirects to `:id`. |

**Shared form**: `income-new.tsx` and `expense-new.tsx` are near-identical (366 lines each). They are replaced by one `components/cashflow-form.tsx`, parameterized by flow type (labels, item account type filter, invoice number visible for income only), used by the four create/edit routes. `submitIncomeForm` and `submitExpenseForm` become one `submitCashflowForm` that takes `status`.

**Form layout**:
- Header: payment account (grouped by `groupPaymentAccounts`), date, currency, invoice number (income) or reference (expense), description.
- Items table built with `Form.List`: item account (searchable `code - name`), description, quantity, unit price, line total (read-only, computed from the watched quantity and unit price), and a remove button. One empty row exists initially and the last row cannot be removed. **Add item** appends a row.
- Document total below the table, followed by the accounting preview (one payment line and one line per item) and the attachment picker.
- The form container is widened from `max-w-3xl` so the five-column table fits.

**Buttons**: **Posting** (primary) submits with `status = POSTED`. **Draft** (default style) submits with `status = DRAFT`. **Cancel** navigates back. Both submit buttons are disabled while a request is pending, and only the clicked one shows the loading state. Drafts are fully validated (required header fields and at least one valid item); there are no incomplete drafts. After a successful save the app navigates to the list.

**`MoneyInput`** (`components/money-input.tsx`): wraps Ant Design `InputNumber` with a grouping `formatter` and `parser` driven by `getActiveDecimalStyle()` and `getCurrencyDecimals(currency)` from `~/lib/format`. The unit price and line total columns are sized to show `999.999.999` plus decimals without truncation. The quantity column stays narrow.

**i18n**: new keys for item columns, **Posting**, **Draft**, status filter, add/remove item, and totals in `i18n/en.ts` and `i18n/id.ts`.

### 5.6 Impact on existing code, tests, and docs
- Backend: `model/cashflow.go`, `domain/cashflow.go`, `service/cashflow.go`, `controller/cashflow.go`, `repository/cashflow.go`, `routes.go`, and their tests are updated.
- Frontend: `income.tsx`, `expense.tsx`, `income-expense.test.tsx`, `cashflow-helpers.ts`, `queries.ts` (`getIncome`, `getExpense` query options), `route.ts`.
- Generated: `apps/api/docs/*` and `packages/openapi/src/api.ts`.
- Documentation: the Finance route list in `README.md`.

---

## 6. Testing Strategy

- **Go unit tests** (next to the files): service rules 1 to 9 (positive and negative cases), rounding with a non-trivial exchange rate (debits equal credits in both currencies), draft versus posted entry fields, update replacing postings, update rejected on a posted document, type mismatch on `GET` and `PUT`.
- **Go integration tests** (`internal/fin/test/`): the two reference documents of §1 end to end (create, `GET`, search with and without `status`, `PUT` draft to posted, delete draft, delete posted rejected), and trial balance totals after posting.
- **Web unit tests** (Vitest): `submitCashflowForm` payload for both statuses, line total and document total computation, `MoneyInput` formatting for 9 digits in both decimal styles, button behavior (which status each button sends).
- **E2E** (Cucumber + Playwright, `e2e/features/fin/`): income with two items posted; expense saved as draft, reopened, and posted. Page objects live in `pages/fin/`.

## 7. Acceptance Criteria

1. Creating the income document of §1 with `status = POSTED` produces one `POSTED` entry with three postings: debit `Bank Kaltim` 4,000,000; credit 4130 1,000,000; credit 4110 3,000,000. The trial balance reflects it.
2. Creating the expense document of §1 with `status = DRAFT` produces a `DRAFT` entry with the postings in the expected direction, and the ledger and trial balance are unchanged.
3. The draft appears in `/fin/expense` with a draft tag, opens at `/fin/expense/:id/edit`, and the **Posting** button posts it.
4. A `PUT` to a posted document returns `409 ENTRY_NOT_DRAFT`.
5. A unit price of `999999999` with quantity `1` is fully visible in the input and survives a round trip.
6. `bun run lint`, `bun run typecheck`, `bun run test`, `cd apps/api && go test ./...`, and `bun run e2e` pass.

## 8. What's Left to the Builder

- Exact query keys and loader structure for the new `GET` routes, following the existing `incomeKeys` and `expenseKeys` in `queries.ts`.
- Column widths and responsive behavior of the items table, within the design tokens of `BRANDING.md`.
- Wording of the new i18n strings.
