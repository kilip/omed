# Design Spec: MCP Foundation (`platform/mcpx` Transport and Personal Access Tokens)

- **Date:** 2026-11-01
- **Status:** Accepted
- **Modules:** `core` (personal access tokens), `platform/mcpx` (transport), `apps/web` module `core` (token page)
- **Author:** Claude (pair programming with the repository owner)
- **Related:** [`2026-11-01-fin-mcp-integration-design.md`](./2026-11-01-fin-mcp-integration-design.md) defines the three MCP sub-projects and is the consumer of this one. [`2026-10-30-agent-provider-design.md`](./2026-10-30-agent-provider-design.md) keeps the MCP server out of the `agent` module; this spec does not change that.

---

## 1. Intent & The Why

This is **sub-project 2** of the MCP effort described in the `fin` MCP spec:

| # | Sub-project | Output |
| --- | --- | --- |
| 1 | Spike: Streamable HTTP transport of go-sdk on Fiber v3 | Folded into this spec (section 5.5, plan task 1) |
| 2 | **This spec:** Foundation | Transport, token auth, PAT lifecycle in `core`, token page |
| 3 | `fin` tools | Read-only financial Q&A, drafts later |

The outcome: a signed-in user creates a personal access token (PAT) in the dashboard, pastes it into an external MCP client such as Claude Desktop, and the client calls module tools at one endpoint, `/mcp`, under the same permissions the user has in the dashboard.

Decisions inherited from the `fin` MCP spec:

- The MCP server is a product feature for end users, not developer tooling.
- Omed stays a modular monolith. Modules define tools; one shared server exposes them.
- PAT first, OAuth deferred.
- The workspace is an explicit tool argument, so a PAT belongs to a user, not to a workspace.
- The SDK is `github.com/modelcontextprotocol/go-sdk`, imported only from `platform/mcpx` and tests.

Decisions taken while writing this spec:

- A PAT is valid **only** for `/mcp`. It is never accepted by REST.
- PATs are created, listed, and revoked through an API in `core`, plus a minimal page in the dashboard. The token is shown once.
- Expiry is mandatory: 30, 90, or 365 days, default 90. There are no non-expiring tokens.
- Tokens are opaque random strings. Only a SHA-256 hash is stored.
- An expired token gets an explicit error that links to the token page. There is no refresh and no regenerate action: the user creates a new token.
- `/mcp` is always on. There is no feature flag and no new environment variable.
- The spike (sub-project 1) is folded into this one: the first plan task verifies the transport.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **PAT lifecycle in `core`:** create, list, revoke, and verify, with a per-user limit on active tokens.
- **`/mcp` endpoint** served by `platform/mcpx` behind a Fiber adapter, authenticating PATs and populating the caller identity that the existing permission wrapper reads.
- **Module wiring:** the boot phase builds the registrar once, calls `RegisterMCP` on every module implementing `config.HasMCP`, and mounts `/mcp`.
- **Dashboard token page:** list, create (name and expiry), one-time reveal with copy, revoke.
- **Unblock** Task 7 of the `fin` MCP plan: an endpoint and a Go test helper to create a PAT.

### Non-Goals
- **OAuth** authorization server, refresh tokens, and token regeneration.
- **Idempotency keys for phase B drafts and the ledger tool.** They return to the `fin` phase B and later specs.
- **An internal agent path.** The `agent` module has no tool-calling loop yet. When it does, it will not use PATs (it runs in-process under the session user); that design belongs to the agent tool-loop spec.
- **Per-token scopes**, per-workspace tokens, and `last_used_at` tracking.
- **Changes to the registrar's permission wrapper**, error mapping, or size cap, except what wiring requires.
- **New environment variables or Helm changes.**

### Anti-Goals
- Never store, log, or return a token in plaintext after creation. `slog` calls must not include the token or the `Authorization` header.
- Never accept a PAT on REST routes, and never accept a session cookie on `/mcp`.
- Never reveal through the response whether a token is unknown, malformed, or revoked.
- Never let a PAT create, list, or revoke PATs.
- Never import `core` from `mcpx`: `mcpx` receives a verifier function and a URL string.

---

## 3. Constraints

- **Layering (AGENTS.md section 3).** `controller -> service -> repository`. Only repositories use Ent. `mcpx` reaches `core` only through an injected verifier.
- **Database.** The new Ent schema lives in `apps/api/ent/schema/core_*.go` with `entsql.Schema("core")`. After editing it, run `go generate ./ent` and generate a versioned migration (`go run ./cmd/migrate <name>`), then commit `ent/*` and `migrations/*`.
- **Existing auth middleware.** `core/middleware/auth.go` treats a Bearer value as a **session** token and validates it with `SessionService.ValidateSession`. A PAT therefore already fails on REST. `/mcp` is not in its bypass list and must be added.
- **Existing contract.** `config.HasMCP` (`RegisterMCP(r *mcpx.Registrar) error`) already exists in `config/contracts.go`. The registrar already reads the caller from `req.Extra.TokenInfo.UserID`.
- **Web URL.** `OMED_WEB_URL` (`Config.WebUrl`, default `http://localhost:3000`) already exists and is used for invitation links. The token page link is derived from it.
- **Web module.** The page lives in `apps/web/app/modules/core/`, with its route in `route.ts`. API types come from `@omed/openapi` through `openapi-fetch`; styling follows `BRANDING.md`.
- **Contract files.** After changing Swaggo annotations, regenerate and commit `apps/api/docs/*` and `packages/openapi/src/api.ts`.
- **Language.** English, Markdown (AGENTS.md section 1).

---

## 4. Approaches Considered

### Approach 1: Opaque random token, hash stored (Selected by the owner)
`omed_pat_<random>`; the database holds a SHA-256 hash and a short display prefix. Verification is one indexed lookup. Revocation and expiry take effect immediately.

### Approach 2: Self-contained signed token (JWT-like)
No lookup on verification, but revocation needs a denylist, which brings the state back. Rejected.

### Approach 3: Reuse the session table with a `kind` column
Less new code, but it mixes browser-session lifetime and cookie semantics with API tokens, and `ValidateSession` could then accept PATs on REST. Rejected.

---

## 5. Detailed Design

### 5.1 Components

```
apps/api/ent/schema/core_personal_access_token.go     # new schema, entsql.Schema("core")
apps/api/internal/core/
├── model/access_token.go          # request/response DTOs
├── repository/access_token.go     # Ent access
├── service/access_token.go        # create, list, revoke, verify, limit
├── controller/access_token.go     # /users/me/access-tokens
└── test/                          # integration tests
apps/api/internal/platform/mcpx/
└── handler.go                     # Handler(): StreamableHTTPHandler + bearer auth + Fiber adapter
apps/api/internal/config/app.go    # boot wiring
apps/web/app/modules/core/         # route, queries, components for the token page
e2e/features/core/access-tokens.feature
```

File names are indicative; the builder follows the naming already used in each directory (section 8).

### 5.2 Data model

Entity `PersonalAccessToken` in schema `core`:

| Field | Notes |
| --- | --- |
| `id` | primary key, as other core entities |
| `user` | required edge to the owning user |
| `name` | required, human label |
| `token_hash` | SHA-256 of the full token, unique and indexed |
| `prefix` | first characters of the token for display, never enough to use it |
| `expires_at` | required |
| `revoked_at` | nullable; revocation is a soft state |
| `created_at` | as other core entities |

Status is derived, not stored: `revoked` if `revoked_at` is set, else `expired` if `expires_at` is in the past, else `active`.

### 5.3 Token format and lifecycle

- **Generation:** 32 bytes from `crypto/rand`, base64url, prefixed `omed_pat_`. The plaintext exists only in the create response.
- **Storage:** SHA-256 of the full string, hex or bytes (builder's choice), plus the display prefix.
- **Limit:** at most 20 **active** tokens per user, a constant in code. Creating the 21st returns a validation error that tells the user to revoke one.
- **Revocation:** sets `revoked_at`; repeating it is a no-op that still succeeds.
- **Service surface:** `Create(ctx, userID, req)`, `List(ctx, userID)`, `Revoke(ctx, userID, id)`, `Verify(ctx, token)`. `Verify` returns the owner's user id, or one of two distinct errors: expired, or invalid (unknown, malformed, or revoked).

### 5.4 REST API (session-authenticated only)

| Method and path | Behavior |
| --- | --- |
| `POST /users/me/access-tokens` | Body `name`, `expires_in_days` in `{30, 90, 365}`. Returns 201 with `id`, `name`, `prefix`, `expires_at`, and `token` (plaintext, this response only). |
| `GET /users/me/access-tokens` | The caller's tokens without secrets: `id`, `name`, `prefix`, `expires_at`, `created_at`, `status`. |
| `DELETE /users/me/access-tokens/:id` | Revoke. A token owned by another user returns `NOT_FOUND`. |

Controllers carry Swaggo annotations. The routes sit behind the existing session middleware, which rejects PATs by construction, so a PAT cannot manage PATs.

### 5.5 `/mcp` transport and authentication

- **Handler.** `mcpx` builds `mcp.NewStreamableHTTPHandler` with `Stateless: true` and `JSONResponse: true`, so no event stream has to cross the Fiber adapter, and wraps it with the SDK's bearer-token middleware. The adapter bridges Fiber v3 to `net/http`.
- **Verifier.** `mcpx` accepts a function `func(ctx, token string) (userID int64, expiresAt time.Time, err error)` (exact shape fixed in the plan) and a `TokenSettingsURL` string. Boot wires the function to `AccessTokenService.Verify` and the URL to `{cfg.WebURL}/core/access-tokens`. The verifier sets `TokenInfo.UserID`, which the existing wrapper reads.
- **Middleware order.** The global session middleware bypasses `/mcp`. `/mcp` reads only `Authorization: Bearer ...`; it never reads cookies.
- **Spike (plan task 1).** Before building on it, verify with a Fiber-served test: the handler works with `Stateless` and `JSONResponse` behind the adapter, `initialize`, `tools/list`, and `tools/call` round trip, and a failed bearer check returns 401 with the header described in 5.6. Record the result in this spec. Symbol names come from the SDK documentation and must be confirmed with `go doc`.

### 5.6 Errors at the transport

All are returned before any MCP processing.

| Case | Response |
| --- | --- |
| Token expired | 401, `WWW-Authenticate: Bearer error="invalid_token", error_description="Token expired. Create a new access token: <TokenSettingsURL>"`, JSON body with code `TOKEN_EXPIRED` and the same message |
| Missing, malformed, unknown, or revoked | 401, `WWW-Authenticate: Bearer error="invalid_token"`, generic "invalid token" body, no distinction between causes |

- The expired message is returned only for a token whose hash matched a stored record, so it discloses nothing to someone guessing tokens.
- The real cause is logged with `slog` (without the token). `TOKEN_EXPIRED` is registered with `base.AddToErrorMap`.
- Whether an MCP client shows the message or the link as clickable depends on the client and is outside Omed's control.

### 5.7 Dashboard token page

Route `/core/access-tokens` (module `core`): a table with name, prefix, expiry, and status; a create modal with name and expiry choice (30, 90, 365, default 90); a one-time reveal view with a copy button and a warning that the token will not be shown again; a revoke action with confirmation. The page is available to every signed-in user. Strings are in i18n `id` and `en`. Types come from regenerated `@omed/openapi`.

### 5.8 Boot wiring

1. Build `mcpx.NewRegistrar(...)` once, with the existing `Authorizer` and `MembershipResolver`.
2. For each module that implements `config.HasMCP`, call `RegisterMCP`.
3. Mount `mcpx.Handler(...)` at `/mcp` with the verifier and `TokenSettingsURL`.

### 5.9 Impact on existing code, tests, and docs

- New: the schema, migration, `core` layers for access tokens, `mcpx/handler.go`, the web page, the e2e feature.
- Changed: `core/middleware/auth.go` (bypass `/mcp`), `config/app.go` (wiring), `core` routes, `apps/web` module `core` routes and i18n.
- Docs: README gains the `/mcp` endpoint and the token page; `apps/api/docs/*` and `packages/openapi/src/api.ts` are regenerated.
- Deployment: no new environment variables, so no Helm change expected. The chart already passes `OMED_WEB_URL` to the API (`api.webUrl`, else the web ingress host).
- Test support: a Go helper that creates a PAT for a user, named in the plan, for Task 7 of the `fin` MCP plan.

---

## 6. Testing Strategy

- **Go unit tests** (next to the files): service (create, defaults, limit of 20, list status, revoke idempotency and ownership, verify for active, expired, revoked, and unknown tokens, hash only stored) and repository.
- **Go integration tests** (`core/test/`): token lifecycle through the API; a PAT is rejected on REST; a session cookie is rejected on `/mcp`; a PAT cannot call the token endpoints; an expired token returns the explicit error with the link; a revoked token returns the generic error and no tool runs.
- **`mcpx` tests:** handler tests with `httptest` and the SDK in-memory client, including the bearer failure shapes of section 5.6.
- **E2E (Cucumber and Playwright):** one scenario in group `core` that creates a token (seeing it once) and revokes it. Page objects live in `pages/core/`.

---

## 7. Acceptance Criteria

1. A signed-in user can create a token with a name and a 30, 90, or 365 day expiry, sees the plaintext exactly once, and can list and revoke tokens. The database holds no plaintext.
2. A valid PAT can run `initialize`, `tools/list`, and `tools/call` at `/mcp`, and the tool handlers see that user.
3. A PAT sent to any REST route returns 401. A session cookie sent to `/mcp` returns 401.
4. An expired PAT returns 401 with `TOKEN_EXPIRED` and a link to `{OMED_WEB_URL}/core/access-tokens`. Unknown, malformed, and revoked tokens return the same generic 401.
5. A user cannot revoke or see another user's token, and a PAT cannot call the token endpoints.
6. The 21st active token is refused with an actionable message.
7. Logs never contain a token or the `Authorization` header.
8. Task 7 of the `fin` MCP plan can create a PAT with the test helper and run its integration tests against `/mcp`.
9. `bun run lint`, `bun run typecheck`, `cd apps/api && go test ./...`, and `bun run e2e` pass.

---

## 8. What's Left to the Builder

- File and type names within each directory's existing conventions.
- Hash encoding (hex or bytes), the display prefix length, and the maximum name length.
- The exact shape of the verifier function and the Fiber-to-`net/http` bridging mechanism, as confirmed by the spike.
- The wording of user-facing strings and the i18n keys.
- The name of the PAT test helper.

---

## 9. Open Items to Verify Before Implementation

1. **Transport spike:** `Stateless` plus `JSONResponse` behind the Fiber v3 adapter, and the exact SDK symbols (`go doc`). `go.mod` already contains the SDK (v1.8.0) because the registrar uses it.
2. **Origin and CORS:** whether the SDK validates the `Origin` header on Streamable HTTP, and whether browser-based MCP clients need CORS on `/mcp`. Decide based on the spike.
3. **Rate limiting:** the existing limiter (general and sensitive tiers) may or may not cover `/mcp`. Decide whether `/mcp` falls under the general tier and whether failed PAT checks count against the sensitive tier.
4. **Client behavior:** whether common MCP clients surface the `error_description` text to the user.
