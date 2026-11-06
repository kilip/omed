# Design: Talk To Agent Chat for Receipt Analysis and Draft Entries

Status: Approved. Date: 2026-11-06. Target branch: `feat/issue-63` (GitHub Issue #63).

## 1. Purpose & Motivation

Recording business transactions—such as multi-line shop purchases, vendor invoices, or ATM cash withdrawals—is often tedious and error-prone when accountants or business operators must manually transcribe dates, reference codes, line items, and ledger accounts.

While the stateless Financial Agent ([`2026-11-02-financial-agent-attachment-analysis-design.md`](./2026-11-02-financial-agent-attachment-analysis-design.md)) introduced prefilling for the single-document Income and Expense forms, users frequently have complex source documents (such as ATM cash withdrawal slips or multi-item receipts) that require discussion, contextual clarification, or different voucher types (e.g. double-entry journal vouchers).

**Talk To Agent** provides an interactive, persisted multi-turn chat directly in the Finance (`fin`) module. Users can converse with the workspace's configured AI agent, upload images or PDF documents (receipts, invoices, slips), and receive structured, pre-validated draft transaction proposals rendered as interactive preview cards in the chat stream.

The user remains in full control: they can review line items, edit fields inline, and explicitly confirm or discard the proposal. A confirmed draft creates an **unposted draft transaction** in `fin`; the agent never posts entries or mutates financial state on its own.

### Success Criteria

- An authorized user can access Talk To Agent under `/fin/agent` when the workspace has an active financial agent provider.
- Sending an image or PDF attachment immediately triggers analysis and tool-calling execution within the chat.
- The model leverages workspace reference tools (`list_accounts`, `list_currencies`, `list_periods`, `propose_draft`) to resolve Chart of Accounts IDs, currencies, and accounting periods.
- Reference case 1 (ATM withdrawal slip): produces a balanced double-entry Journal draft (Dr Cash on Hand / Cr Bank account).
- Reference case 2 (Shop receipt): produces a multi-item Expense draft matching line item totals.
- Zero-write AI boundary: `propose_draft` outputs a preview card only; no record is created in `fin` until the user clicks "Confirm".
- Draft-only persistence: Confirmed proposals save strictly as draft transactions in `fin`, never posted to ledgers.
- Chat conversations and uploaded attachments persist across browser reloads.
- Deleting a conversation removes all associated attachment objects from S3 storage and soft-deletes database records.
- If the agent is disabled, misconfigured, rate-limited, or the user lacks permissions, graceful feedback is shown without impacting standard finance features.

## 2. Scope

### In Scope

- **Database Schemas & Migrations:** Ent schemas in PostgreSQL `agent` schema (`agent_chat_conversation`, `agent_chat_message`, `agent_chat_attachment`) with workspace isolation, audit mixins, soft delete, and Atlas versioned migration.
- **Backend Architecture (`apps/api`):**
  - Domain models, error definitions, and chat repository in `internal/agent/repository/chat.go`.
  - Tool definitions and handlers (`list_accounts`, `list_currencies`, `list_periods`, `propose_draft`) conforming to `llmx.Tool`.
  - Server-side validation logic for draft proposals across expense, income, and journal kinds.
  - Bounded multi-turn tool calling loop in `ChatService` with configurable timeout and iteration limits.
  - Cross-module integration via `FinanceDraftContext` interface implemented over `fin` services in `config/finance_draft_context.go`.
  - Object storage integration (`s3x.Storage`) for attachment uploads, downloads, and deletion cleanup.
  - REST API controllers and routes under `/workspaces/:wid/agent/chat/*`.
  - OpenAPI specification and generated TypeScript definitions.
- **Frontend Architecture (`apps/web`):**
  - Route `/fin/agent` and launcher/navigation menu entry in the `fin` module.
  - Chat layout: `ConversationSidebar`, `MessageList`, `MessageComposer`, and `DraftPreviewCard`.
  - Inline editing of proposed draft fields (date, account, currency, items, entries).
  - Draft state lifecycle: Proposed -> Confirmed (with direct link to created transaction) or Discarded.
  - Dual-language localization (`en`, `id`).
- **Testing:** Unit tests, integration tests (`chat_test.go`), and end-to-end Cucumber + Playwright scenarios.

### Out of Scope

- Auto-posting entries to the General Ledger or any unconfirmed mutation of financial data.
- Creating a separate "Transfer" transaction type (transfers and ATM withdrawals map to balanced journal vouchers).
- Analysis of spreadsheet files (`.csv`, `.xls`, `.xlsx`).
- Background auto-expiry or automated TTL pruning jobs for chat conversations.
- Streaming responses (chunked server-sent events or WebSocket); uses standard request/response HTTP.
- Modifying the existing stateless Income/Expense form prefill flow.

## 3. Architectural Decisions

| # | Decision | Rationale |
| --- | --- | --- |
| **D1** | **Persisted multi-turn chat in `agent` schema with workspace and user ownership scoping.** | Chat history, messages, and attachment metadata live in the `agent` PostgreSQL schema (`agent_chat_conversation`, `agent_chat_message`, `agent_chat_attachment`). Conversations are owned by the creator (`created_by`) and isolated per workspace. |
| **D2** | **Stored attachments in S3 workspace bucket with cleanup on conversation deletion.** | Files sent in chat are uploaded to S3-compatible storage (`s3x.Storage`) under `workspaces/:wid/agent/chat/:uuid.:ext`. To prevent orphaned objects and uncontrolled storage growth, deleting a conversation purges all linked S3 objects before soft-deleting database records. |
| **D3** | **Relaxed D6/D7 from 2026-11-02: Model has tools (`list_accounts`, `list_currencies`, `list_periods`, `propose_draft`).** | The earlier design spec (`2026-11-02`) prohibited tools (D6) and restricted output to form prefill (D7). Talk To Agent explicitly relaxes this: the model is given access to read-only workspace reference tools and a `propose_draft` tool to build structured voucher suggestions. |
| **D4** | **Zero-write model guarantee: `propose_draft` validates and outputs a preview only.** | The `propose_draft` tool does NOT insert or update records in `fin`. It validates references and returns a structured preview DTO attached to the assistant message. Persistence to `fin` occurs strictly via an explicit user confirmation endpoint (`POST /messages/:id/confirm`) with complete server-side re-validation. |
| **D5** | **Draft-only persistence: Confirmed drafts always save as DRAFT transactions in `fin`, never posted.** | When the user confirms a proposal, `FinanceDraftContext.SaveDraft` creates cashflow or transaction records strictly in `DRAFT` status. Ledger posting remains an explicit, manual action in `fin`. |
| **D6** | **Reference cases: ATM withdrawal slip and Shop receipt.** | Reference cases drive acceptance testing: (1) An ATM cash withdrawal slip maps to a balanced double-entry Journal draft (Dr Cash on Hand / Cr Bank account). (2) A store receipt maps to a multi-item Expense draft where individual line items sum to the document total. |
| **D7** | **Retention: Manual deletion only (no background auto-expiry).** | Users retain complete control over conversation history. Deletion is initiated manually by the user. No background TTL jobs, cron workers, or extra configuration environment variables are introduced. |
| **D8** | **Non-streaming request/response first.** | Messages with text and optional attachments are sent via standard multipart HTTP `POST`. The backend runs the tool loop synchronously within request deadlines (default 60s timeout) and returns the completed assistant reply and draft proposal. |

## 4. Data Model

All chat tables are located in the `agent` PostgreSQL schema, leveraging Ent mixins (`WorkspaceMixin`, `AuditMixin`, and `SoftDeleteMixin`).

```
┌─────────────────────────────────┐
│     AgentChatConversation      │
├─────────────────────────────────┤
│ id: int64 (PK)                  │
│ workspace_id: int64             │
│ title: string (optional)        │
│ created_by: int64               │
│ created_at / updated_at         │
│ deleted_at: timestamp (soft)    │
└────────────────┬────────────────┘
                 │ 1
                 │
                 │ has many
                 ▼ *
┌─────────────────────────────────┐       1 ┌───────────────────────────────┐
│        AgentChatMessage         ├─────────┤      AgentChatAttachment      │
├─────────────────────────────────┤ has many├───────────────────────────────┤
│ id: int64 (PK)                  │ *       │ id: int64 (PK)                │
│ conversation_id: int64 (FK)     │         │ conversation_id: int64 (FK)   │
│ workspace_id: int64             │         │ message_id: int64 (FK, opt)   │
│ role: enum                      │         │ workspace_id: int64           │
│   (user, assistant, system, tool)         │ object_key: string            │
│ content: text                   │         │ bucket: string                │
│ parts: jsonb (optional)         │         │ mime: string                  │
│ draft_status: enum              │         │ size: int64                   │
│   (none, proposed,              │         │ file_name: string             │
│    confirmed, discarded)        │         │ created_at / updated_at       │
│ confirmed_transaction_id: int64 │         └───────────────────────────────┘
│ created_at / updated_at         │
└─────────────────────────────────┘
```

### Table Definitions

1. **`agent_chat_conversation` (`agent.agent_chat_conversations`)**
   - `id`: `int64` primary key.
   - `workspace_id`: `int64` workspace scope.
   - `title`: optional conversation title (e.g., initial user prompt or document name).
   - `created_by`: user ID of the conversation owner.
   - `deleted_at`: timestamp for soft delete.
   - Indexes: `(workspace_id, deleted_at)`, `(workspace_id, created_by)`.

2. **`agent_chat_message` (`agent.agent_chat_messages`)**
   - `id`: `int64` primary key.
   - `conversation_id`: `int64` foreign key referencing `agent_chat_conversations(id)` with cascade delete.
   - `workspace_id`: `int64` workspace scope.
   - `role`: enum (`user`, `assistant`, `system`, `tool`).
   - `content`: text content of the message.
   - `parts`: optional JSON payload holding tool call history or draft proposal structures.
   - `draft_status`: enum (`none`, `proposed`, `confirmed`, `discarded`), default `none`.
   - `confirmed_transaction_id`: optional `int64` pointing to the created draft transaction in `fin`.
   - Indexes: `(workspace_id, conversation_id)`.

3. **`agent_chat_attachment` (`agent.agent_chat_attachments`)**
   - `id`: `int64` primary key.
   - `conversation_id`: `int64` foreign key referencing `agent_chat_conversations(id)` with cascade delete.
   - `message_id`: optional `int64` foreign key referencing `agent_chat_messages(id)` with cascade delete.
   - `workspace_id`: `int64` workspace scope.
   - `object_key`: S3 storage key (e.g., `workspaces/1/agent/chat/<uuid>.jpg`).
   - `bucket`: S3 bucket name.
   - `mime`: MIME type (sniffed server-side; image or PDF).
   - `size`: file size in bytes.
   - `file_name`: original uploaded file name.
   - Indexes: `(workspace_id, conversation_id)`, `(workspace_id, message_id)`.

## 5. Tool Calling Architecture & Draft Validation

### 5.1 Available Tools

The model is equipped with four tools adhering to the `llmx.Tool` specification:

1. **`list_accounts`**
   - Input: `{}`
   - Output: List of active, postable accounts in the workspace (`id`, `code`, `name`).
2. **`list_currencies`**
   - Input: `{}`
   - Output: List of active workspace currencies (`code`, `name`, `symbol`, `decimals`).
3. **`list_periods`**
   - Input: `{}`
   - Output: List of accounting periods (`id`, `name`, `startDate`, `endDate`, `fiscalYear`, `status`).
4. **`propose_draft`**
   - Input schema accepts:
     - `kind`: `"expense"` | `"income"` | `"journal"` (required)
     - `transactionDate`: `"YYYY-MM-DD"` (required)
     - `description`: string (required)
     - `reference`: optional string
     - `currency`: currency code (required)
     - `exchangeRate`: optional decimal string
     - `paymentAccountId`: integer account ID (required for expense and income)
     - `periodId`: optional integer period ID (used for journal entries)
     - `totalAmount`: decimal string
     - `items`: array of line items (`accountId`, `description`, `quantity`, `unitPrice`, `amount`) for expense/income
     - `entries`: array of journal postings (`accountId`, `direction` `"DEBIT"`/`"CREDIT"`, `amount`, `description`) for journal drafts

### 5.2 Server-Side Validation Rules (`ValidateDraftProposal`)

Validation is strictly enforced on every `propose_draft` call and re-executed during confirmation:

- **Accounts:** Every referenced account ID (`paymentAccountId`, `items[].accountId`, `entries[].accountId`) must exist in the workspace and be active and postable.
- **Currencies:** `currency` must match an active workspace currency code (case-insensitive).
- **Dates:** `transactionDate` must parse as `YYYY-MM-DD` and fall within plausible business date boundaries.
- **Periods:** If `periodId` is supplied, it must identify an active accounting period.
- **Expense & Income Drafts:**
  - `paymentAccountId` is mandatory.
  - At least 1 item is required.
  - Quantities must be `> 0`; unit prices must be `>= 0`.
  - Item amount defaults to `quantity * unitPrice`.
  - The sum of item amounts must equal `totalAmount` (tolerating rounding differences up to 0.01).
- **Journal Drafts:**
  - At least 2 entries required.
  - Must include at least one `DEBIT` and at least one `CREDIT`.
  - Double-entry balance check: Total Debits must strictly equal Total Credits (`sum(Debit) == sum(Credit)`).
  - Total amount equals the balanced debit/credit sum.
- **Untrusted Document Defense:**
  - System prompts instruct the LLM that attachment text constitutes untrusted user data, not operational commands.
  - Prompt injections embedded in document text cannot execute arbitrary actions because tool arguments are validated against workspace entities, and the model has zero write privileges.

### 5.3 Tool Loop Execution

```
User Message + Attachment
          │
          ▼
    Store in S3
          │
          ▼
   Assemble Context & System Prompt
   (Inject LLM tools: list_accounts, list_currencies, etc.)
          │
    ┌─────┴────────────────────────┐
    ▼                              │
LLM Turn ◄─────────────────┐       │
    │                      │       │
    ├─► Tool Call?         │       │
    │   (e.g. list_accounts)│      │
    │   Execute Tool ──────┘       │ (Iteration cap: max 6)
    │                              │
    └─► Tool Call: propose_draft?  │
        Validate proposal          │
        Generate preview DTO       │
        End Loop ──────────────────┘
          │
          ▼
Persist Assistant Message (status: "proposed", parts: DraftProposal)
Return response to Client
```

## 6. User Confirmation Workflow & Zero-Write Security Boundary

### 6.1 The Zero-Write Guarantee

At no point in the tool execution loop can the LLM insert, update, or delete records in the `fin` database. The `propose_draft` tool only returns a validated `DraftProposal` object.

```
                    ┌─────────────────────────┐
                    │      Model / Agent      │
                    └────────────┬────────────┘
                                 │
                 Calls propose_draft(args)
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │ Server-Side Validation  │
                    │   (No DB writes made)   │
                    └────────────┬────────────┘
                                 │
                     Returns Preview in Chat
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ User UI (apps/web): DraftPreviewCard                            │
│                                                                 │
│ - Inspect line items / entries                                  │
│ - Edit fields inline (date, account, items, amounts)            │
│ - Actions: [ Confirm Draft ]  or  [ Discard ]                   │
└────────────────────────────────┬────────────────────────────────┘
                                 │
               User clicks "Confirm Draft"
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ POST /workspaces/:wid/agent/chat/messages/:id/confirm           │
│                                                                 │
│ 1. Verify message exists and draft_status == "proposed"         │
│ 2. Re-validate proposal (including any inline edits)            │
│ 3. FinanceDraftContext.SaveDraft(proposal)                      │
│    -> fin.CreateExpense / fin.CreateIncome / fin.CreateTrans.   │
│    -> ALWAYS EntryStatusDraft / draft transaction               │
│ 4. Update message: draft_status="confirmed",                    │
│    confirmed_transaction_id=tx.ID                               │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Confirmation State Machine

- `none`: Standard conversational message without a proposal.
- `proposed`: Assistant message contains a validated draft proposal. UI renders action buttons (Confirm / Discard) and inline edit controls.
- `confirmed`: User confirmed the draft. Database transaction created in `fin`. UI renders confirmed badge and navigation link to `/fin/transactions/:id` or cashflow detail. Subsequent confirmation requests return `409 Conflict` (`DRAFT_ALREADY_CONFIRMED`).
- `discarded`: User clicked Discard (`POST /messages/:id/discard`). The draft proposal is archived and cannot be confirmed (`409 Conflict` `DRAFT_ALREADY_DISCARDED`).

### 6.3 Cross-Module Isolation (`FinanceDraftContext`)

To preserve clean architectural boundaries as mandated by `AGENTS.md` (`controller -> service -> repository`), the `agent` module never accesses `fin` database tables or repositories directly.

`agent/service` declares the consumer interface:

```go
type FinanceDraftContext interface {
    ListAccounts(ctx context.Context) ([]AccountOption, error)
    ListCurrencies(ctx context.Context) ([]CurrencyOption, error)
    ListPeriods(ctx context.Context) ([]PeriodOption, error)
    SaveDraft(ctx context.Context, proposal model.DraftProposal) (int64, error)
}
```

The adapter in `apps/api/internal/config/finance_draft_context.go` implements this interface using `fin` public domain services:
- `finMod.AccountService`
- `finMod.CurrencyService`
- `finMod.PeriodService`
- `finMod.CashFlowService` (invoking `CreateExpense` / `CreateIncome` with `EntryStatusDraft`)
- `finMod.TransactionService` (invoking `Create` with balanced journal postings)

## 7. S3 Attachment Lifecycle & Cleanup

1. **Upload:** User attaches an image (`.png`, `.jpg`, `.jpeg`, `.webp`) or PDF (`.pdf`) in the message composer.
2. **Edge Validation:** Fiber controller validates file size against `OMED_ATTACHMENT_MAX_SIZE_MB` and verifies MIME type using byte-sniffing (`http.DetectContentType`).
3. **Storage:** ChatService uploads the payload to S3 via `s3x.Storage.PutObject` using object key `workspaces/<wid>/agent/chat/<uuid>.<ext>` within the workspace bucket.
4. **Attachment Linking:** An `agent_chat_attachment` record is created, referencing both the conversation and user message.
5. **Download:** Authorized users can retrieve attachments via `GET /workspaces/:wid/agent/chat/attachments/:id/download`, which streams directly from S3.
6. **Purge on Conversation Deletion:**
   When `DELETE /workspaces/:wid/agent/chat/conversations/:id` is invoked:
   - Query all attachments linked to the conversation within the workspace.
   - For each attachment, call `storage.DeleteObject(ctx, att.Bucket, att.ObjectKey)`.
   - Soft-delete conversation and cascade-delete messages/attachment records.

## 8. Permissions & Security

### Dual Permission Model

Operating the financial chat requires two distinct permissions:
1. `agent:financial` with action `use` (held by role `agent:financial-user` or `agent:admin`).
2. `fin:transaction` with action `create` (held by `fin:accountant` or `fin:manager`).

This prevents privilege escalation: an agent user cannot inspect financial accounts or draft transactions without financial transaction creation authority, and a finance user cannot trigger LLM consumption without explicit agent feature authorization.

### Ownership & Scope

- Every request resolves the authenticated workspace from URL path `:wid`.
- Conversations are filtered by `workspace_id = :wid` and `created_by = user.ID`. Users cannot view or delete other members' conversations.

## 9. API Specification

All routes are grouped under `/workspaces/:wid/agent/chat`:

| Method | Path | Permissions Required | Description |
| --- | --- | --- | --- |
| `GET` | `/conversations` | `agent:financial:use` + `fin:transaction:create` | List conversations (paginated: `page`, `limit`). |
| `POST` | `/conversations` | `agent:financial:use` + `fin:transaction:create` | Create a new conversation (`title`). |
| `GET` | `/conversations/:id` | `agent:financial:use` + `fin:transaction:create` | Get conversation details with full message history and attachments. |
| `DELETE` | `/conversations/:id` | `agent:financial:use` + `fin:transaction:create` | Soft-delete conversation and purge its S3 attachments. |
| `POST` | `/conversations/:id/messages` | `agent:financial:use` + `fin:transaction:create` | Multipart send: `content` text + `files` (image/PDF). Runs tool loop and returns assistant response. |
| `POST` | `/messages/:id/confirm` | `agent:financial:use` + `fin:transaction:create` | Confirm proposed draft (with optional edited proposal payload). Returns `{ transactionId }`. |
| `POST` | `/messages/:id/discard` | `agent:financial:use` + `fin:transaction:create` | Discard proposed draft. |
| `GET` | `/attachments/:id/download` | `agent:financial:use` + `fin:transaction:create` | Stream attachment binary data from S3. |

### Error Codes

- `CONVERSATION_NOT_FOUND` (404): Conversation does not exist or was deleted.
- `MESSAGE_NOT_FOUND` (404): Message does not exist.
- `ATTACHMENT_NOT_FOUND` (404): Attachment does not exist or is inaccessible.
- `DRAFT_ALREADY_CONFIRMED` (409): Proposal has already been converted to a transaction.
- `DRAFT_ALREADY_DISCARDED` (409): Proposal has already been discarded.
- `DRAFT_NOT_PROPOSED` (422): Message contains no active proposal to confirm or discard.
- `INVALID_DRAFT_PROPOSAL` (422): Proposal failed validation checks (unbalanced entries, invalid IDs).
- `FINANCIAL_AGENT_DISABLED` (409): Workspace setting has disabled the financial agent.
- `FINANCIAL_AGENT_NO_PROVIDER` (422): No valid provider configured or provider missing required capabilities (`ToolCalling`, `Vision`, `Documents`).
- `UNSUPPORTED_DOCUMENT` (422): Configured provider cannot process the uploaded document format.
- `RATE_LIMITED` (429): User exceeded requests-per-minute chat capacity.

## 10. Frontend Architecture

### 10.1 Routes & Navigation

- **Module Route:** `/fin/agent` configured in `apps/web/app/modules/fin/route.ts` pointing to `routes/chat.tsx`.
- **Launcher Tile:** Added to `finLauncherItems` in `apps/web/app/modules/fin/routes/launcher.tsx` under key `"talk-to-agent"`.
- **Navigation Menu:** Registered in `apps/web/app/modules/fin/module.ts` menu with `RobotOutlined` icon.

### 10.2 Component Hierarchy

```
routes/chat.tsx
└── ChatLayout
    ├── ConversationSidebar (list, create new, delete with popconfirm)
    └── ActiveChatArea
        ├── MessageList
        │   ├── UserMessageItem (text + attachment preview badges)
        │   └── AssistantMessageItem
        │       ├── Markdown Content
        │       └── DraftPreviewCard
        │           ├── ExpenseVariant (items list, total, payment account)
        │           ├── IncomeVariant (items list, total, deposit account)
        │           ├── JournalVariant (balanced debit/credit table)
        │           ├── Inline Edit Modal / Controls
        │           └── Actions ([Confirm], [Discard], [View Transaction])
        └── MessageComposer
            ├── Attachment Dropzone & Thumbnail Preview
            ├── Multiline Text Input
            └── Submit Button (triggers upload + processing indicator)
```

### 10.3 Internationalization (i18n)

Complete localized copy is provided in English (`en.ts`) and Indonesian (`id.ts`) under namespaces:
- `fin.chat.*`: Conversation controls, titles, confirmation dialogs, placeholders, and error messages.
- `fin.chat.draft.*`: Card titles, line item headers, debit/credit badges, confirmation statuses, and validation warnings.

## 11. Testing & Verification

1. **Go Unit Testing (`mise run test:api`):**
   - Repository tests (`internal/agent/repository/chat_test.go`): CRUD, soft delete, message ordering, workspace isolation.
   - Tool validation tests (`internal/agent/service/tools_test.go`): validation of ATM withdrawal slip (journal) and shop receipt (expense); detection of unbalanced entries, missing accounts, and invalid totals.
   - Service unit tests (`internal/agent/service/chat_test.go`): memory adapter tool-calling loop, attachment upload, draft confirmation, duplicate confirmation rejection, and S3 cleanup on conversation deletion.
   - Adapter tests (`internal/config/finance_draft_context_test.go`): verifies `SaveDraft` creates drafts strictly in unposted state.
2. **Web Unit & Component Testing (`mise run test`):**
   - Hook tests (`apps/web/app/modules/fin/hooks/use-agent-chat.test.ts`).
   - Launcher route tests (`apps/web/app/modules/fin/routes/launcher.test.tsx`).
   - Component tests (`apps/web/app/modules/fin/components/chat/draft-preview-card.test.tsx`): rendering expense, income, and journal proposals; verifying edit, confirm, and discard actions.
3. **End-to-End Testing (`mise run e2e`):**
   - Feature file: `e2e/features/fin/talk-to-agent.feature`.
   - Scenarios:
     - Analyzing an ATM slip produces a journal voucher proposal; confirming creates an unposted draft transaction.
     - Analyzing a multi-line receipt produces an expense voucher proposal with matching item amounts.
     - Discarding a proposal prevents transaction creation.
     - Chat messages and attachments persist after reloading the browser.
     - Users lacking `agent:financial` or `fin:transaction` permissions cannot access the chat.
