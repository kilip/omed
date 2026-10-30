# Agent Module (LLM Adapter + Workspace BYOK) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a provider-agnostic LLM adapter and a workspace-scoped, encrypted BYOK provider settings feature (API + dashboard) as the new `agent` module.

**Architecture:** `platform/secretx` (AES-GCM) and `platform/llmx` (adapter interface, drivers, SSRF guard) are pure libraries. `internal/agent` composes them behind `controller -> service -> repository`, stores `agent_provider` rows per workspace, and exposes `ResolveAdapter` to other modules. The web module `agent` adds one settings screen.

**Tech Stack:** Go 1.27.1, Fiber v3, Ent + Atlas, Casbin (`authz`), Swaggo, `anthropics/anthropic-sdk-go`, `openai/openai-go`, `googleapis/go-genai`, React Router v8, Ant Design, Vitest, Cucumber + Playwright.

**Spec:** `docs/superpowers/specs/2026-10-30-agent-provider-design.md`. Read it and `AGENTS.md` first; this plan does not restate AGENTS.md conventions.

## Global Constraints

- Postgres schema `agent`; Ent entity `AgentProvider` (`ent.TypeAgentProvider`, file `ent/schema/agent_provider.go`); `field.Int64("id")`; `WorkspaceMixin` + `AuditMixin`; no `SoftDeleteMixin`; hard delete.
- Roles `agent:viewer` (`agent:provider` `read`) and `agent:admin` (inherits viewer, `agent:*` wildcard). Permission object `agent:provider`; actions `read`, `create`, `update`, `delete`. No `core:file` grants.
- Ciphertext format `v1.<base64url(nonce || ciphertext || tag)>`; AES-256-GCM; 12-byte random nonce; AAD is the decimal `workspace_id`.
- Env vars: `OMED_SECRET_KEY` (secret, base64 of exactly 32 bytes, optional at startup) and `OMED_LLM_ALLOW_PRIVATE_URLS` (plain, default `false`). Malformed `OMED_SECRET_KEY` fails startup; unset yields `503` on create/update/test.
- Provider kinds: `openai_compat`, `anthropic`, `gemini`. Sentinels: `ErrAuth`, `ErrRateLimit`, `ErrModelNotFound`, `ErrBadRequest`, `ErrUnavailable`.
- Provider HTTP calls never follow redirects; the dial guard checks the resolved IP; `http` URLs only when private URLs are allowed; userinfo in URLs is rejected.
- `apiKey` is write-only; responses carry `hasKey` and `keyHint` (last 4 chars) only; the key never appears in logs or error text.
- Routes live under `/workspaces/:wid/agent/providers`, static routes registered before `/:id`, guarded with `az.Require(domain.ObjProvider, ...)`. Foreign-workspace IDs return `404`.
- `ResolveAdapter` does not check `agent:*` roles.
- Commits: `type(scope): <emoji> subject`. Use scope `api` for Tasks 1-6, `agent` from Task 7 for module work, `web`, `e2e`, `deploy`, `docs` elsewhere.
- Go unit tests sit next to the file under test; integration tests in `internal/agent/test/`; mocks via `go:generate` mockgen like `cms/service`.
- Web: types from `@omed/openapi` via `openapi-fetch`; strings in both `i18n/en.ts` and `i18n/id.ts`; tokens from BRANDING.md, no hex; mirror `modules/cms`.

## Review Focus

1. Upstream error body echoes the API key: the mapped error text must not contain it (Task 2 test, applied in Tasks 4-6).
2. `PATCH` that changes `baseUrl` without a new `apiKey` would send the stored key to a new host: reject with `422 PROVIDER_KEY_REQUIRED` (Task 10).
3. Stored key cannot be decrypted (master key changed): `ResolveAdapter` and test return `409 PROVIDER_KEY_UNREADABLE`, not `500` (Task 10).
4. Test connection against a hanging upstream must end within 20 seconds with `ErrUnavailable` (Task 10).
5. Odd URL encodings pass string checks but hit loopback or private hosts: `http://2130706433/`, `http://[::ffff:127.0.0.1]/`, `http://localhost./`, hostnames resolving to private IPs (Task 3).

## File Structure

```
apps/api/internal/platform/secretx/{cipher.go,cipher_test.go}
apps/api/internal/platform/llmx/{types.go,errors.go,capabilities.go,factory.go,memory.go,guard.go,openai.go,anthropic.go,gemini.go} + *_test.go
apps/api/internal/agent/{module.go,routes.go}
apps/api/internal/agent/domain/{authz.go,error.go,provider.go}
apps/api/internal/agent/model/provider.go
apps/api/internal/agent/repository/provider.go (+ _test)
apps/api/internal/agent/service/{provider.go,ratelimit.go} (+ _test, mock/)
apps/api/internal/agent/controller/provider.go
apps/api/internal/agent/test/provider_test.go
apps/api/ent/schema/agent_provider.go
apps/web/app/modules/agent/{module.ts,route.ts,queries.ts,hooks/,components/,routes/}
e2e/{features/agent/,steps/agent/,pages/agent/}
```

---

### Task 1: `secretx` cipher

**Files:**
- Create: `apps/api/internal/platform/secretx/cipher.go`
- Test: `apps/api/internal/platform/secretx/cipher_test.go`

**Interfaces:**
- Produces:
  ```go
  var ErrDisabled, ErrInvalidKey, ErrDecrypt error
  type Cipher struct{ /* unexported */ }
  func New(masterKeyB64 string) (*Cipher, error) // "" => disabled Cipher, nil error
  func (c *Cipher) Enabled() bool
  func (c *Cipher) Encrypt(plaintext, aad string) (string, error)
  func (c *Cipher) Decrypt(stored, aad string) (string, error)
  ```

- [x] **Step 1: Write failing tests** in `cipher_test.go`: `TestRoundTrip` (Decrypt(Encrypt(x,"7"),"7") == x); `TestOutputHasV1Prefix` (starts with `v1.`); `TestNonceIsRandom` (two encryptions of the same input differ); `TestWrongAAD` (`errors.Is(err, ErrDecrypt)`); `TestTamperedCiphertext` (flip one byte, `ErrDecrypt`); `TestWrongKey` (`ErrDecrypt`); `TestUnknownVersionPrefix` (`v9.xxx` gives `ErrDecrypt`); `TestEmptyKeyDisabled` (`New("")` returns nil error, `Enabled()==false`, `Encrypt`/`Decrypt` return `ErrDisabled`); `TestInvalidKey` (not base64, or 16 bytes, gives `ErrInvalidKey`).
- [x] **Step 2: Run** `cd apps/api && go test ./internal/platform/secretx/...` Expected: FAIL (package missing).
- [x] **Step 3: Implement** the interface above in `cipher.go` with `crypto/aes` + `cipher.NewGCM`; AAD passed as `[]byte(aad)`.
- [x] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): ✨ add secretx cipher`

### Task 2: `llmx` types, errors, factory, memory driver

**Files:**
- Create: `apps/api/internal/platform/llmx/{types.go,errors.go,capabilities.go,factory.go,memory.go}`
- Test: `factory_test.go`, `memory_test.go`, `errors_test.go`

**Interfaces:**
- Produces (`package llmx`):
  ```go
  type Kind string // KindOpenAICompat="openai_compat", KindAnthropic="anthropic", KindGemini="gemini"
  type Role string // RoleSystem, RoleUser, RoleAssistant, RoleTool
  type Image struct{ MIME string; Data []byte }
  type Part struct{ Text string; Image *Image }
  type ToolCall struct{ ID, Name string; Arguments json.RawMessage }
  type Message struct{ Role Role; Parts []Part; ToolCalls []ToolCall; ToolCallID string }
  type Tool struct{ Name, Description string; Schema json.RawMessage }
  type Request struct{ Model string; Messages []Message; Tools []Tool; ResponseSchema json.RawMessage; MaxTokens int; Temperature *float64 }
  type Usage struct{ InputTokens, OutputTokens int }
  type Response struct{ Message Message; Usage Usage; StopReason string }
  type EventType int // EventText, EventToolCall, EventDone
  type Event struct{ Type EventType; Text string; ToolCall *ToolCall; Usage *Usage }
  type Capabilities struct{ Vision, ToolCalling, StructuredOutput, Streaming bool }
  type Adapter interface {
      Chat(ctx context.Context, req Request) (*Response, error)
      Stream(ctx context.Context, req Request) iter.Seq2[Event, error]
      Capabilities() Capabilities
  }
  type Config struct{ Kind Kind; BaseURL, Model, APIKey string; Capabilities *Capabilities; AllowPrivateURLs bool; HTTPClient *http.Client }
  type Factory func(Config) (Adapter, error)
  func New(cfg Config) (Adapter, error)               // dispatch by Kind; ErrUnsupportedKind otherwise
  func DefaultCapabilities(k Kind) Capabilities       // all four true for the three kinds
  func MergeCapabilities(base Capabilities, override *Capabilities) Capabilities
  func NewMemory(replies ...Response) *Memory          // implements Adapter; Requests() []Request records calls
  var ErrAuth, ErrRateLimit, ErrModelNotFound, ErrBadRequest, ErrUnavailable, ErrBlockedURL, ErrUnsupportedKind error
  func MapHTTPError(status int, body []byte, secrets ...string) error // wraps the sentinel; message redacts every secret
  ```
- `New` builds the guarded HTTP client (Task 3) when `cfg.HTTPClient` is nil; until Task 3 lands it uses a stub `newHTTPClient(allowPrivate bool) *http.Client` in `guard.go` that returns `http.DefaultClient`; Task 3 replaces the stub.

- [ ] **Step 1: Write failing tests:** `TestMapHTTPError` table: 401/403 → `ErrAuth`, 404 → `ErrModelNotFound`, 429 → `ErrRateLimit`, 400/422 → `ErrBadRequest`, 500/502/503 → `ErrUnavailable`; `TestMapHTTPErrorRedactsSecret` (body contains `sk-secret`, `err.Error()` does not); `TestMemoryReplaysInOrder` and `TestMemoryRecordsRequests`; `TestMemoryStreamEmitsTextThenDone`; `TestMergeCapabilities` (nil override keeps base; override replaces all four); `TestNewUnsupportedKind`.
- [ ] **Step 2: Run** `go test ./internal/platform/llmx/...` Expected: FAIL.
- [ ] **Step 3: Implement** the files above. `New` returns `ErrUnsupportedKind` for unknown kinds until Tasks 4-6 register drivers (use an internal `map[Kind]func(Config) (Adapter, error)` that each driver file fills in `init()`).
- [ ] **Step 4: Run** same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): ✨ add llmx adapter contract and memory driver`

### Task 3: `llmx` SSRF guard

**Files:**
- Create: `apps/api/internal/platform/llmx/guard.go`
- Test: `guard_test.go`

**Interfaces:**
- Consumes: `ErrBlockedURL` (Task 2), `newHTTPClient` seam (replace the stub).
- Produces:
  ```go
  func ValidateBaseURL(raw string, allowPrivate bool) error // wraps ErrBlockedURL
  func newHTTPClient(allowPrivate bool) *http.Client        // dial-time IP check, redirects disabled, 60s timeout
  func isBlockedIP(ip netip.Addr) bool                      // loopback, private, link-local, unspecified, multicast, 169.254.0.0/16, fc00::/7; unmaps ::ffff:a.b.c.d
  ```

- [ ] **Step 1: Write failing tests.** `TestValidateBaseURL` table, all with `allowPrivate=false` unless noted: `https://api.openai.com/v1` ok; `http://api.example.com` blocked (scheme); `https://user:pw@host/` blocked (userinfo); `ftp://x` blocked; `http://127.0.0.1:11434` ok only with `allowPrivate=true`; plus the Review Focus 5 inputs (`http://2130706433/`, `http://[::ffff:127.0.0.1]/`, `http://localhost./`) blocked when `allowPrivate=false`. `TestGuardBlocksDialToLoopback` (httptest server on 127.0.0.1, client from `newHTTPClient(false)` errors with `ErrBlockedURL`; with `true` succeeds). `TestGuardBlocksHostnameResolvingToPrivate` (use a custom resolver seam, hostname maps to 10.0.0.5, expect blocked). `TestRedirectsAreNotFollowed` (server returns 302; client returns the 302 response without following; status 302 maps to `ErrUnavailable` via `MapHTTPError`).
- [ ] **Step 2: Run** `go test ./internal/platform/llmx/ -run 'Guard|Validate|Redirect'` Expected: FAIL.
- [ ] **Step 3: Implement** `guard.go`. Use `net.Dialer{Control: ...}` checking the connecting IP so DNS rebinding cannot bypass; expose a package-private resolver variable for the test; `CheckRedirect` returns `http.ErrUseLastResponse`.
- [ ] **Step 4: Run** `go test ./internal/platform/llmx/...` Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): ✨ add llmx url guard`

### Tasks 4-6: drivers (`openai`, `anthropic`, `gemini`)

Each driver is its own task with the same shape. Order: Task 4 `openai.go` (`KindOpenAICompat`, SDK `github.com/openai/openai-go`, base URL from `Config.BaseURL`), Task 5 `anthropic.go` (`KindAnthropic`, SDK `github.com/anthropics/anthropic-sdk-go`), Task 6 `gemini.go` (`KindGemini`, SDK `github.com/googleapis/go-genai`).

**Files (per driver):** Create `apps/api/internal/platform/llmx/<driver>.go`; Test `<driver>_test.go`; modify `apps/api/go.mod`/`go.sum`.

**Interfaces:**
- Consumes: `Adapter`, `Config`, `MapHTTPError`, `DefaultCapabilities`, `MergeCapabilities`, `newHTTPClient` (Tasks 2-3).
- Produces: `func newOpenAI(cfg Config) (Adapter, error)` (and `newAnthropic`, `newGemini`) registered for its kind in `init()`.

- [ ] **Step 1: Verify the SDK.** Open the SDK's repository page and `pkg.go.dev`, confirm the import path, latest stable tag at least 14 days old, and the option that injects a custom `*http.Client` (for example `option.WithHTTPClient` / `genai.ClientConfig.HTTPClient`). Record the chosen version in the commit body. If a path differs from the one above, stop and note it under **Blockers** at the end of this file.
- [ ] **Step 2: Write failing tests** against an `httptest.Server` that mimics the provider, constructing the adapter with `AllowPrivateURLs: true` and the server URL: `TestChatMapsRequestAndResponse` (assert the request body carries model, system and user text, one image part when `Vision`, the tool definitions, and the response schema when set; assert the decoded `Response` text, tool calls, usage, stop reason); `TestStreamYieldsTextDeltasThenDone` (SSE fixture, events in order, final `EventDone` carries `Usage`); `TestStreamToolCallDeltasAssembleOneToolCall`; `TestErrorMapping` (401, 404, 429, 500 map to the sentinels); `TestErrorBodyNeverContainsAPIKey` (server echoes the key in its error body; `err.Error()` must not contain it); `TestNoRedirectFollow`; `TestCapabilitiesOverride` (`Config.Capabilities` replaces defaults).
- [ ] **Step 3: Run** `go test ./internal/platform/llmx/ -run <Driver>` Expected: FAIL.
- [ ] **Step 4: Implement** the constructor and the three `Adapter` methods. Normalize provider events into `Event`; wrap every SDK error through `MapHTTPError` (extract status and body from the SDK error type) passing `cfg.APIKey` as a secret.
- [ ] **Step 5: Run** `cd apps/api && go mod tidy && go test ./internal/platform/llmx/...` Expected: PASS.
- [ ] **Step 6: Commit** `feat(api): ✨ add llmx <driver> driver`

### Task 7: Config, commitlint scope, Helm chart

**Files:**
- Modify: `apps/api/internal/config/config.go`, `config_test.go`, `apps/api/.env.example`, `README.md` (env table), `commitlint.config.ts`, `deploy/helm/omed/{values.yaml,values.schema.json,templates/_helpers.tpl,templates/secret.yaml,README.md}`, `AGENTS.md` (scope list line only).

**Interfaces:**
- Produces: `Config.SecretKey string` (`env:"OMED_SECRET_KEY"`), `Config.LLMAllowPrivateURLs bool` (`env:"OMED_LLM_ALLOW_PRIVATE_URLS,default=false"`).

- [ ] **Step 1: Write failing tests** in `config_test.go` mirroring the CMS media config tests: `TestAgentConfigDefaults` (`LLMAllowPrivateURLs==false`, `SecretKey==""`), `TestAgentConfigEnvOverride`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/config/...` Expected: FAIL.
- [ ] **Step 3: Implement** the two fields following `CMSMediaMaxSizeMB` style; add both vars to `.env.example` and the README environment table; add `agent` to the commitlint scope enum and to the scope list in AGENTS.md section 2.
- [ ] **Step 4: Helm** per AGENTS.md section 6: `OMED_LLM_ALLOW_PRIVATE_URLS` in `values.yaml`, `values.schema.json`, `omed.api.env`; `OMED_SECRET_KEY` in `templates/secret.yaml` and the chart README Secrets table (note that losing the key makes stored provider keys unrecoverable). Run `helm lint deploy/helm/omed --strict -f deploy/helm/omed/ci/full-values.yaml` and `helm template omed deploy/helm/omed -f deploy/helm/omed/ci/full-values.yaml | kubeconform -strict -summary`. Expected: no errors.
- [ ] **Step 5: Run** `go test ./internal/config/...` and `bun run lint`. Expected: PASS.
- [ ] **Step 6: Commit** `feat(api): ✨ add agent config and chart values` (scope `api`; the `agent` scope is valid from the next commit). Commit the Helm changes separately as `feat(deploy): ✨ add agent env to helm chart`.

### Task 8: Ent schema, migration, tenant filter

**Files:**
- Create: `apps/api/ent/schema/agent_provider.go`, migration in `apps/api/migrations/`.
- Modify: `apps/api/internal/core/mutator.go` (both `isWorkspaceScopedType` and the `filter` switch), `mutator_test.go`; generated `apps/api/ent/*`.

**Interfaces:**
- Produces: Ent entity `ent.AgentProvider` with fields `workspace_id`, `name`, `kind` (enum `openai_compat|anthropic|gemini`), `base_url` (optional), `model`, `api_key_enc`, `key_hint`, `is_default` (default false), `capabilities` (`field.JSON`, type `llmx.Capabilities`, optional); indexes unique `(workspace_id, name)`, partial unique `(workspace_id)` with `entsql.IndexWhere("is_default")`.

- [ ] **Step 1: Write failing test** in `mutator_test.go` following the existing cms cases: `TestAgentProviderIsWorkspaceFiltered` (rows in two workspaces, query under workspace A returns only A's) and `TestAgentProviderCreateGetsWorkspaceID`.
- [ ] **Step 2: Implement the schema** per spec section 7 (`entsql.Schema("agent")`, `Mixin` = `WorkspaceMixin{}, AuditMixin{}`). Run `cd apps/api && go generate ./ent`.
- [ ] **Step 3: Register** `ent.TypeAgentProvider` in `isWorkspaceScopedType` and add the `case` with `agentprovider.FieldWorkspaceID` in `workspaceInterceptor.filter`.
- [ ] **Step 4: Migration:** `cd apps/api && go run ./cmd/migrate add_agent_provider`; confirm the new SQL creates schema `agent` and both indexes, including the `WHERE is_default` partial one.
- [ ] **Step 5: Run** `cd apps/api && go build ./... && go test ./internal/core/...` Expected: PASS.
- [ ] **Step 6: Commit** `feat(agent): ✨ add agent provider schema` (include generated `ent/*` and `migrations/*`).

### Task 9: Domain and repository

**Files:**
- Create: `internal/agent/domain/{authz.go,error.go,provider.go}`, `internal/agent/repository/provider.go`
- Test: `internal/agent/domain/authz_test.go`, `internal/agent/repository/provider_test.go`

**Interfaces:**
- Produces:
  ```go
  // domain
  const ObjProvider = "agent:provider"; const RoleViewer = "agent:viewer"; const RoleAdmin = "agent:admin"
  var ErrProviderNotFound, ErrProviderNameTaken, ErrNoDefaultProvider, ErrInvalidBaseURL,
      ErrSecretKeyMissing, ErrProviderKeyRequired, ErrProviderKeyUnreadable, ErrTestRateLimited error
  // registered with base.AddToErrorMap: 404, 409, 422, 422, 503, 422, 409, 429 respectively
  type Provider struct{ ID, WorkspaceID int64; Name string; Kind llmx.Kind; BaseURL, Model, KeyHint string
      HasKey, IsDefault bool; Capabilities *llmx.Capabilities; CreatedAt, UpdatedAt time.Time }
  // repository
  type ProviderRepository interface {
      Create(ctx, p NewProvider) (*ent.AgentProvider, error)       // NewProvider: Name, Kind, BaseURL, Model, APIKeyEnc, KeyHint, Capabilities, IsDefault
      List(ctx) ([]*ent.AgentProvider, error)                      // ordered is_default DESC, name ASC
      Get(ctx, id int64) (*ent.AgentProvider, error)               // ErrProviderNotFound
      GetDefault(ctx) (*ent.AgentProvider, error)                  // ErrNoDefaultProvider
      Update(ctx, id int64, u ProviderUpdate) (*ent.AgentProvider, error)   // ProviderUpdate: Name, BaseURL, Model, APIKeyEnc, KeyHint *string; Capabilities *llmx.Capabilities (nil = unchanged)
      Delete(ctx, id int64) error
      SetDefault(ctx, id int64) (*ent.AgentProvider, error)        // single tx: clear old, set new
  }
  func NewProviderRepository(cli *ent.Client) ProviderRepository
  ```
  Repositories add an explicit `workspace_id` predicate from `base.WorkspaceIDFromContext(ctx)` in addition to the interceptor. Unique violations map to `ErrProviderNameTaken`.

- [ ] **Step 1: Write failing tests.** Authz: `TestAgentRolesRegistered` (both roles exist; admin inherits viewer; viewer has only `read`; no `core:file` permission) using the registry lookup API in `platform/authz/registry.go`. Repository (use `testutil.GetEntCli()`): `TestCreateAndGet`, `TestNameUniquePerWorkspace` (same name allowed in another workspace, `ErrProviderNameTaken` in the same), `TestGetForeignWorkspaceIsNotFound`, `TestSetDefaultMovesFlag` (exactly one default after switching), `TestConcurrentSetDefaultLeavesOneDefault` (two goroutines, final count of defaults is 1, no error other than a retriable one), `TestGetDefaultNone` (`ErrNoDefaultProvider`), `TestDeleteIsHard` (row gone, not soft-deleted).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/agent/...` Expected: FAIL.
- [ ] **Step 3: Implement** domain and repository per the interfaces; roles via `authz.AddRole` in `init()` mirroring `fin/domain/authz.go`.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(agent): ✨ add agent domain and repository`

### Task 10: Service

**Files:**
- Create: `internal/agent/service/{provider.go,ratelimit.go}`, `internal/agent/model/provider.go`
- Test: `provider_test.go`, `ratelimit_test.go`; generate `service/mock/mock_provider.go`.

**Interfaces:**
- Consumes: `ProviderRepository`, `secretx.Cipher`, `llmx.Factory`, `llmx.ValidateBaseURL`.
- Produces:
  ```go
  // model
  type CreateProviderRequest struct{ Name, Kind, BaseURL, Model, APIKey string; Capabilities *llmx.Capabilities; IsDefault bool }
  type UpdateProviderRequest struct{ Name, BaseURL, Model, APIKey *string; Capabilities *llmx.Capabilities }  // kind is immutable
  type TestProviderRequest struct{ Kind, BaseURL, Model, APIKey string; Capabilities *llmx.Capabilities }
  type TestResult struct{ OK bool; Code string; LatencyMs int64 }   // Code: "", "AUTH_FAILED", "MODEL_NOT_FOUND", "RATE_LIMITED", "BAD_REQUEST", "UNAVAILABLE", "BLOCKED_URL"
  // service
  type ProviderService interface {
      List(ctx) ([]domain.Provider, error)
      Get(ctx, id int64) (domain.Provider, error)
      Create(ctx, req model.CreateProviderRequest) (domain.Provider, error)
      Update(ctx, id int64, req model.UpdateProviderRequest) (domain.Provider, error)
      Delete(ctx, id int64) error
      SetDefault(ctx, id int64) (domain.Provider, error)
      Test(ctx, id int64) (model.TestResult, error)
      TestConfig(ctx, req model.TestProviderRequest) (model.TestResult, error)
      ResolveAdapter(ctx, providerID *int64) (llmx.Adapter, error)   // nil => default
  }
  func NewProviderService(repo repository.ProviderRepository, cipher *secretx.Cipher, allowPrivate bool, opts ...ProviderOption) ProviderService
  func WithAdapterFactory(f llmx.Factory) ProviderOption
  func WithClock(now func() time.Time) ProviderOption
  func WithTestTimeout(d time.Duration) ProviderOption // default 20s
  ```

- [ ] **Step 1: Write failing tests** (repository mocked, `llmx.NewMemory` via `WithAdapterFactory`): `TestCreateEncryptsKeyAndStoresHint` (stored `APIKeyEnc` starts `v1.`, `KeyHint` is last 4 chars, returned Provider has `HasKey` true and no key); `TestCreateValidatesBaseURL` (`ErrInvalidBaseURL`); `TestCreateRejectsBlankOrOverlongName` (trimmed empty, over 100 chars → validation error); `TestFirstProviderBecomesDefault`; `TestCreateWithoutSecretKey` (disabled cipher → `ErrSecretKeyMissing`; `List` and `Delete` still work); `TestUpdateKeepsKeyWhenAPIKeyNil`; `TestUpdateBaseURLWithoutKeyRejected` (`ErrProviderKeyRequired`, Review Focus 2); `TestResolveAdapterUsesDefault`; `TestResolveAdapterNoDefault` (`ErrNoDefaultProvider`); `TestResolveAdapterUnreadableKey` (decrypt with a different cipher → `ErrProviderKeyUnreadable`, Review Focus 3); `TestResolveAdapterMergesCapabilities`; `TestTestMapsSentinelsToCodes` (memory/stub adapter returning each sentinel → `Code`); `TestTestHangingUpstreamTimesOut` (stub adapter blocks until `ctx.Done()`; with `WithTestTimeout(50*time.Millisecond)` the result is `OK=false`, `Code="UNAVAILABLE"`, returned within 1s; a separate assertion checks the default timeout constant equals 20s, Review Focus 4); `TestRateLimitPerUser` (11th test within a minute returns `ErrTestRateLimited`; another user unaffected; refills after the injected clock advances 1 minute).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/agent/service/...` Expected: FAIL.
- [ ] **Step 3: Implement** the service. The key is encrypted with AAD `strconv.FormatInt(workspaceID, 10)`. Test connection sends one user message `"ping"` with `MaxTokens: 1` under the test timeout. Rate limit: in-memory token bucket, 10 tests per minute per user, keyed by the user ID in `base.UserFromContext(ctx)`. Never log or wrap the key into errors.
- [ ] **Step 4: Generate mocks** with the same `go:generate` directive style as `cms/service/post.go`; run `go generate ./internal/agent/...`.
- [ ] **Step 5: Run** the same test command. Expected: PASS.
- [ ] **Step 6: Commit** `feat(agent): ✨ add agent provider service`

### Task 11: Controller, routes, module wiring, integration tests

**Files:**
- Create: `internal/agent/controller/provider.go`, `internal/agent/{module.go,routes.go}`, `internal/agent/test/provider_test.go`
- Modify: `internal/config/app.go` (construct `agent.New` after `cmsMod`, `SetAuthorizer`, append to `modules`), `testutil/init.go` (test env: a fixed valid `OMED_SECRET_KEY` and `OMED_LLM_ALLOW_PRIVATE_URLS=true`).

**Interfaces:**
- Consumes: `ProviderService` (Task 10).
- Produces: `agent.Config{FiberApp *fiber.App, Logger *slog.Logger, EntCli *ent.Client, Cipher *secretx.Cipher, AllowPrivateURLs bool}`, `agent.New(Config) *AgentModule`, field `AgentModule.ProviderService service.ProviderService` (other modules call `ResolveAdapter` through it), `SetAuthorizer`, `PreBoot`, `Boot`. Response DTO `model.ProviderResponse` (`id, name, kind, baseUrl, model, hasKey, keyHint, isDefault, capabilities, auditMeta`). In `config/app.go`, build the cipher with `secretx.New(state.Config.SecretKey)` and `log.Fatalf` on error, like the mailer init there.

- [ ] **Step 1: Write failing integration tests** (`testutil.ApiTestSuite[model.ProviderResponse]`, an `httptest` OpenAI-compatible fake as the provider base URL): `TestCRUD`; `TestSetDefault`; `TestPermissionMatrix` (viewer: `GET` ok, `POST/PATCH/DELETE/test` `403`; admin all ok; member with no role `403`; unauthenticated `401`); `TestCrossWorkspaceIsolation` (provider of workspace A by ID under workspace B → `404`); `TestKeyNeverInAnyResponse` (create, get, list, update bodies do not contain the key; `hasKey` true, `keyHint` is the last 4); `TestTestEndpointOK` and `TestTestEndpointAuthFailure` (fake returns 401, body `ok=false`, `code=AUTH_FAILED`, HTTP 200); `TestTestConfigBeforeSave`; `TestDeleteDefaultLeavesNoDefault`; `TestUpdateBaseURLWithoutKey422`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/agent/test/...` Expected: FAIL.
- [ ] **Step 3: Implement** controller with Swaggo annotations (`@Tags Agent Providers`, `@Security BearerAuth`, error responses incl. 503) and routes exactly as in Global Constraints: `GET /`, `POST /`, `POST /test`, `GET /:id`, `PATCH /:id`, `DELETE /:id`, `POST /:id/default`, `POST /:id/test`. Required actions: reads `read`; create and `POST /test` `create`; `PATCH`, default, `/:id/test` `update`; delete `delete`. Mirror `cms/controller/page.go` and `cms/routes.go` for response helpers and registration.
- [ ] **Step 4: Run** `cd apps/api && go test ./...` Expected: PASS.
- [ ] **Step 5: Commit** `feat(agent): ✨ add agent provider endpoints`

### Task 12: OpenAPI regeneration

- [ ] **Step 1: Run** `bunx turbo run oa:gen` then `bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts`.
- [ ] **Step 2: Verify** the spec contains all eight operations under `/workspaces/{wid}/agent/providers`; run `bun run typecheck`. Expected: PASS.
- [ ] **Step 3: Commit** `chore(api): 🔧 regenerate openapi for agent` (include `apps/api/docs/*`, `packages/openapi/src/api.ts`).

### Task 13: Web module scaffolding and permissions

**Files:**
- Create: `apps/web/app/modules/agent/{module.ts,route.ts,queries.ts,hooks/index.ts,hooks/use-agent-permissions.ts,hooks/use-agent-permissions.test.ts}`
- Modify: `apps/web/app/lib/roles.ts`, `apps/web/app/i18n/{en.ts,id.ts}`

**Interfaces:**
- Produces: `useAgentPermissions(): { canRead: boolean; canManage: boolean }` (`canManage` = `agent:admin`); `queries.ts` exports `agentKeys`, `useProviders()`, `useCreateProvider()`, `useUpdateProvider()`, `useDeleteProvider()`, `useSetDefaultProvider()`, `useTestProvider()`, `useTestProviderConfig()`.

- [ ] **Step 1: Write failing tests** `use-agent-permissions.test.ts` mirroring `use-cms-permissions.test.ts`: viewer → `canRead` true, `canManage` false; admin → both true; no role → both false.
- [ ] **Step 2: Run** `bun run test` for the web package. Expected: FAIL.
- [ ] **Step 3: Implement** `module.ts` via `defineModule` (`id: "agent"`, i18n label/description keys, Ant Design icon, `order: 50`, one menu entry `providers`), `route.ts` (`prefix("agent", [...])`, index redirects to `providers`, route file placeholder), `queries.ts` mirroring `modules/cms/queries.ts`, role labels `role.agent.viewer` and `role.agent.admin`, i18n keys `agent.label`, `agent.description`, `agent.menu.providers`.
- [ ] **Step 4: Run** `bun run typecheck && bun run lint && bun run test`. Expected: PASS; the launcher shows the module.
- [ ] **Step 5: Commit** `feat(web): ✨ add agent module scaffolding`

### Task 14: Providers screen

**Files:**
- Create: `apps/web/app/modules/agent/routes/providers.tsx`, `components/{provider-form.tsx,provider-test-button.tsx}` plus `*.test.tsx`
- Modify: `route.ts`, i18n files.

**Interfaces:**
- Consumes: `useAgentPermissions`, query hooks (Task 13).

- [ ] **Step 1: Write failing component tests** (mirror `transaction-attachments.test.tsx`): `renders provider table with default marker and key hint`; `viewer sees no add, edit, delete, test, or set-default controls`; `admin create submits apiKey and clears it from form state after success`; `edit leaves apiKey empty and omits it from the PATCH body`; `changing base URL without a new API key shows the PROVIDER_KEY_REQUIRED error inline`; `test connection shows success and shows the mapped failure code`; `503 shows the secret-key-missing message`.
- [ ] **Step 2: Run** `bun run test`. Expected: FAIL.
- [ ] **Step 3: Implement** the table, the create/edit drawer (kind select, base URL, model, password-type API key, four capability switches), row actions (Test, Set default, Delete with confirm), gated by `canManage`. All strings in both i18n files; tokens only, no hex; types from `@omed/openapi`.
- [ ] **Step 4: Run** `bun run typecheck && bun run lint && bun run test`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): ✨ add agent providers screen`

### Task 15: E2E

**Files:**
- Create: `e2e/pages/agent/providers.page.ts`, `e2e/features/agent/providers.feature`, `e2e/features/agent/permissions.feature`, `e2e/steps/agent/*.steps.ts`
- Modify: `e2e/pages/index.ts` (register `agent` group in `createPages()`), `e2e/support/servers.ts` (set `OMED_SECRET_KEY` to a fixed valid test key and `OMED_LLM_ALLOW_PRIVATE_URLS=true` for the started API), plus a small fake OpenAI-compatible server helper in `e2e/support/`.

- [ ] **Step 1: Write the features** (tag `@authenticated @agent`): admin adds a provider, sets it default, tests the connection against the fake server, deletes it; viewer sees the list without management controls (role assigned with `addMember({ workspaceId, userId, role: "agent:viewer" })`, see `features/fin/permissions.feature`).
- [ ] **Step 2: Implement** page objects (locators and `expect*` only here) and steps (call page objects through `this.pages.agent`).
- [ ] **Step 3: Run** `bunx playwright install chromium` once, then `bun run e2e`. Expected: PASS.
- [ ] **Step 4: Commit** `test(e2e): ✅ cover agent providers`

### Task 16: Docs

- [ ] **Step 1:** README: add an Agent section in the style of the Finance and CMS route lists (web route `/agent/providers`, API summary, roles, the two env vars if not already listed); mention schema `agent` in AGENTS.md section 3. Do not change BRANDING.md.
- [ ] **Step 2: Verify** the README routes and env vars match the code; run `bun run lint`.
- [ ] **Step 3: Commit** `docs(docs): 📝 document agent module`

---

## Definition of done

- All tasks checked; `cd apps/api && go test ./...`, `bun run lint`, `bun run typecheck`, `bun run test`, `bun run e2e`, `helm lint` and `kubeconform` pass.
- `apps/api/docs/*` and `packages/openapi/src/api.ts` match the controllers.
- No controller touches Ent or repositories; no module reaches another module's repository.
- `ent.TypeAgentProvider` is covered by the workspace filter and its test.
- No API response, log line, or error text contains a provider API key.

## Blockers

_Record anything that blocked you here (task ID, what you found, what you need)._
