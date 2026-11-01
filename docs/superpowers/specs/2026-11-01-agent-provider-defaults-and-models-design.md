# Design: Agent Provider Default URLs, Model Selection, and Page Forms

Status: Draft. Date: 2026-11-01. Target branch: `main` of `kilip/omed`. Issue: #39.

## 1. Purpose

On `/agent/providers`, users currently have to enter provider base URLs manually with no defaults, and must know exact model names in advance because available models are not listed. Furthermore, managing providers via a side drawer is cramped and inconsistent with other multi-field resource creation in Omed.

This design addresses GitHub issue #39 by:
1. Setting sensible default base URLs for each provider kind (`openai_compat`, `anthropic`, `gemini`) while preserving any user-customized URL.
2. Allowing users to select from available models retrieved directly from the provider API, while retaining the ability to type custom model names.
3. Clearly displaying loading states and errors when fetching models fails.
4. Transitioning provider creation and editing from a Drawer to dedicated page routes: `/agent/providers/new` and `/agent/providers/:id/edit`.

## 2. Scope

### In Scope:
- **`platform/llmx`**:
  - Extend provider adapters / package with `ListModels(ctx context.Context, cfg Config) ([]string, error)` supporting `openai_compat`, `anthropic`, and `gemini`, plus in-memory mock for unit tests.
- **`agent` module (API)**:
  - Add service methods to list models:
    - Unsaved config probe: `ListModelsConfig(ctx, req model.ListModelsRequest) ([]string, error)`
    - Existing provider: `ListModels(ctx, id int64) ([]string, error)` using decrypted workspace credentials.
  - Rate limiting for model listing requests using the existing `RateLimiter`.
  - HTTP Endpoints:
    - `POST /workspaces/{wid}/agent/providers/models`: probe available models for a given configuration.
    - `GET /workspaces/{wid}/agent/providers/{id}/models`: retrieve available models for an existing provider.
  - Controller, routes, Swagger annotations, and OpenAPI schema generation.
- **`agent` module (Web)**:
  - Add routes: `/agent/providers/new` (`provider-new.tsx`) and `/agent/providers/:id/edit` (`provider-edit.tsx`).
  - Extract reusable `ProviderForm` page component replacing `ProviderFormDrawer`.
  - Intelligent default base URLs:
    - `openai_compat`: `https://api.openai.com/v1`
    - `anthropic`: `https://api.anthropic.com`
    - `gemini`: `https://generativelanguage.googleapis.com`
    - Preserves user edits when switching provider kinds unless untouched/default.
  - Searchable model selection with custom text entry (`AutoComplete` or `Select` with custom search & tag/text entry).
  - Automatic model fetching once valid credentials/URL are present, with debouncing and a manual refresh button.
  - Clear error notifications when model fetching fails.
  - Update `/agent/providers` list page to navigate to the new routes.
  - Update unit and integration tests.

### Out of Scope:
- Database schema changes (the existing `agent_providers` table already stores `kind`, `baseUrl`, `model`, `apiKeyEnc`, etc.).
- Filtering models by specific capability (e.g. vision or tool calling) on the provider API side; the full list is returned and sorted alphabetically.

## 3. Approaches Considered

### Approach 1: Dedicated REST Endpoints with `llmx.ListModels` (Selected)
- **Design**:
  - `POST /workspaces/{wid}/agent/providers/models` for probing unsaved credentials.
  - `GET /workspaces/{wid}/agent/providers/{id}/models` for fetching models using saved credentials.
  - `llmx.ListModels(ctx, cfg)` invoked by the service layer.
  - Dedicated React Router route pages (`provider-new.tsx` and `provider-edit.tsx`).
- **Tradeoffs**: Requires two endpoints in Go, but follows strict REST conventions and existing patterns in Omed (such as `/test` and `/:id/test`). Clean separation of concerns and clear typing in OpenAPI.

### Approach 2: Unified Probe Endpoint (`POST /providers/models`)
- **Design**: A single POST endpoint taking optional `id` or raw credentials.
- **Tradeoffs**: Fewer endpoints, but less idiomatic REST semantics (using POST to query an existing unchanged provider) and complicates OpenAPI contract. Rejected in favor of Approach 1.

## 4. Technical Specifications

### 4.1 Backend (`apps/api`)

#### Platform `llmx`
Add model listing support:
```go
// ListModels queries the provider for available model identifiers.
func ListModels(ctx context.Context, cfg Config) ([]string, error)
```
- **OpenAI**: calls `client.Models.List(ctx, openai.ModelListParams{})`, extracts `model.ID`.
- **Anthropic**: calls `client.Models.List(ctx, anthropic.ModelListParams{})`, extracts `model.ID`.
- **Gemini**: calls `genClient.Models.List(ctx, nil)`, extracts model names (stripping `models/` prefix if present).
- **In-Memory**: returns a mock list of models (e.g., `[]string{"mock-gpt-4", "mock-gpt-3.5"}`).

#### Service Layer (`agent/service/provider.go`)
- `ListModelsConfig(ctx context.Context, req model.ListModelsRequest) ([]string, error)`:
  - Enforces user rate limit (`s.limiter.Allow(userID)`).
  - Validates `req.BaseURL` if provided using `llmx.ValidateBaseURL`.
  - Calls `s.listModelsFactory` / `llmx.ListModels`.
- `ListModels(ctx context.Context, id int64) ([]string, error)`:
  - Enforces user rate limit.
  - Loads provider by ID from `s.repo`.
  - Decrypts `APIKeyEnc`.
  - Calls `s.listModelsFactory` / `llmx.ListModels`.

#### Controller & Routing (`agent/controller/provider.go`, `agent/routes.go`)
- `POST /workspaces/:wid/agent/providers/models` (authz: `agent:provider`, `create`)
- `GET /workspaces/:wid/agent/providers/:id/models` (authz: `agent:provider`, `read`)

### 4.2 Frontend (`apps/web`)

#### Routes & Navigation
- `/agent/providers`:
  - "Add Provider" button navigates to `/agent/providers/new`.
  - Table row "Edit" action navigates to `/agent/providers/${id}/edit`.
  - Remove `ProviderFormDrawer`.
- `/agent/providers/new`:
  - Renders `ProviderForm` in create mode.
  - On submit: calls `useCreateProvider`, navigates to `/agent/providers`.
  - On cancel: navigates to `/agent/providers`.
- `/agent/providers/:id/edit`:
  - Fetches provider with `useProvider(id)`.
  - Renders `ProviderForm` in edit mode.
  - On submit: calls `useUpdateProvider`, navigates to `/agent/providers`.
  - On cancel: navigates to `/agent/providers`.

#### Default Base URL Handling
- Predefined defaults:
  - `openai_compat`: `https://api.openai.com/v1`
  - `anthropic`: `https://api.anthropic.com`
  - `gemini`: `https://generativelanguage.googleapis.com`
- When `kind` changes:
  - If `baseUrl` has not been modified by the user (or matches the previous kind's default or is empty), automatically update it to the new kind's default.
  - If user entered a custom URL, preserve their value.

#### Model Selection & Auto-fetch
- Component: `AutoComplete` with `options={modelOptions}` and `filterOption` enabled, allowing free text entry so custom model names can be typed freely.
- Query Hooks in `apps/web/app/modules/agent/queries.ts`:
  - `useProviderModels(wid, id)`: queries `GET /workspaces/{wid}/agent/providers/{id}/models`.
  - `useProbeProviderModels(wid)`: mutation or query for `POST /workspaces/{wid}/agent/providers/models`.
- Behavior:
  - When editing: automatically fetches models using saved provider credentials.
  - When creating (or when API key is entered): automatically triggers model probing once API key has a value (debounced by ~500ms).
  - Refresh button allows manual re-fetching.
  - Loading indicator during fetch.
  - If fetch fails: display a non-blocking error Alert / message showing why (e.g. "Unable to load models: invalid API key"), without preventing the user from typing a model name manually.

## 5. Verification Plan

1. **Backend Tests**:
   - Unit tests for `llmx.ListModels` across all providers and mock.
   - Unit tests for `ProviderService.ListModels` and `ProviderService.ListModelsConfig`.
   - Controller & integration tests for `/workspaces/:wid/agent/providers/models` and `/:id/models`.
2. **Frontend Tests**:
   - `provider-form.test.tsx`: test default URL updates on kind change, preserving custom URL, model selection, custom typing, and error display.
   - Route tests for `/agent/providers`, `/agent/providers/new`, and `/agent/providers/:id/edit`.
3. **End-to-End / Manual Verification**:
   - Navigate to `/agent/providers/new`.
   - Verify default base URL is `https://api.openai.com/v1`.
   - Switch kind to Anthropic -> URL updates to `https://api.anthropic.com`.
   - Change URL to `https://my-proxy.com`. Switch to Gemini -> URL remains `https://my-proxy.com`.
   - Enter API key and verify models are fetched and appear in dropdown.
   - Type a custom model name not in the list -> verify form accepts and saves it.
