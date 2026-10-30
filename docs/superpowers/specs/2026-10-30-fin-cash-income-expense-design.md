# Design Spec: Income & Expense Quick Entry (`/fin/income` & `/fin/expense`)

- **Date:** 2026-10-30
- **Status:** Draft / Under Review
- **Module:** `fin` (Financial Accounting)
- **Author:** Antigravity (Pair Programming with Pak Bos)

---

## 1. Intent & The Why

In Omed's current `fin` module, recording transactions requires manually balancing debit and credit entries using `PostingsEditor`. While this provides complete double-entry rigor, everyday operational transactions—such as buying office supplies at Indomaret with cash or receiving client invoice payments into a bank account—are cumbersome for business owners and operational staff who are not accountants.

The goal of this feature is to introduce dedicated **Income (`/fin/income`)** and **Expense (`/fin/expense`)** modules that simplify recording into a **1-to-1 Quick Entry** workflow while automatically generating balanced double-entry records (`fin_transaction`, `fin_entry`, `fin_posting`) behind the scenes.

Additionally, navigation in the `fin` module is restructured:
- `/fin` becomes a sub-launcher grid (similar to the root `/` app launcher), presenting visual tiles for all financial menus.
- The existing financial dashboard is relocated from `/fin` to `/fin/overview`.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **Dedicated Income & Expense Routes**:
  - `/fin/income`: Income history list, filters, total income summary, and action button to `/fin/income/new`.
  - `/fin/income/new`: 1-to-1 Income form (Deposit To account: Tunai/E-Wallet/Bank, Revenue category account, amount, date, description, reference, attachment).
  - `/fin/expense`: Expense history list, filters, total expense summary, and action button to `/fin/expense/new`.
  - `/fin/expense/new`: 1-to-1 Expense form (Paid From account: Tunai/E-Wallet/Bank, Expense category account, amount, date, description, reference, receipt attachment).
- **Automated Double-Entry**: The backend atomically constructs and posts balanced debit/credit entries to the general ledger (`status: POSTED`).
- **Receipt / Attachment Support**: Direct upload of receipt photos or transfer proof linked atomically before entry posting.
- **Launcher Hub (`/fin`)**: Displays an Odoo-style app tile grid linking to Overview, Income, Expense, CoA, Periods, Opening Balance, Transactions, Journal, Ledger, and Trial Balance.
- **Relocate Overview (`/fin/overview`)**: Existing `/fin` dashboard cleanly moved to `/fin/overview`.

### Non-Goals
- **Multi-split lines in Income/Expense Quick Entry**: Strictly 1-to-1 (one payment account against one revenue/expense account). Users needing complex multi-line splits will use `/fin/transactions/new`.
- **Separate Sub-Ledger**: No separate disconnected database tables. All movements live in the core `fin` double-entry tables (`fin_transaction`, `fin_entry`, `fin_posting`).
- **Bank Reconciliation / Feed Import**: Automatic bank statement sync is out of scope for this spec.

### Anti-Goals
- **Do not bypass accounting invariants**: Income and Expense entries must respect closed periods, multi-schema workspace boundaries, soft-deletes, decimal precision, and `POSTED` entry immutability.
- **Do not break existing routes**: Deep links to `/fin/transactions/*`, `/fin/journal/*`, `/fin/accounts`, etc., remain unchanged.

---

## 3. Constraints

- **Language & Tech Stack**:
  - Backend: Go 1.23+, Fiber v3, Ent ORM, PostgreSQL schema `fin`, `shopspring/decimal`.
  - Frontend: React Router v7, React 19, TypeScript (strict), Ant Design 5, Biome linter/formatter.
  - API Contracts: OpenAPI via Swaggo (`apps/api/docs`), generated client in `@omed/openapi`.
- **Database Architecture**: Ent multi-schema (`entsql.Schema("fin")`). Zero schema migrations needed; reuse `fin_account`, `fin_transaction`, `fin_entry`, `fin_posting`, and `core.files`.
- **Immutability of POSTED Entries**: Once saved, quick-entry transactions are `POSTED`. Corrections follow standard accounting reversals.

---

## 4. Approaches Considered

### Approach 1: Pure Frontend Abstraction
- *Description*: Frontend computes debit/credit lines and calls existing `POST /workspaces/{wid}/fin/transactions` with `inlineEntry`.
- *Pros*: Zero backend changes, no OpenAPI regeneration.
- *Cons*: Business logic (validating payment account is ASSET, category is EXPENSE/REVENUE, single 1-to-1 balance, and atomic posting) is enforced only in the client. Moreover, existing `POST /fin/transactions` only creates `DRAFT` inline entries, requiring extra round-trips to upload attachments and post the entry.

### Approach 2: Dedicated Go Backend Endpoints + Frontend (Selected by Pak Bos)
- *Description*: Create explicit backend endpoints under `/workspaces/{wid}/fin/income` and `/workspaces/{wid}/fin/expense`. Server validates account classification, links uploaded receipt files, and atomically generates the transaction, `POSTED` entry, and balanced postings in a single database transaction.
- *Pros*: Domain integrity guaranteed at API level; single atomic operation for creation + attachment linking + posting; clean semantics for web and future mobile clients.
- *Cons*: Requires Go controller/service implementation, OpenAPI spec update, and Go unit/integration tests.
- *Decision*: **Selected Approach 2** for data integrity and atomic lifecycle handling.

---

## 5. Detailed Architecture & Design Decisions

### 5.1 Payment Account Selection (No Schema Flag)
- No database column or flag is added to `fin_account`.
- In Omed's Chart of Accounts, liquid payment accounts live under `1110` (Kas & E-Wallet), `1120` (Bank Operasional), and `1130` (Platform & Gateway), all with `type == "ASSET"` and `is_postable == true`.
- In the UI selector, postable `ASSET` accounts are grouped by their parent account (putting Tunai, E-Wallet, and Bank groups at the top, and excluding non-liquid assets like Fixed Assets `12xx`, Accounts Receivable `114x`, and Prepaid Tax `115x`).
- Backend validates that `paymentAccountId` is an active, postable `ASSET` account (`type == "ASSET" && is_active && is_postable`), and the category account is an active, postable `EXPENSE` or `REVENUE` account.

### 5.2 Backend Architecture (`apps/api`)

#### Attachment & Posting Lifecycle:
In `TransactionService`, `UploadAttachment` blocks adding files once an entry has `status == POSTED` (`ErrTransactionHasPostedEntries`). To respect this invariant without requiring multiple round-trips:
1. If the user attaches receipt files in `/fin/expense/new` or `/fin/income/new`, the frontend first uploads each file via the existing core endpoint `POST /workspaces/{wid}/files`, receiving `FileResponse.id`.
2. The frontend sends `attachmentFileIds: [id1, ...]` in the create expense/income request.
3. The backend service creates `fin_transaction`, links the files to the transaction via `fileSvc.Update` (`EntityType: "fin:transaction"`, `EntityID: tx.ID`), and creates the `fin_entry` with `status = POSTED` and 2 balanced `fin_posting` rows.

#### New Endpoints in `internal/fin/controller/`:
Following the `apps/api/internal/fin/routes.go` conventions (where search endpoints use `POST .../search` and monetary values use `decimal.Decimal` string representation):

1. **Expense Endpoints (`/workspaces/{wid}/fin/expense`)**:
   - `POST /workspaces/{wid}/fin/expense` (`CreateExpense`):
     - **Authz**: `az.Require(domain.ObjTransaction, "create")`
     - **Request Body** (`CreateExpenseRequest`):
       ```json
       {
         "paymentAccountId": 10001,
         "expenseAccountId": 50001,
         "amount": "75000",
         "transactionDate": "2026-10-30",
         "description": "Belanja di Indomaret (kertas & snack)",
         "reference": "STRUK-1234",
         "currency": "IDR",
         "exchangeRate": "1",
         "attachmentFileIds": [123]
       }
       ```
     - **Validation**:
       - `amount` > 0 (`decimal.Decimal`).
       - `transactionDate` falls within an `OPEN` fiscal period in `fin_period` (auto-resolved by date).
       - `paymentAccountId` exists in workspace, `is_active && is_postable`, `type == "ASSET"`.
       - `expenseAccountId` exists in workspace, `is_active && is_postable`, `type == "EXPENSE"`.
     - **Generated Double Entry (Atomic DB Transaction)**:
       - `fin_transaction`: date, description, reference, currency, exchangeRate.
       - Link `attachmentFileIds` to `fin_transaction.id`.
       - `fin_entry`: `status = POSTED`, `posted_at = now`, `posted_by = userId`, auto-generated entry number.
       - `fin_posting` 1 (Debit): `account_id = expenseAccountId`, `direction = DEBIT`, `amount`.
       - `fin_posting` 2 (Credit): `account_id = paymentAccountId`, `direction = CREDIT`, `amount`.
     - **Response**: `httpx.Created(c, CashFlowItemResponse)` (`201 Created`).
   - `POST /workspaces/{wid}/fin/expense/search` (`SearchExpenses`):
     - **Authz**: `az.Require(domain.ObjTransaction, "read")`
     - **Request Body** (`SearchCashFlowRequest`): `dateFrom`, `dateTo`, `paymentAccountId`, `categoryAccountId`, `query`, `cursor`, `limit`.
     - **Response**: `httpx.WithCursor(c, []CashFlowItemResponse, cursor)` returning paginated expense transactions (where an `EXPENSE` debit posting is paired with an `ASSET` credit posting) along with `paymentAccount`, `categoryAccount`, `amount`, `attachments`.

2. **Income Endpoints (`/workspaces/{wid}/fin/income`)**:
   - `POST /workspaces/{wid}/fin/income` (`CreateIncome`):
     - **Authz**: `az.Require(domain.ObjTransaction, "create")`
     - **Request Body** (`CreateIncomeRequest`):
       ```json
       {
         "paymentAccountId": 10002,
         "revenueAccountId": 40001,
         "amount": "15000000",
         "transactionDate": "2026-10-30",
         "description": "Pembayaran jasa dari Company X",
         "reference": "INV-2026-001",
         "currency": "IDR",
         "exchangeRate": "1",
         "attachmentFileIds": [456]
       }
       ```
     - **Validation**:
       - `amount` > 0.
       - `transactionDate` falls within an `OPEN` fiscal period.
       - `paymentAccountId` exists in workspace, `is_active && is_postable`, `type == "ASSET"`.
       - `revenueAccountId` exists in workspace, `is_active && is_postable`, `type == "REVENUE"`.
     - **Generated Double Entry (Atomic DB Transaction)**:
       - `fin_posting` 1 (Debit): `account_id = paymentAccountId`, `direction = DEBIT`, `amount`.
       - `fin_posting` 2 (Credit): `account_id = revenueAccountId`, `direction = CREDIT`, `amount`.
     - **Response**: `httpx.Created(c, CashFlowItemResponse)` (`201 Created`).
   - `POST /workspaces/{wid}/fin/income/search` (`SearchIncomes`):
     - **Authz**: `az.Require(domain.ObjTransaction, "read")`
     - **Request Body** (`SearchCashFlowRequest`): `dateFrom`, `dateTo`, `paymentAccountId`, `categoryAccountId`, `query`, `cursor`, `limit`.
     - **Response**: `httpx.WithCursor(c, []CashFlowItemResponse, cursor)` returning paginated income transactions (where an `ASSET` debit posting is paired with a `REVENUE` credit posting) along with `paymentAccount`, `categoryAccount`, `amount`, `attachments`.

---

### 5.3 Frontend Architecture (`apps/web`)

#### Route Changes in `apps/web/app/modules/fin/route.ts`:
1. `index("modules/fin/routes/launcher.tsx")` -> Mounted at `/fin` (Module launcher grid).
2. `route("overview", "modules/fin/routes/dashboard.tsx")` -> Mounted at `/fin/overview` (Relocated dashboard).
3. `route("income", "modules/fin/routes/income.tsx")` -> Mounted at `/fin/income` (Income history list).
4. `route("income/new", "modules/fin/routes/income-new.tsx")` -> Mounted at `/fin/income/new` (Create 1-to-1 Income).
5. `route("expense", "modules/fin/routes/expense.tsx")` -> Mounted at `/fin/expense` (Expense history list).
6. `route("expense/new", "modules/fin/routes/expense-new.tsx")` -> Mounted at `/fin/expense/new` (Create 1-to-1 Expense).

#### Sub-Launcher (`modules/fin/routes/launcher.tsx`):
- Odoo-style app tile grid matching `apps/web/app/routes/launcher.tsx` (`home-backdrop`, `.app-tile`, `.app-tile--1..3`).
- Displays tiles for all items in `fin`:
  1. **Overview** (`/fin/overview`)
  2. **Income** (`/fin/income`)
  3. **Expense** (`/fin/expense`)
  4. **Bagan Akun / CoA** (`/fin/accounts`)
  5. **Periode Fiskal** (`/fin/periods`)
  6. **Saldo Awal** (`/fin/opening-balance`)
  7. **Transaksi** (`/fin/transactions`)
  8. **Jurnal Umum** (`/fin/journal`)
  9. **Buku Besar** (`/fin/ledger`)
  10. **Neraca Saldo** (`/fin/trial-balance`)

#### Top Navbar Menu (`modules/fin/module.ts`):
- Updated `menu` in `apps/web/app/modules/fin/module.ts`:
  - `overview`: `path: "overview"`, `labelKey: "fin.menu.overview"`, `icon: DashboardOutlined`
  - `income`: `path: "income"`, `labelKey: "fin.menu.income"`, `icon: ArrowDownOutlined` (or `RiseOutlined`)
  - `expense`: `path: "expense"`, `labelKey: "fin.menu.expense"`, `icon: ArrowUpOutlined` (or `FallOutlined`)
  - `accounting`: `labelKey: "fin.menu.accounting"`, `icon: AuditOutlined`, `children`:
    - `accounts` (`path: "accounts"`)
    - `periods` (`path: "periods"`)
    - `opening-balance` (`path: "opening-balance"`)
    - `transactions` (`path: "transactions"`)
    - `journal` (`path: "journal"`)
    - `ledger` (`path: "ledger"`)
    - `trial-balance` (`path: "trial-balance"`)

#### Income & Expense Pages:
1. **`/fin/income` (`modules/fin/routes/income.tsx`)**:
   - **Header & Action**: Title "Pemasukan / Income" + primary button `+ Catat Pemasukan` (`/fin/income/new`).
   - **Filter Bar**: Date range picker, Payment Account selector (Tunai / E-Wallet / Bank), Revenue Category selector, Search input.
   - **History Table**: Tanggal, No. Transaksi (links to `/fin/transactions/:id`), Rekening Penerima (`paymentAccount`), Kategori Pendapatan (`categoryAccount`), Keterangan, Referensi, Nominal, Lampiran.
2. **`/fin/income/new` (`modules/fin/routes/income-new.tsx`)**:
   - Uses `FormView` layout:
     - **Deposit To (`paymentAccountId`)**: Grouped postable `ASSET` accounts (`Kas & E-Wallet`, `Bank Operasional`, `Platform & Gateway`).
     - **Income Category (`revenueAccountId`)**: Postable `REVENUE` accounts.
     - **Amount**, **Transaction Date** (default today), **Description**, **Reference**, **Attachments** (`PendingAttachmentsPicker`).
     - **Live Double-Entry Preview**: Read-only box showing `(D) Payment Account` / `(K) Revenue Account`.
3. **`/fin/expense` (`modules/fin/routes/expense.tsx`)**:
   - **Header & Action**: Title "Pengeluaran / Expense" + primary button `+ Catat Pengeluaran` (`/fin/expense/new`).
   - **Filter Bar**: Date range picker, Payment Account selector (Tunai / E-Wallet / Bank), Expense Category selector, Search input.
   - **History Table**: Tanggal, No. Transaksi (links to `/fin/transactions/:id`), Bayar Pakai (`paymentAccount`), Kategori Beban (`categoryAccount`), Keterangan, Referensi, Nominal, Lampiran Struk.
4. **`/fin/expense/new` (`modules/fin/routes/expense-new.tsx`)**:
   - Uses `FormView` layout:
     - **Paid From (`paymentAccountId`)**: Grouped postable `ASSET` accounts (`Kas & E-Wallet`, `Bank Operasional`, `Platform & Gateway`).
     - **Expense Category (`expenseAccountId`)**: Postable `EXPENSE` accounts.
     - **Amount**, **Transaction Date** (default today), **Description**, **Reference**, **Receipt Attachments** (`PendingAttachmentsPicker`).
     - **Live Double-Entry Preview**: Read-only box showing `(D) Expense Account` / `(K) Payment Account`.

#### Impact on Existing Tests & Redirects:
- Update existing tests (`e2e/pages/fin/dashboard.page.ts`, `e2e/features/fin/*.feature`, web unit tests) that assumed `/fin` rendered the dashboard directly to use `/fin/overview` or navigate via the `/fin` launcher.

---

## 6. What's Left to the Builder

- Shared UI components between `income.tsx` / `expense.tsx` and `income-new.tsx` / `expense-new.tsx`.
- Translation keys in `apps/web/app/i18n/en.ts` and `id.ts`.
- Unit, integration, and E2E test updates.
