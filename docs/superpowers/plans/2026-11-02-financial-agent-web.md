# Financial Agent Web Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a workspace admin switch the Financial Agent on and pick its provider, and make the Income/Expense form analyze every attachment as it is added and prefill the form for the user to review.

**Architecture:** The `agent` module gains a settings page, queries, a permission flag, and a `useFinancialAnalysis` hook built on a framework-free sequential queue. The `fin` module consumes that hook (same cross-module direction as the backend), adds pure prefill helpers, and extends `PendingAttachmentsPicker` with per-file status and a manual button. When the agent is unavailable the form renders exactly as it does today.

**Tech Stack:** React Router, Ant Design, TanStack React Query, `openapi-fetch` with types from `@omed/openapi`, i18next, Vitest, Biome.

**Spec:** `docs/superpowers/specs/2026-11-02-financial-agent-attachment-analysis-design.md` (sections 7 and 8). Prerequisite: `docs/superpowers/plans/2026-11-02-financial-agent-backend.md` Task 10 is merged, so `packages/openapi/src/api.ts` contains `/workspaces/{wid}/agent/settings`, `/workspaces/{wid}/agent/financial/status`, `/workspaces/{wid}/agent/financial/analyze`, and `model.FinancialSuggestion`.

## Global Constraints

- Commits follow AGENTS.md: `type(scope): <emoji> subject`, scope `web` (or `docs`).
- API types come from `@omed/openapi`. Do not hand-write response types.
- Style with the design tokens from `BRANDING.md` (for example `var(--omed-info)`, `var(--omed-border)`). No hardcoded hex values.
- Unit tests sit next to the file under test. Verification commands: `cd apps/web && bunx vitest run <path>`, `bun run typecheck`, `bun run lint`.
- Strings live in both `apps/web/app/i18n/en.ts` and `apps/web/app/i18n/id.ts`, under `agent.financial.*` and `fin.cashflow.ai.*`.
- Only analyzable files are queued: `.png`, `.jpg`, `.jpeg`, `.webp`, `.pdf` (same set as `isPreviewableAttachment`). CSV, XLS, XLSX stay attachable and are never analyzed.
- Client queue: concurrency 1, upload order, one `AbortController` per file, aborted on file removal and on unmount. A 429 pauses the queue; remaining files keep their manual button.
- Auto-analyze runs on every analyzable file added, in create and edit mode, only when the status query reports `available: true`.
- Default prefill fills a scalar only when it is empty or `form.isFieldTouched` is false. "Analisa ulang" runs after a `Popconfirm` and overwrites every non-null scalar and replaces the items with that file's items. A `payment_proof` adds or replaces no items.
- Total mismatch tolerance is 0.01. The check is skipped when the suggestion currency differs from the form currency.
- Failure never blocks the form. The agent never posts: the user still presses Posting or Draft.
- The settings page shows a plain notice that every uploaded document is sent to the configured LLM provider.
- The status and settings queries use a 60 second stale time and are enabled only when the user can use or manage the agent.

## Review Focus

1. A file is removed while its analysis is running or queued: the request is aborted, the result is discarded, and no form field changes (tested in Task 4).
2. The user typed or picked a value before the result arrived: default mode never overwrites it, and a result arriving after unmount or after submit starts is discarded (tested in Tasks 3 and 6).
3. The user has no agent permission, the agent is disabled, or the provider has no vision: no status or analyze call is made, no AI UI is rendered, and the form is unchanged (tested in Tasks 1 and 6).
4. The same file is added twice, or several files contain the same line items: duplicates are dropped and only one blank row is ever replaced (tested in Tasks 3 and 6).
5. Rate limited, provider unavailable, timeout, unsupported PDF, and an empty result: each shows its own message, the form stays editable, and Posting and Draft still work (tested in Task 6).
6. "Analisa ulang" on a `payment_proof` leaves the existing items untouched, and a total mismatch for a different currency raises no warning (tested in Task 3).

---

### Task 1: Agent queries, permissions, analyze client

**Files:**
- Modify: `apps/web/app/modules/agent/queries.ts`, `apps/web/app/modules/agent/hooks/use-agent-permissions.ts`
- Test: `apps/web/app/modules/agent/hooks/use-agent-permissions.test.ts`, `apps/web/app/modules/agent/queries.test.ts` (create)

**Interfaces:**
- Produces in `queries.ts`:
  ```ts
  export type AgentSetting = components["schemas"]["model.SettingResponse"];
  export type FinancialStatus = components["schemas"]["model.FinancialStatus"];
  export type FinancialSuggestion = components["schemas"]["model.FinancialSuggestion"];
  agentKeys.settings(wid), agentKeys.financialStatus(wid)
  settingsQueryOptions(wid)            // staleTime 60_000
  financialStatusQueryOptions(wid)     // staleTime 60_000
  useUpdateSettings(wid)               // PATCH /settings, invalidates settings and financialStatus
  analyzeFinancialDocument(wid: number, flow: "INCOME" | "EXPENSE", file: File, signal?: AbortSignal): Promise<FinancialSuggestion>
  ```
  `analyzeFinancialDocument` sends multipart `file` + `flow` with the same body pattern as `uploadTransactionAttachment` and rethrows `ApiError` with the API error code intact.
- Produces in permissions: `AGENT_ROLES.FINANCIAL_USER = "agent:financial-user"`; `evaluateAgentPermissions` returns `canManageSettings` (admin, or `agent:setting` `update`) and `canUseFinancial` (admin, `agent:financial-user` role, or `agent:financial` `use`).

- [ ] **Step 1: Write failing tests.** Permissions: owner and `agent:admin` get both flags; a viewer gets neither; `agent:financial-user` gets `canUseFinancial` only; a raw `{object: "agent:financial", action: "use"}` grant gets `canUseFinancial`. Queries: `analyzeFinancialDocument` posts a `FormData` with `file` and `flow` and returns the unwrapped body; an aborted signal rejects with an abort error; a 429 response rejects with an `ApiError` whose `code` is `RATE_LIMITED`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/agent`. Expected: FAIL.
- [ ] **Step 3: Implement** the types, keys, query options, mutation, and the two permission flags.
- [ ] **Step 4: Run** the same command plus `bun run typecheck`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ add financial agent queries and permissions`.

### Task 2: Agent settings page and provider Documents toggle

**Files:**
- Create: `apps/web/app/modules/agent/routes/settings.tsx`, `apps/web/app/modules/agent/routes/settings.test.tsx`
- Modify: `apps/web/app/modules/agent/route.ts`, `module.ts`, `routes/index.tsx` (launcher link), `components/provider-form.tsx`, `components/provider-form.test.tsx`, `apps/web/app/i18n/en.ts`, `id.ts`

**Interfaces:**
- Consumes: `settingsQueryOptions`, `useUpdateSettings`, `providersQueryOptions`, `useAgentPermissions(...).canManageSettings` (Task 1).
- Produces: route `/agent/settings` (`clientLoader` returns `{ wid }` via `requireWorkspaceId`, like `providers.tsx`); a menu entry `settings` in `module.ts`; `DEFAULT_CAPABILITIES_BY_KIND` gains `documents` (`true` for `anthropic` and `gemini`, `false` for `openai_compat`).

- [ ] **Step 1: Write failing tests.**
  - Settings page: renders a Switch reflecting `financialEnabled`; a Select with "Provider default" plus every workspace provider; the privacy notice text is visible; saving sends `{ financialEnabled, financialProviderId }` with `null` for "Provider default"; a `FINANCIAL_AGENT_NO_PROVIDER` error shows a form-level error and leaves the Switch off; a user without `canManageSettings` sees a read-only page or is redirected like the providers page does.
  - Provider form: a "Documents (PDF)" toggle exists, defaults follow the kind, and its value is submitted inside `capabilities.documents`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/agent`. Expected: FAIL.
- [ ] **Step 3: Implement** the page with Ant Design `Form`, `Switch`, `Select`, and an info `Alert` for the notice, register the route and menu item, add the launcher link, add the toggle beside the existing capability toggles, and add `en` and `id` strings under `agent.financial.*` and `agent.menu.settings`.
- [ ] **Step 4: Run** `cd apps/web && bunx vitest run app/modules/agent` and `bun run typecheck`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ add financial agent settings page`.

### Task 3: Prefill helpers (pure functions)

**Files:**
- Modify: `apps/web/app/modules/fin/components/cashflow-helpers.ts`, `cashflow-helpers.test.ts`

**Interfaces:**
- Consumes: `FinancialSuggestion` from `~/modules/agent/queries`, existing `CashflowItemValue`, `computeTotal`.
- Produces:
  ```ts
  export type PrefillMode = "default" | "overwrite";
  export type PrefillScalar = "transactionDate" | "paymentAccountId" | "currency" | "exchangeRate" | "reference" | "description";
  export interface PrefillResult {
    values: Partial<{ transactionDate: Dayjs; paymentAccountId: number; currency: string; exchangeRate: number; reference: string; description: string; items: CashflowItemValue[] }>;
    filledFields: string[];            // scalar names plus "items" when items changed
  }
  export function applyAnalysis(args: {
    current: { [K in PrefillScalar]?: unknown } & { items: CashflowItemValue[] };
    isTouched: (field: PrefillScalar) => boolean;
    suggestion: FinancialSuggestion;
    mode: PrefillMode;
  }): PrefillResult;
  export function mergeItems(existing: CashflowItemValue[], incoming: NonNullable<FinancialSuggestion["items"]>): CashflowItemValue[];
  export function checkTotalMismatch(args: { suggestion: FinancialSuggestion; formTotal: number; formCurrency: string }): { fileTotal: number; compared: number } | null;
  ```

- [ ] **Step 1: Write failing tests** (one per rule in spec 7.3):
  - Default mode fills an empty scalar, fills an untouched scalar that holds a default (date, currency, exchange rate), and leaves a touched scalar alone even when the suggestion differs.
  - Null suggestion fields change nothing in either mode.
  - `mergeItems`: a form holding only the blank default row (`{quantity: 1, unitPrice: 0}` with no account or description) is replaced; a form with real rows appends; an incoming item equal to an existing one (same `accountId`, description compared case-insensitively, `quantity`, `unitPrice`) is skipped; two identical incoming items from different calls produce one row; incoming `accountId: null` stays unset and quantity and unit price are converted from strings to numbers.
  - `payment_proof` suggestion: `applyAnalysis` returns no items in either mode and `filledFields` does not contain `"items"`.
  - Overwrite mode: every non-null scalar is applied regardless of touched state, items are replaced with the suggestion's items, and a `payment_proof` leaves the existing items untouched.
  - `checkTotalMismatch`: for a `payment_proof`, compares `documentTotal` with `formTotal`; for another type, compares `documentTotal` with the sum of the suggestion's own items; a difference of 0.01 returns `null` and 0.02 returns the pair; a suggestion currency different from `formCurrency` returns `null`; a missing `documentTotal` returns `null`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin/components/cashflow-helpers.test.ts`. Expected: FAIL.
- [ ] **Step 3: Implement** the three functions. `applyAnalysis` converts `transactionDate` with `dayjs`, `exchangeRate` with `Number`, and builds `filledFields` from what actually changed.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ add financial agent prefill helpers`.

### Task 4: Analysis queue and `useFinancialAnalysis`

**Files:**
- Create: `apps/web/app/modules/agent/hooks/analysis-queue.ts`, `analysis-queue.test.ts`, `use-financial-analysis.ts`, `use-financial-analysis.test.tsx`
- Modify: `apps/web/app/modules/agent/hooks/index.ts`

**Interfaces:**
- Produces in `analysis-queue.ts` (no React):
  ```ts
  export type JobStatus = "queued" | "analyzing" | "done" | "failed" | "paused";
  export interface Job { id: string; file: File; status: JobStatus; errorCode?: string }
  export interface AnalysisQueue {
    add(file: File): string;               // returns job id, duplicate (name+size) returns the existing id
    cancel(id: string): void;              // aborts a running request, drops a queued job
    retry(id: string): void;               // re-queues a failed or paused job
    cancelAll(): void;
    jobs(): Job[];
  }
  export function createAnalysisQueue(args: {
    run: (file: File, signal: AbortSignal) => Promise<FinancialSuggestion>;
    onResult: (job: Job, suggestion: FinancialSuggestion) => void;
    onChange: () => void;
  }): AnalysisQueue;
  ```
- Produces in the hook:
  ```ts
  export function useFinancialAnalysis(args: { wid: number; flow: "INCOME" | "EXPENSE"; onResult: (job: Job, suggestion: FinancialSuggestion) => void }): {
    available: boolean;                    // status query enabled, permitted, and status.available
    jobs: Job[];
    enqueue(file: File): void;             // no-op when not available or the file is not analyzable
    reanalyze(file: File, onResult: (job: Job, s: FinancialSuggestion) => void): void;
    cancel(file: File): void;
  }
  ```

- [ ] **Step 1: Write failing queue tests** with a controllable fake `run`:
  - Jobs run one at a time in add order; the second starts only after the first settles.
  - `cancel` on the running job aborts its signal, never calls `onResult` for it, and the queue moves on; cancel on a queued job means it never runs.
  - A result that settles after `cancel` or `cancelAll` is discarded (Review Focus 1).
  - A rejection with `errorCode: "RATE_LIMITED"` marks that job `failed`, marks every remaining queued job `paused`, and stops the queue; `retry` on a paused job resumes processing.
  - Any other rejection marks the job `failed` and the queue continues with the next job.
  - Adding the same file (name and size) twice returns the same id and runs once.
- [ ] **Step 2: Write failing hook tests:** with no `canUseFinancial`, the status query is not enabled and `enqueue` never calls the API; with `status.available=false` or `reason` set, `enqueue` is a no-op; non-analyzable files (`.csv`, `.xlsx`) are never queued; unmount calls `cancelAll` and a later-settling result does not call `onResult` (Review Focus 2 and 3).
- [ ] **Step 3: Run** `cd apps/web && bunx vitest run app/modules/agent/hooks`. Expected: FAIL.
- [ ] **Step 4: Implement** the queue as a plain closure over a job array and one `AbortController` per running job, and the hook with a `useRef` queue plus a `useSyncExternalStore`-style subscription (or `useReducer` tick via `onChange`). The hook builds `run` from `analyzeFinancialDocument(wid, flow, file, signal)` and reads `available` from `financialStatusQueryOptions` gated by `useAgentPermissions(wid).canUseFinancial`.
- [ ] **Step 5: Run** the same command and `bun run typecheck`. Expected: PASS.
- [ ] **Step 6: Commit** `feat(web): ✨ add financial analysis queue and hook`.

### Task 5: Picker per-file status and manual button

**Files:**
- Modify: `apps/web/app/modules/fin/components/transaction-attachments.tsx`, `transaction-attachments.test.tsx`

**Interfaces:**
- Consumes: `Job["status"]` type from `~/modules/agent/hooks/analysis-queue`.
- Produces: `PendingAttachmentsPickerProps` gains optional
  ```ts
  analysis?: {
    statusOf: (file: File) => JobStatus | undefined;
    onAnalyze: (file: File) => void;     // the manual "Analisa dengan AI" button
  };
  onFileAdded?: (file: File) => void;    // called once per accepted file
  ```
  When `analysis` is omitted the component renders and behaves exactly as today.

- [ ] **Step 1: Write failing tests:** `onFileAdded` fires for each accepted file and not for a rejected or duplicate one; with `analysis` provided, each analyzable file row shows a status (queued, analyzing, done, failed) and an "Analisa dengan AI" button that calls `onAnalyze(file)`; a `.csv` row shows neither button nor status; the button is disabled while that file is `analyzing`; with `analysis` omitted, no AI button or status is rendered and existing tests still pass.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin/components/transaction-attachments.test.tsx`. Expected: FAIL.
- [ ] **Step 3: Implement** the optional props using the existing list rendering; analyzable detection reuses `isPreviewableAttachment({ name: file.name, mimeType: file.type })`. Add `fin.cashflow.ai.*` button and status strings to both locales.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ show analysis status in pending attachments picker`.

### Task 6: `CashflowForm` integration and feedback

**Files:**
- Modify: `apps/web/app/modules/fin/components/cashflow-form.tsx`, `cashflow-form.test.tsx`, `apps/web/app/i18n/en.ts`, `id.ts`

**Interfaces:**
- Consumes: `useFinancialAnalysis` (Task 4), `applyAnalysis`, `checkTotalMismatch` (Task 3), picker props (Task 5).
- Produces: no new exports. Behavior in `CashflowForm`: it passes `analysis` and `onFileAdded` to `PendingAttachmentsPicker` only when `available`; keeps an `aiFilled` set of field names that carry a subtle marker (`var(--omed-info)` border, no hex) until the user edits that field (`onValuesChange` clears it); renders an info `Alert` "Menganalisa {file}…" while a job runs and a success `Alert` "Agent mengisi N field dari {file}. Periksa sebelum posting." listing server `warnings`; the picker's remove handler calls the hook's `cancel(file)`; a "Analisa ulang" action wrapped in `Popconfirm` calls `reanalyze` with overwrite mode.

- [ ] **Step 1: Write failing component tests** with the API mocked (the existing test file's mocking style):
  - Agent available: adding a PNG triggers one analyze call and fills the date, payment account, reference, and items; the success alert names the file and field count.
  - Edit mode (`transactionId` set) auto-analyzes too.
  - A field the user changed before the result arrived keeps the user's value (Review Focus 2).
  - Two files added back to back are analyzed sequentially and their items merge without duplicates and without a leftover blank row (Review Focus 4).
  - Removing a file mid-analysis aborts the request and leaves the form untouched (Review Focus 1).
  - Agent unavailable, disabled, or user without permission: no AI button, no status or analyze request, and the existing form tests' snapshot of fields is unchanged (Review Focus 3).
  - "Analisa ulang": the confirm text says items will be replaced; confirming overwrites scalars and replaces items; for a `payment_proof` suggestion the items stay (Review Focus 6).
  - A total mismatch shows a warning naming the file and both amounts; a different currency shows none.
  - Failure matrix (Review Focus 5): `RATE_LIMITED` shows `message.warning` and keeps the manual buttons; `UNSUPPORTED_DOCUMENT` shows `message.warning`; 502 or 504 shows `message.error` "Analisa gagal, isi manual"; an empty result shows an info message; in every case Posting and Draft still submit successfully.
  - Unmounting during an analysis cancels it and a late result does not throw or update state.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/fin/components/cashflow-form.test.tsx`. Expected: FAIL.
- [ ] **Step 3: Implement** the integration. `onResult` reads `form.getFieldsValue(true)` and `form.isFieldTouched`, calls `applyAnalysis` in default mode, applies `values` with `form.setFieldsValue`, records `filledFields` in `aiFilled`, and runs `checkTotalMismatch` against `grandTotal`. Submission aborts pending analyses by calling `cancelAll` through the hook. Add the remaining `fin.cashflow.ai.*` and `agent.financial.*` strings to `en` and `id`.
- [ ] **Step 4: Run** `cd apps/web && bunx vitest run`, then `bun run typecheck` and `bun run lint`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ prefill income and expense forms from attachment analysis`.
