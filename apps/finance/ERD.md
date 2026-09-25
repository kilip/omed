# ERD — apps/finance (Personal Accounting, Double-Entry, Standard COA)

Scope: double-entry bookkeeping, per-Workspace (tenant), budgeting, attachment/receipt. Aligned to standard accounting practice (GnuCash / QuickBooks / Beancount patterns).

## Diagram

```mermaid
erDiagram
    WORKSPACE ||--o{ ACCOUNT : owns
    WORKSPACE ||--o{ JOURNAL : owns
    WORKSPACE ||--o{ BUDGET : owns
    WORKSPACE ||--o{ ACCOUNTING_PERIOD : owns
    WORKSPACE ||--o{ USER_REF : "has members"

    ACCOUNT ||--o{ ACCOUNT : "parent_of (COA hierarchy)"
    ACCOUNT ||--o{ JOURNAL_LINE : "posted_to"
    ACCOUNT ||--o{ ACCOUNT_BALANCE_SNAPSHOT : "snapshotted_by"
    ACCOUNT ||--o{ BUDGET : "budgeted_for (income/expense account)"

    JOURNAL ||--|{ JOURNAL_LINE : "has (>=2, balanced)"
    JOURNAL ||--o{ ATTACHMENT : "has"
    JOURNAL }o--o| EXCHANGE_RATE : "uses (if cross-currency)"
    JOURNAL }o--|| USER_REF : "created_by"
    JOURNAL |o--o| JOURNAL : "reversed_by (self-ref, nullable)"
    ATTACHMENT }o--|| USER_REF : "uploaded_by"

    ACCOUNT {
        uuid id PK
        uuid workspace_id FK
        uuid parent_id FK "nullable, self-ref (COA tree)"
        string code "COA number, e.g. 1000, 1100"
        string name
        enum account_type "asset|liability|equity|income|expense"
        enum subtype "cash|bank|ewallet|credit_card|payable|receivable (nullable)"
        enum normal_balance "debit|credit — derived from account_type"
        string currency
        bool is_placeholder "true = group/header, no direct posting"
        bool is_system "true = system account e.g. Opening Balance Equity"
        bool is_active
        timestamp created_at
        timestamp updated_at
    }

    ACCOUNTING_PERIOD {
        uuid id PK
        uuid workspace_id FK
        date period_start
        date period_end
        enum status "open|closed"
        timestamp closed_at "nullable"
        uuid closed_by FK "-> USER_REF, nullable"
    }

    JOURNAL {
        uuid id PK
        uuid workspace_id FK
        int journal_no "sequential per workspace"
        date txn_date
        string description
        string reference_no "nullable, external ref"
        enum status "draft|posted|void"
        uuid reversed_by_journal_id FK "nullable, self-ref"
        uuid exchange_rate_id FK "nullable, set if cross-currency"
        uuid created_by FK "-> USER_REF"
        timestamp created_at
        timestamp updated_at
    }

    JOURNAL_LINE {
        uuid id PK
        uuid journal_id FK
        uuid account_id FK
        decimal debit "native account currency"
        decimal credit "native account currency"
        decimal base_debit "converted to workspace base_currency"
        decimal base_credit "converted to workspace base_currency"
        string memo "nullable"
    }

    BUDGET {
        uuid id PK
        uuid workspace_id FK
        uuid account_id FK "must be an income/expense account"
        enum period_type "monthly|yearly"
        date period_start
        date period_end
        decimal amount
        timestamp created_at
    }

    ACCOUNT_BALANCE_SNAPSHOT {
        uuid id PK
        uuid workspace_id FK
        uuid account_id FK
        date snapshot_date
        decimal balance
        timestamp created_at
    }

    EXCHANGE_RATE {
        uuid id PK
        string from_currency
        string to_currency
        decimal rate
        date rate_date
        timestamp created_at
    }

    ATTACHMENT {
        uuid id PK
        uuid journal_id FK
        string file_url
        string file_name
        string mime_type
        bigint size_bytes
        uuid uploaded_by FK "-> USER_REF"
        timestamp created_at
    }

    USER_REF {
        uuid id PK "= auth.User.id, no cross-DB FK constraint"
        uuid workspace_id FK
        string name
        string email
        string avatar_url "nullable"
        timestamp synced_at "last sync from apps/auth"
    }
```

## Business Rules

1. **Unified Chart of Accounts**: no separate `Category` table — Income & Expense are `Account` rows (`account_type = income|expense`), same tree as Asset/Liability/Equity.
2. **Account hierarchy**: `parent_id` builds the COA tree. `is_placeholder = true` marks group/header accounts — only leaf accounts receive `JOURNAL_LINE` postings.
3. **normal_balance**: `asset`/`expense` = debit-normal; `liability`/`equity`/`income` = credit-normal.
4. **subtype** (`cash|bank|ewallet|credit_card|payable|receivable`) only applies to `asset`/`liability` accounts; `null` otherwise.
5. **No `opening_balance` field.** Opening balance is a normal `Journal`: debit the new account, credit a system `Opening Balance Equity` account (`account_type = equity`, `is_system = true`). Keeps every balance traceable purely from `JOURNAL_LINE` — no out-of-band number breaking the double-entry invariant.
6. **Balanced entry**: `SUM(debit) = SUM(credit)` per `JOURNAL`, minimum 2 `JOURNAL_LINE` rows.
7. **Transfer**: `JOURNAL` between two asset/liability accounts, no income/expense touched — no special table needed.
8. **Budget**: scoped to an income/expense `Account` + period; actual spend computed from `JOURNAL_LINE` at query time.
9. **Balance calculation**: `ACCOUNT_BALANCE_SNAPSHOT` written periodically; current balance = latest snapshot + `SUM(lines)` posted after `snapshot_date`.
10. **Multi-currency**: `Workspace.base_currency` + dated `EXCHANGE_RATE`. `JOURNAL_LINE` stores native + base-currency amounts; `Journal.exchange_rate_id` is the audit trail of which rate was applied.
11. **Journal lifecycle**: `draft` (editable, not counted) → `posted` (immutable, counted in balances) → `void`. Voiding **never deletes or edits a posted journal** — it creates a reversing `Journal` and links it via `reversed_by_journal_id`, preserving full audit history.
12. **Period locking**: `ACCOUNTING_PERIOD` with `status = closed` blocks new/edited postings with `txn_date` inside that period — standard books-closing control.
13. **User reference**: `apps/finance` does not FK directly into `apps/auth`'s DB (different service/DB). `USER_REF` is a local shadow table, synced from `apps/auth` (event-driven or lazy-cache), used only for display/audit — never for auth/permission decisions (Casbin remains source of truth for that).
14. All top-level entities carry `workspace_id` — enforced by Casbin RBAC as domain, consistent with existing apps/finance pattern.
