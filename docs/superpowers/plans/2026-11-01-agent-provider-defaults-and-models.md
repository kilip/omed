# Agent Provider Default URLs, Model Selection, and Page Forms Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement default base URLs per provider kind, model selection with custom text entry and error handling, and dedicated full-page routes (`/agent/providers/new` and `/agent/providers/:id/edit`) replacing the drawer on `/agent/providers`.

**Architecture:** Extend `platform/llmx` with `ListModels` querying upstream provider SDKs (OpenAI, Anthropic, Gemini, Memory). Expose model listing via `agent/service` and Fiber endpoints (`POST .../models` for probing unsaved configs, `GET .../:id/models` for saved providers). In `apps/web`, replace `ProviderFormDrawer` with a full-page `ProviderForm` component rendered by new route pages, featuring intelligent base URL defaulting, searchable `AutoComplete` model input with free-text typing, debounced credentials auto-fetch, and clear error alerts.

**Tech Stack:** Go 1.24+, Fiber v3, `openai/openai-go`, `anthropics/anthropic-sdk-go`, `google.golang.org/genai`, Swaggo, React Router v8, Ant Design, TanStack React Query, Vitest.

**Spec:** `docs/superpowers/specs/2026-11-01-agent-provider-defaults-and-models-design.md`.

## Global Constraints

- Commits follow AGENTS.md: `<type>(<scope>): <emoji> <subject>` (e.g. `feat(api): ✨ ...`, `feat(agent): ✨ ...`, `feat(web): ✨ ...`).
- Provider kinds: `openai_compat`, `anthropic`, `gemini`.
- Default base URLs:
  - `openai_compat`: `https://api.openai.com/v1`
  - `anthropic`: `https://api.anthropic.com`
  - `gemini`: `https://generativelanguage.googleapis.com`
- SSRF guard (`llmx.ValidateBaseURL`) must validate any custom or default base URL when private URLs are disallowed.
- Rate limiting: model listing requests are subject to the same `RateLimiter` per user as provider connection tests.
- Model input allows both selecting from fetched options and free-text typing.
- When loading models fails, a non-blocking error alert is displayed; the user is not prevented from entering a model name manually.
- When switching provider kinds, `baseUrl` is updated to the kind's default only if the field is untouched, empty, or equals the previous kind's default; user-customized URLs must never be overwritten.
- All copy must have entries in both `apps/web/app/i18n/en.ts` and `apps/web/app/i18n/id.ts`.
- Formatting follows `.editorconfig` (Biome for TS/JSON/CSS, gofmt for Go).

## Review Focus

1. Upstream model listing error echoes the API key: mapped error text or logs must never contain the plaintext API key (tested in Task 1).
2. Changing provider kind after user enters a custom base URL must preserve the user's custom URL (tested in Task 6).
3. Switching provider kind when the base URL matches the default of the previous kind must update the base URL to the new kind's default (tested in Task 6).
4. Probing models fails due to invalid credentials or network error: non-blocking error alert is displayed and the user can still type a model manually and submit the form (tested in Task 6).
5. Attempting to list models with SSRF-violating base URL (e.g. `http://127.0.0.1:8080` when private URLs disallowed) is rejected before network dial (tested in Task 2 & Task 3).

---

### Task 1: Add `ListModels` to `platform/llmx`

**Files:**
- Modify: `apps/api/internal/platform/llmx/types.go`
- Modify: `apps/api/internal/platform/llmx/factory.go`
- Modify: `apps/api/internal/platform/llmx/openai.go`
- Modify: `apps/api/internal/platform/llmx/anthropic.go`
- Modify: `apps/api/internal/platform/llmx/gemini.go`
- Modify: `apps/api/internal/platform/llmx/memory.go`
- Modify: `apps/api/internal/platform/llmx/factory_test.go`
- Modify: `apps/api/internal/platform/llmx/memory_test.go`

**Interfaces:**
- Produces:
  ```go
  // ModelLister defines an optional interface for adapters capable of listing models.
  type ModelLister interface {
      ListModels(ctx context.Context) ([]string, error)
  }

  // ListModels queries the provider for available model identifiers.
  func ListModels(ctx context.Context, cfg Config) ([]string, error)
  ```

- [ ] **Step 1: Write failing unit test for `llmx.ListModels` in `factory_test.go`**

```go
func TestListModels_MemoryAdapter(t *testing.T) {
    ctx := context.Background()
    models, err := ListModels(ctx, Config{
        Kind: Kind("memory"),
    })
    if err != nil {
        t.Fatalf("expected nil error, got %v", err)
    }
    if len(models) == 0 {
        t.Fatalf("expected non-empty models list")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test -v ./internal/platform/llmx -run TestListModels_MemoryAdapter`
Expected: FAIL with "undefined: ListModels"

- [ ] **Step 3: Implement `ListModels` in `llmx`**

1. In `types.go`, define `type ModelLister interface { ListModels(ctx context.Context) ([]string, error) }`.
2. In `memory.go`, implement `ListModels(ctx context.Context) ([]string, error)` returning `[]string{"memory-default", "memory-fast"}`.
3. In `openai.go`:
   Implement `ListModels(ctx context.Context) ([]string, error)`:
   - Call `a.client.Models.List(ctx, openai.ModelListParams{})`.
   - Iterate through items, collect `m.ID`, sort them, and return `wrapError(err)`.
4. In `anthropic.go`:
   Implement `ListModels(ctx context.Context) ([]string, error)`:
   - Call `a.client.Models.List(ctx, anthropic.ModelListParams{})`.
   - Iterate through items, collect `m.ID`, sort them, and return `wrapError(err)`.
5. In `gemini.go`:
   Implement `ListModels(ctx context.Context) ([]string, error)`:
   - Call `a.client.Models.List(ctx, nil)`.
   - Iterate through items, strip `models/` prefix if present, sort them, and return `wrapError(err)`.
6. In `factory.go`:
   Implement `ListModels(ctx context.Context, cfg Config) ([]string, error)`:
   - Construct adapter with `New(cfg)`.
   - If adapter implements `ModelLister`, call `ListModels(ctx)`. Otherwise return `ErrUnsupportedKind`.
   - Ensure errors never leak `cfg.APIKey` in error text.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test -v ./internal/platform/llmx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/platform/llmx/
git commit -m "feat(api): ✨ add ListModels to llmx provider drivers"
```

---

### Task 2: Service Layer for Model Listing

**Files:**
- Modify: `apps/api/internal/agent/model/provider.go`
- Modify: `apps/api/internal/agent/service/provider.go`
- Modify: `apps/api/internal/agent/service/provider_test.go`
- Modify: `apps/api/internal/agent/service/mock/mock_provider.go`

**Interfaces:**
- Consumes: `llmx.ListModels`, `llmx.Config`
- Produces:
  ```go
  // In model/provider.go:
  type ListModelsRequest struct {
      Kind    string `json:"kind" validate:"required"`
      BaseURL string `json:"baseUrl"`
      APIKey  string `json:"apiKey" validate:"required"`
  }

  // In service/provider.go ProviderService interface:
  ListModels(ctx context.Context, id int64) ([]string, error)
  ListModelsConfig(ctx context.Context, req model.ListModelsRequest) ([]string, error)
  ```

- [ ] **Step 1: Write failing unit test in `service/provider_test.go`**

```go
func TestProviderService_ListModels_Success(t *testing.T) {
    // Setup service with in-memory repo and mock adapter
    // Call svc.ListModels(ctx, providerID)
    // Assert returns list of models and no error
}

func TestProviderService_ListModelsConfig_SSRF(t *testing.T) {
    // Call svc.ListModelsConfig with private baseUrl when allowPrivate=false
    // Assert returns ErrInvalidBaseURL or SSRF error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test -v ./internal/agent/service -run TestProviderService_ListModels`
Expected: FAIL

- [ ] **Step 3: Implement `ListModels` and `ListModelsConfig` in `service/provider.go`**

1. In `model/provider.go`, define `ListModelsRequest`.
2. In `service/provider.go`:
   - Add `ListModels(ctx context.Context, id int64) ([]string, error)` and `ListModelsConfig(ctx context.Context, req model.ListModelsRequest) ([]string, error)` to `ProviderService` interface.
   - In `providerService` struct, add `listModels func(context.Context, llmx.Config) ([]string, error)` initialized to `llmx.ListModels`.
   - In `ListModelsConfig`:
     - Check rate limiter (`s.limiter.Allow(userID)`).
     - Trim `req.BaseURL`. If non-empty, call `llmx.ValidateBaseURL(req.BaseURL, s.allowPrivate)`.
     - Call `s.listModels` with `llmx.Config{Kind: req.Kind, BaseURL: req.BaseURL, APIKey: req.APIKey, AllowPrivateURLs: s.allowPrivate}`.
   - In `ListModels`:
     - Check rate limiter (`s.limiter.Allow(userID)`).
     - Load provider via `s.repo.Get(ctx, id)`.
     - Decrypt `APIKeyEnc`.
     - Call `s.listModels` with `llmx.Config{Kind: row.Kind, BaseURL: row.BaseURL, APIKey: apiKey, AllowPrivateURLs: s.allowPrivate}`.
3. Update mock generator if needed: `cd apps/api && go generate ./internal/agent/service/...`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test -v ./internal/agent/service`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/agent/model/ apps/api/internal/agent/service/
git commit -m "feat(agent): ✨ implement ListModels and ListModelsConfig in provider service"
```

---

### Task 3: Controller, Routes & Integration Tests

**Files:**
- Modify: `apps/api/internal/agent/controller/provider.go`
- Modify: `apps/api/internal/agent/routes.go`
- Modify: `apps/api/internal/agent/test/provider_test.go`

**Interfaces:**
- Consumes: `ProviderService.ListModels`, `ProviderService.ListModelsConfig`
- Produces:
  - `POST /workspaces/:wid/agent/providers/models`
  - `GET /workspaces/:wid/agent/providers/:id/models`

- [ ] **Step 1: Write integration tests in `internal/agent/test/provider_test.go`**

```go
func TestProvider_ListModels_ExistingAndConfig(t *testing.T) {
    // 1. POST /workspaces/:wid/agent/providers/models with valid payload -> 200 OK with []string
    // 2. GET /workspaces/:wid/agent/providers/:id/models -> 200 OK with []string
    // 3. POST /workspaces/:wid/agent/providers/models with blocked URL -> 422 or 400
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test -v ./internal/agent/test -run TestProvider_ListModels`
Expected: FAIL (route not found)

- [ ] **Step 3: Implement Controller methods and register routes**

1. In `controller/provider.go`:
   - Add Swaggo annotations:
     - `ListModelsConfig`: `@Summary Probe provider models` `@Router /workspaces/{wid}/agent/providers/models [post]`
     - `ListModels`: `@Summary List models for provider` `@Router /workspaces/{wid}/agent/providers/{id}/models [get]`
   - Implement `ListModelsConfig(c fiber.Ctx) error` and `ListModels(c fiber.Ctx) error`.
2. In `routes.go`:
   - Register `providers.Post("/models", az.Require(domain.ObjProvider, "create"), ctl.ListModelsConfig)` before `/:id`.
   - Register `providers.Get("/:id/models", az.Require(domain.ObjProvider, "read"), ctl.ListModels)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test -v ./internal/agent/test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/agent/controller/ apps/api/internal/agent/routes.go apps/api/internal/agent/test/
git commit -m "feat(agent): ✨ expose models listing endpoints with authz and tests"
```

---

### Task 4: Regenerate OpenAPI Schema

**Files:**
- Modify: `apps/api/docs/docs.go`
- Modify: `apps/api/docs/swagger.json`
- Modify: `apps/api/docs/swagger.yaml`
- Modify: `packages/openapi/src/api.ts`

- [ ] **Step 1: Regenerate Swagger annotations and TypeScript OpenAPI schema**

Run:
```bash
bunx turbo run oa:gen
bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts
```

- [ ] **Step 2: Verify `packages/openapi/src/api.ts` contains the new endpoints**

Run: `grep -E "agent/providers/models|agent/providers/\{id\}/models" packages/openapi/src/api.ts`
Expected: Matches found in generated paths.

- [ ] **Step 3: Run Biome check & typecheck**

Run: `bun run lint && bun run typecheck`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add apps/api/docs/ packages/openapi/src/api.ts
git commit -m "chore(agent): 🔧 regenerate OpenAPI contract for provider models endpoints"
```

---

### Task 5: Frontend Queries and Translations

**Files:**
- Modify: `apps/web/app/modules/agent/queries.ts`
- Modify: `apps/web/app/i18n/en.ts`
- Modify: `apps/web/app/i18n/id.ts`

**Interfaces:**
- Produces:
  ```ts
  export type ListModelsRequest = components["schemas"]["model.ListModelsRequest"];
  export function useProviderModels(wid?: number, providerId?: number);
  export function useProbeProviderModels(wid?: number);
  ```

- [ ] **Step 1: Add query hooks in `queries.ts`**

1. In `agentKeys`:
   - `providerModels: (wid: number, id: number) => ["agent", "workspaces", wid, "providers", id, "models"] as const`.
2. Add `useProviderModels(wid?: number, providerId?: number)`:
   - Calls `GET /workspaces/{wid}/agent/providers/{id}/models`.
   - Enabled when `Boolean(resolvedWid && providerId)`.
3. Add `useProbeProviderModels(wid?: number)`:
   - Uses `useMutation` calling `POST /workspaces/{wid}/agent/providers/models`.

- [ ] **Step 2: Add translation keys in `en.ts` and `id.ts`**

Under `agent.providers`:
- `routes.createTitle`: "Add Provider" / "Tambah Penyedia"
- `routes.editTitle`: "Edit Provider" / "Ubah Penyedia"
- `models.loading`: "Loading models..." / "Memuat model..."
- `models.loadFailed`: "Failed to load models: {{error}}" / "Gagal memuat model: {{error}}"
- `models.refresh`: "Refresh models" / "Segarkan model"
- `models.customHint`: "Select from list or type custom model name" / "Pilih dari daftar atau ketik nama model kustom"

- [ ] **Step 3: Run Biome format and lint**

Run: `cd apps/web && bunx biome check --write app/modules/agent/queries.ts app/i18n/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add apps/web/app/modules/agent/queries.ts apps/web/app/i18n/
git commit -m "feat(web): ✨ add provider models query hooks and translation keys"
```

---

### Task 6: Upgrade `ProviderForm` Component with Default URLs and Model Selection

**Files:**
- Modify: `apps/web/app/modules/agent/components/provider-form.tsx`
- Modify: `apps/web/app/modules/agent/components/provider-form.test.tsx`

**Interfaces:**
- Consumes: `useProviderModels`, `useProbeProviderModels`, default URLs dictionary
- Produces:
  ```tsx
  export type ProviderFormProps = {
      provider?: ProviderResponse | null;
      wid?: number;
      onSuccess?: () => void;
      onCancel?: () => void;
  };
  export function ProviderForm(props: ProviderFormProps): React.ReactElement;
  ```

- [ ] **Step 1: Write unit tests in `provider-form.test.tsx` for URL defaulting and custom URL preservation**

```tsx
it("sets default baseUrl when kind changes if user has not entered a custom URL", () => {
    // When kind changed to anthropic, baseUrl defaults to https://api.anthropic.com
});

it("preserves custom baseUrl when kind changes", () => {
    // User types https://custom.proxy.com, changes kind, baseUrl remains https://custom.proxy.com
});

it("allows selecting a model from options or typing a custom model", () => {
    // AutoComplete renders and accepts free text
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && bunx vitest run app/modules/agent/components/provider-form.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement `ProviderForm` improvements**

1. Define `DEFAULT_BASE_URLS_BY_KIND: Record<ProviderKind, string>`:
   - `openai_compat`: `"https://api.openai.com/v1"`
   - `anthropic`: `"https://api.anthropic.com"`
   - `gemini`: `"https://generativelanguage.googleapis.com"`
2. Add tracking for whether `baseUrl` is customized:
   - Store `isBaseUrlCustomized` or check if `currentValue` is empty or matches previous default.
   - On kind select change, update `baseUrl` if not customized.
3. Model AutoComplete:
   - Use `AutoComplete` (or `Select` with `showSearch` and custom text support) for `model`.
   - Integrate auto-probing:
     - If in edit mode and `provider.id` is available: load via `useProviderModels`.
     - If API key is entered in form: debounce (500ms) and probe via `useProbeProviderModels`.
     - Display a refresh button icon next to the model field to manually re-trigger.
     - Display an inline warning/alert if model loading fails, while keeping the input fully editable.
4. Convert `ProviderForm` from Drawer wrapper to a card/form component with Cancel and Save action buttons.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/web && bunx vitest run app/modules/agent/components/provider-form.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/modules/agent/components/provider-form.tsx apps/web/app/modules/agent/components/provider-form.test.tsx
git commit -m "feat(web): ✨ add default base URLs, model AutoComplete and auto-fetch to ProviderForm"
```

---

### Task 7: Implement Route Pages `/agent/providers/new` and `/agent/providers/:id/edit`

**Files:**
- Create: `apps/web/app/modules/agent/routes/provider-new.tsx`
- Create: `apps/web/app/modules/agent/routes/provider-new.test.tsx`
- Create: `apps/web/app/modules/agent/routes/provider-edit.tsx`
- Create: `apps/web/app/modules/agent/routes/provider-edit.test.tsx`
- Modify: `apps/web/app/modules/agent/route.ts`
- Modify: `apps/web/app/modules/agent/routes/providers.tsx`
- Modify: `apps/web/app/modules/agent/routes/providers.test.tsx`

**Interfaces:**
- Routes:
  - `/agent/providers/new` -> `provider-new.tsx`
  - `/agent/providers/:id/edit` -> `provider-edit.tsx`
- List actions:
  - "Add Provider" button navigates to `/agent/providers/new`
  - Table row "Edit" action navigates to `/agent/providers/${record.id}/edit`

- [ ] **Step 1: Write route tests for `provider-new.test.tsx` and `provider-edit.test.tsx`**

Verify that:
- `provider-new` renders the create form and redirects to `/agent/providers` on submit or cancel.
- `provider-edit` loads provider data and renders the edit form, redirecting to `/agent/providers` on submit or cancel.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && bunx vitest run app/modules/agent/routes/provider-new.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement route components and update list page**

1. Create `provider-new.tsx`:
   - Uses `Page` header with breadcrumbs and title `t("agent.providers.routes.createTitle")`.
   - Renders `ProviderForm` with `onSuccess={() => navigate("/agent/providers")}` and `onCancel={() => navigate("/agent/providers")}`.
2. Create `provider-edit.tsx`:
   - Extracts `id` from route params.
   - Fetches provider with `useProvider(id)`.
   - Renders `ProviderForm` with loaded provider.
3. In `route.ts`:
   - Add `route("providers/new", "modules/agent/routes/provider-new.tsx")`.
   - Add `route("providers/:id/edit", "modules/agent/routes/provider-edit.tsx")`.
4. In `providers.tsx`:
   - Remove `ProviderFormDrawer` state and component.
   - Update "Add Provider" button `onClick={() => navigate("new")}`.
   - Update table "Edit" action `onClick={() => navigate(`${record.id}/edit`)}`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/web && bunx vitest run app/modules/agent/routes/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/modules/agent/routes/ apps/web/app/modules/agent/route.ts
git commit -m "feat(web): ✨ add dedicated new and edit provider routes replacing drawer"
```

---

### Task 8: Full Verification & Formatting

**Files:**
- Repository-wide checks

- [ ] **Step 1: Run Go tests**

Run: `cd apps/api && go test ./...`
Expected: All tests pass.

- [ ] **Step 2: Run TypeScript typecheck**

Run: `bun run typecheck`
Expected: All projects pass without type errors.

- [ ] **Step 3: Run Biome lint & format**

Run: `bun run lint && bun run format`
Expected: Clean lint and formatting.

- [ ] **Step 4: Run Web tests**

Run: `cd apps/web && bunx vitest run app/modules/agent`
Expected: All agent module tests pass.
