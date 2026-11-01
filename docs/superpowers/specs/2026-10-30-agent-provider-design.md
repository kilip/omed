# Design: `agent` module, LLM adapter and workspace BYOK providers

Status: Completed. Date: 2026-10-30. Target branch: `main` of `kilip/omed`.

## 1. Purpose

Omed needs a way to call LLMs using a **provider and API key supplied by the workspace** (BYOK). This iteration delivers the foundation only: a provider-agnostic adapter, encrypted per-workspace provider configuration, and a settings screen. Features that consume it (receipt extraction, chat assistant, MCP) come later and are not part of this design.

Success criteria:

- A workspace admin can add, edit, delete, test, and set a default LLM provider config from the dashboard.
- Backend code can obtain a working `llmx.Adapter` for a workspace with one service call.
- API keys are encrypted at rest and never returned by the API or written to logs.
- Adding a fourth provider kind requires a new driver only, not changes to callers.

## 2. Scope

In scope:

- `platform/llmx`: adapter interface, drivers (OpenAI-compatible, Anthropic, Gemini, in-memory for tests), capability flags, base URL validation (SSRF guard).
- `platform/secretx`: authenticated encryption for stored secrets.
- `agent` module (API): entity `agent_provider`, repository, service, controller, routes, roles `agent:viewer` and `agent:admin`.
- `agent` module (web): provider settings page.
- Config, Helm chart, commitlint scope, README and AGENTS.md updates.

Out of scope (do not build):

- Tool registry, tool-calling loop, MCP server, agent runs, audit log of tool calls.
- Per-feature provider binding (a feature picks a provider by ID or uses the default).
- Per-user personal providers, usage metering, cost tracking.
- Key rotation tooling (the ciphertext format only leaves room for it).
- Redis- or Kafka-backed anything.

## 3. Decisions

| # | Decision |
| --- | --- |
| D1 | New module `agent` (Go `internal/agent`, Postgres schema `agent`, web `modules/agent`, commit scope `agent`). |
| D2 | Own thin adapter wrapping official SDKs. No agent framework (langchaingo, Genkit, Eino). |
| D3 | Providers are **per workspace**. A workspace may have many, exactly one default. A user needing a private key uses their personal workspace. |
| D4 | Custom base URL is allowed. Loopback, private, link-local, and unspecified addresses are blocked unless `OMED_LLM_ALLOW_PRIVATE_URLS=true`. |
| D5 | API key encrypted with AES-256-GCM. Master key `OMED_SECRET_KEY` is separate from `OMED_AUTH_SECRET`. |
| D6 | `apiKey` is write-only over the API. Responses carry `hasKey` and `keyHint` only. |
| D7 | Missing `OMED_SECRET_KEY` does not stop startup. Write and test endpoints return `503`. |
| D8 | Provider rows are hard-deleted so ciphertext does not linger. |
| D9 | Roles follow the `fin`/`cms` pattern: `agent:viewer` (list and read providers) and `agent:admin` (everything, inherits viewer). Permission object `agent:provider`, actions `read`, `create`, `update`, `delete`. |

## 4. Architecture

```
apps/api/internal/platform/
├── llmx/      # Adapter interface, drivers, capabilities, URL guard
└── secretx/   # AES-GCM encrypt/decrypt

apps/api/internal/agent/
├── domain/      # authz roles, errors, kind constants
├── model/       # request/response DTOs
├── repository/  # Ent access, workspace-filtered
├── service/     # CRUD, set default, test connection, ResolveAdapter
├── controller/  # Fiber handlers, Swaggo annotations
├── test/        # integration tests
├── module.go
└── routes.go
```

Dependency direction follows AGENTS.md: `controller -> service -> repository`. `llmx` and `secretx` know nothing about Ent, workspaces, or HTTP routes. The `agent` service composes them. Other modules obtain adapters only through a narrow contract on the `agent` service (`ResolveAdapter`), never through its repository.

Flow:

```
caller -> agent.service.ResolveAdapter(ctx, providerID?)   # workspace ID from context
       -> repository (workspace filter) -> secretx.Decrypt -> llmx.New(kind, cfg) -> Adapter
```

`ResolveAdapter` does not check `agent:*` roles. It is an internal contract: the calling feature's controller gates access with its own permission (for example a `fin` role), so a member can use the workspace's provider through a feature without holding any `agent:*` role. Consequence: the workspace owner of the key bears the cost of all such use.

## 5. `llmx`

```go
type Adapter interface {
    Chat(ctx context.Context, req Request) (*Response, error)
    Stream(ctx context.Context, req Request) iter.Seq2[Event, error]
    Capabilities() Capabilities
}
```

- `Request`: model, messages (text and image parts), tool definitions, response schema, max tokens, temperature.
- `Response`: assistant message, tool calls, usage, stop reason.
- `Event`: text delta, tool-call delta, done with usage.
- `Capabilities`: `Vision`, `ToolCalling`, `StructuredOutput`, `Streaming`. Callers check flags before using a feature.
- Tool types exist in the interface and each driver maps them, so the later tool loop does not change the contract. The loop itself is not built.
- Errors map to sentinels: `ErrAuth`, `ErrRateLimit`, `ErrModelNotFound`, `ErrBadRequest`, `ErrUnavailable`. Messages are sanitized: no keys, no request headers.
- Drivers: `openai` (configurable base URL, covers OpenAI, OpenRouter, Ollama, Groq and similar), `anthropic`, `gemini`, `memory` (scripted responses for tests, same role as `mailx` and `s3x` memory drivers).
- Default capabilities per `kind` are built in; an admin may override them per provider.

### SSRF guard

1. On save: parse the URL, require `https` (or `http` only when private URLs are allowed), reject userinfo.
2. On every outbound call: a custom `net.Dialer` `Control`/`DialContext` checks the **resolved IP** against the blocked ranges. This defeats DNS rebinding that string validation misses.
3. HTTP redirects are disabled for provider calls. A redirect response is treated as an error.
4. `OMED_LLM_ALLOW_PRIVATE_URLS=true` disables the range check (for local Ollama), not the scheme or userinfo checks.

## 6. `secretx`

- AES-256-GCM, random 12-byte nonce per encryption.
- Stored form: `v1.<base64url(nonce || ciphertext || tag)>`. The version prefix leaves room for key IDs and rotation later.
- Master key: base64-encoded 32 bytes from `OMED_SECRET_KEY`. Invalid length or encoding fails construction with a clear error; an unset key yields a disabled instance whose methods return a typed `ErrDisabled`.
- Associated data: `workspace_id`. A ciphertext copied into another workspace's row fails to decrypt.

## 7. Data model

Ent schema `apps/api/ent/schema/agent_provider.go`, `entsql.Schema("agent")`. Uses `WorkspaceMixin` and `AuditMixin`, like `fin_period.go`. No `SoftDeleteMixin`. Declare `id` as `field.Int64("id")` like existing schemas.

| Field | Notes |
| --- | --- |
| `name` | required, unique per workspace |
| `kind` | enum `openai_compat`, `anthropic`, `gemini` |
| `base_url` | optional; driver default when empty |
| `model` | required |
| `api_key_enc` | required, `secretx` output |
| `key_hint` | last 4 characters of the key |
| `is_default` | bool, default false |
| `capabilities` | JSON, overrides on top of kind defaults |

Indexes: unique `(workspace_id, name)`; partial unique `(workspace_id)` where `is_default` to guarantee at most one default. Setting a default runs in one transaction (`platform/entx` `WithTx`): clear the old default, set the new one.

**Tenant isolation (required):** `internal/core/mutator.go` filters queries with a hard-coded `switch` over entity types, and the filter is fail-open for types not listed. Add `ent.TypeAgentProvider` (use the actual generated constant) to `workspaceInterceptor.filter` and extend `mutator_test.go` to prove it is filtered. Repositories should also add an explicit `workspace_id` predicate as defense in depth.

After editing the schema: `go generate ./ent`, then a versioned migration with `go run ./cmd/migrate add_agent_provider`. Commit `ent/*` and `migrations/*`.

## 8. API and authorization

Roles registered with `authz.AddRole` in `init()` of `agent/domain/authz.go` (mirror `fin/domain/authz.go`):

- `agent:viewer`: `agent:provider` `read`.
- `agent:admin`: inherits `agent:viewer`, plus `agent:*` wildcard. Do **not** grant `core:file` permissions; the module does not use files.

Routes, workspace-scoped, guarded with `az.Require(domain.ObjProvider, <action>)`. Static routes (`/test`) are registered before `/:id`, as `fin` does.

```
GET    /workspaces/:wid/agent/providers                  read
POST   /workspaces/:wid/agent/providers                  create
POST   /workspaces/:wid/agent/providers/test             create   # test an unsaved config
GET    /workspaces/:wid/agent/providers/:id              read
PATCH  /workspaces/:wid/agent/providers/:id              update
DELETE /workspaces/:wid/agent/providers/:id              delete
POST   /workspaces/:wid/agent/providers/:id/default      update
POST   /workspaces/:wid/agent/providers/:id/test         update
```

- A provider ID from another workspace returns `404`. A member without a role gets `403`; unauthenticated gets `401`.
- Response never contains `api_key_enc` or the key. It contains `hasKey` and `keyHint`.
- `PATCH` without `apiKey` keeps the stored key.
- Deleting the default leaves no default. `ResolveAdapter` without an ID then returns a "no default provider" domain error.
- Test connection sends a minimal prompt and reports auth or model errors using the `llmx` sentinels, mapped to domain errors.
- Test endpoints are rate limited per user with an in-memory token bucket. Limitation: per-process, so multi-replica deployments get N times the limit. Acceptable for v1 because Redis is not used yet.
- When `OMED_SECRET_KEY` is unset, create, update, and test return `503` with a clear message. List and delete still work.

Swaggo annotations: `@Tags Agent Providers`, `@Security BearerAuth`, error responses. Regenerate `apps/api/docs/*` and `packages/openapi/src/api.ts`.

## 9. Config and deploy

New env vars (`apps/api/internal/config/config.go`, `.env.example`, README table):

| Variable | Kind | Notes |
| --- | --- | --- |
| `OMED_SECRET_KEY` | secret | base64, 32 bytes; optional at startup |
| `OMED_LLM_ALLOW_PRIVATE_URLS` | plain | default `false` |

Helm (AGENTS.md section 6): `OMED_LLM_ALLOW_PRIVATE_URLS` in `values.yaml`, `values.schema.json`, `omed.api.env` helper; `OMED_SECRET_KEY` in `templates/secret.yaml` and the chart README Secrets table. Run `helm lint` and `kubeconform` as specified.

Repo housekeeping: add `agent` to the commitlint scope list; mention the `agent` schema in AGENTS.md section 3 and the module list in README (docs scope).

## 10. Web

`apps/web/app/modules/agent/` with `module.ts` (launcher metadata) and `route.ts` (`prefix("agent", ...)`, index redirects to `providers`). Also:

- `hooks/use-agent-permissions.ts` mirroring `use-fin-permissions.ts` (`agent:viewer|admin`), with tests.
- Role labels `role.agent.viewer` and `role.agent.admin` (see `lib/roles.ts`) in `i18n/en.ts` and `i18n/id.ts`.

One screen, `/agent/providers`:

- Table: name, kind, model, default marker, key hint. Viewers get a read-only table.
- Drawer for create and edit (admin only): kind, base URL, model, API key (password input, write-only), capability toggles.
- Actions (admin only): Test connection, Set default, Delete with confirm.
- Rules: types from `@omed/openapi` via `openapi-fetch`; Ant Design theme and tokens from BRANDING.md; strings in both i18n files; Vitest tests next to files.

## 11. Testing

- Unit, `secretx`: roundtrip, tampered ciphertext, wrong key, wrong AAD, disabled instance.
- Unit, `llmx`: each driver against `httptest` servers (request mapping, error mapping, streaming); URL guard including IPv6, DNS rebinding, and redirect to a private IP.
- Unit, `agent`: repository and service with mocks (`go:generate` + mockgen like existing services); default-switch transaction; capability merge; roles exist with expected inheritance (see `platform/authz/registry.go`).
- Unit, `core`: `mutator_test.go` proves `agent_provider` queries are workspace-filtered.
- Integration (`testutil.ApiTestSuite`): CRUD, permission matrix (viewer read-only, admin manages, no role `403`, unauthenticated `401`), cross-workspace isolation (`404`), key absent from every response, `503` without master key, test endpoint with the memory driver.
- E2E (Cucumber + Playwright): admin adds provider, sets default, tests connection, deletes; viewer sees the list without management actions (`addMember` with `agent:viewer`, see `features/fin/permissions.feature`). Page objects under `e2e/pages/agent/`.

## 12. Risks and assumptions

- **Branch**: `wip` does not exist on the remote; this design was written against `main` at `7ce310f`.
- **SDKs**: package paths and versions for `openai-go` and `go-genai` have not been verified; only `anthropics/anthropic-sdk-go` was confirmed. Verify before adding dependencies.
- **Streaming parity**: providers differ in tool-call delta formats. Drivers normalize to `Event`; parity is covered by `httptest` fixtures, not live calls.
- **Capability defaults** can be wrong for a given model (for example a text-only model on an `openai_compat` endpoint). Admin overrides and the test endpoint mitigate this; no automatic detection.
- **Shared cost and exposure**: every feature allowed to call `ResolveAdapter` spends the workspace's key. `keyHint` (last 4 characters) is visible to anyone with `agent:viewer`; accepted as low risk.
- **Master key loss** makes stored keys unrecoverable; admins must re-enter them. Documented in the chart README.
