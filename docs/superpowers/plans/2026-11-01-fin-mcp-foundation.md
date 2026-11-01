# MCP Foundation (`/mcp` Transport and Personal Access Tokens) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A signed-in user creates a personal access token (PAT) in the dashboard and an external MCP client uses it at `/mcp` to call module tools under the user's own permissions.

**Architecture:** `core` owns the PAT lifecycle (Ent schema, repository, service, REST controller). `platform/mcpx` gains a Fiber handler that authenticates a bearer PAT through an injected verifier and serves the go-sdk Streamable HTTP handler. Boot wires the two, builds the registrar once, calls `RegisterMCP` on every `config.HasMCP` module, and mounts `/mcp`. The dashboard gets a token page in module `core`.

**Tech Stack:** Go 1.27 (Fiber v3.5, Ent, Atlas, testify suites), `github.com/modelcontextprotocol/go-sdk` v1.8.0, React Router + Ant Design + openapi-fetch, Vitest, Cucumber + Playwright.

**Spec:** `docs/superpowers/specs/2026-11-01-mcp-foundation-design.md` (parent: `2026-11-01-fin-mcp-integration-design.md`).

## Global Constraints

- The SDK is imported only from `apps/api/internal/platform/mcpx` and `_test.go` files. `mcpx` never imports `core`; `core` never imports `mcpx`. The mapping between their errors lives in `apps/api/internal/config/mcp.go`.
- Layering: `controller -> service -> repository`; only repositories use Ent. Repositories return domain entities; controllers map domain to DTOs.
- The Ent schema sets `entsql.Schema("core")`. After editing it: `cd apps/api && go generate ./ent`, then `go run ./cmd/migrate <name>`; commit `ent/*` and `migrations/*`.
- Token: `omed_pat_` + base64url of 32 bytes from `crypto/rand`. Stored as SHA-256 hex via the existing `hashToken` in `core/service/session.go`, plus a display prefix. At most **20 active** tokens per user. Expiry **30, 90, or 365** days, default **90**. The plaintext is returned only by create.
- A PAT is valid only at `/mcp`. A session cookie or session token is never accepted at `/mcp`. A PAT is never accepted on REST and cannot manage PATs.
- An expired token returns 401 with code `TOKEN_EXPIRED` and the message `Token expired. Create a new access token: <OMED_WEB_URL>/core/access-tokens`. Every other failure returns the same generic 401 `invalid token`. Revoked wins over expired (generic).
- A token or the `Authorization` header is never logged. No new environment variables, so no Helm change.
- JSON field names are camelCase, as in existing DTOs (the spec's snake_case in section 5.4 is corrected in Task 8).
- Web: API types from `@omed/openapi` (never hand-written), no hardcoded hex values, strings in i18n `id` and `en`.
- Commits: `type(scope): <emoji> subject` per AGENTS.md (scopes `api`, `web`, `e2e`, `docs`), one logical change each, ending with the attribution trailers the executing session requires.

## Already in the repo (do not rebuild)

`mcpx.Registrar` (permission wrapper, error mapping, size cap), `config.HasMCP`, and `FinModule.RegisterMCP` exist. Nothing at boot builds the registrar or calls `RegisterMCP` yet, and nothing mounts `/mcp`. The session middleware treats any Bearer value as a session token, so PATs already fail on REST.

## Review Focus

- RF1: `Authorization` header variants: missing, `Basic x`, `Bearer`, `Bearer ` with spaces, lowercase `bearer <valid>` (scheme is case-insensitive, so accepted). All bad ones return the generic 401 and never call the verifier. Pinned in Task 1.
- RF2: a token name that is whitespace-only is rejected; a 100-rune name with emoji or Indonesian text is accepted and a 101-rune name rejected (count runes, not bytes). Pinned in Task 3.
- RF3: deleting a user deletes the user's tokens, and the old token no longer verifies. Pinned in Task 2.
- RF4: `/mcpx` and `/mcp-foo` are not treated as `/mcp` by the session-middleware bypass; `/mcp` and `/mcp/` are. Pinned in Task 5.
- RF5: a user with no tokens gets `data: []`, not `null`. Pinned in Task 4.

---

### Task 1: `mcpx` Fiber handler with bearer verification (includes the transport spike)

**Files:**
- Create: `apps/api/internal/platform/mcpx/handler.go`, `apps/api/internal/platform/mcpx/handler_test.go`
- Modify: `apps/api/internal/platform/mcpx/errors.go`, `docs/superpowers/specs/2026-11-01-mcp-foundation-design.md`

**Interfaces:**
- Produces:
  - `var ErrTokenExpired = errors.New("TOKEN_EXPIRED")`, `var ErrTokenInvalid = errors.New("TOKEN_INVALID")` (in `errors.go`)
  - `type VerifiedToken struct{ UserID int64; ExpiresAt time.Time }`
  - `type TokenVerifier func(ctx context.Context, token string) (VerifiedToken, error)`
  - `type HandlerConfig struct{ Verifier TokenVerifier; TokenSettingsURL string; Logger *slog.Logger }`
  - `func Handler(r *Registrar, cfg HandlerConfig) fiber.Handler`
- Consumes: `Registrar.Server() *mcp.Server`.

- [ ] **Step 1: Confirm symbols (the spike).** Run `go doc` for `github.com/gofiber/fiber/v3/middleware/adaptor`, `mcp.NewStreamableHTTPHandler`, `mcp.StreamableHTTPOptions`, `mcp.StreamableClientTransport`, and the SDK `auth` package (`TokenInfo` and the way a verified identity reaches `CallToolRequest.Extra.TokenInfo`). Fix any name in this task that differs.
- [ ] **Step 2: Write failing tests** in `handler_test.go`, reusing the harness of `registrar_test.go` (authorizer with test roles, stub resolver). Serve a Fiber app with `Handler` on a loopback listener, register a stub tool `fin_ping` needing `fin:account read`, and connect with `mcp.StreamableClientTransport` and an `Authorization` round tripper:
  - `Test_Handler_ValidToken_ListToolsAndCallSeeUser`: `ListTools` returns `fin_ping`; the stub `MembershipResolver` is called with user id `7`, taken from `VerifiedToken{UserID: 7}`. This proves `Stateless` + `JSONResponse` work behind Fiber.
  - `Test_Handler_AuthorizationVariants_Return401Generic` (RF1): table of missing, `Basic x`, `Bearer`, `Bearer   ` expecting 401, body code `UNAUTHORIZED` message `invalid token`, header `WWW-Authenticate: Bearer error="invalid_token"`, verifier call count `0`; plus `bearer <valid>` expecting success.
  - `Test_Handler_ExpiredToken_Returns401WithLink`: verifier returns `ErrTokenExpired`; assert status 401, code `TOKEN_EXPIRED`, message exactly `Token expired. Create a new access token: https://app.example.com/core/access-tokens` for `TokenSettingsURL: "https://app.example.com/core/access-tokens"`, and the same text in `error_description`.
  - `Test_Handler_InvalidAndUnknownVerifierErrors_Return401Generic`: `ErrTokenInvalid` and `errors.New("db down")` both give the generic body; the unknown error is logged.
  - `Test_Handler_SessionCookieOnly_Returns401`: cookie `omed_session=x`, no header, verifier count `0`.
  - `Test_Handler_NeverLogsToken`: the log buffer contains neither the token nor `Authorization`.
- [ ] **Step 3: Run** `cd apps/api && go test ./internal/platform/mcpx/ -run Handler -v`. Expected: FAIL (`Handler` undefined).
- [ ] **Step 4: Implement** `Handler` in `handler.go`: a Fiber handler that parses the bearer (case-insensitive scheme), calls `cfg.Verifier`, writes the 401 shapes above using `httpx.ErrorResponse`, makes the verified identity reach `Extra.TokenInfo.UserID` as a decimal string (what `defaultUserID` parses), and delegates to `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return r.Server() }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})` through the Fiber adaptor. Keep the bearer check in Fiber so the 401 bodies are ours; use the SDK's context helper from step 1 to inject the identity.
- [ ] **Step 5: Run** the command of step 3. Expected: PASS.
- [ ] **Step 6: Record the spike result** in the spec: replace section 9 item 1 with the verified outcome (works as designed, or the fallback used and why), and note any `Origin`/CORS finding for item 2.
- [ ] **Step 7: Commit**: `feat(api): ✨ add mcpx fiber handler with bearer token verification`.

---

### Task 2: PAT schema, migration, domain, and repository

**Files:**
- Create: `apps/api/ent/schema/core_personal_access_token.go`, `apps/api/internal/core/domain/access_token.go`, `apps/api/internal/core/repository/access_token.go`, `apps/api/internal/core/repository/access_token_test.go`
- Modify: `apps/api/internal/core/domain/error.go`; generated `apps/api/ent/*`, new file in `apps/api/migrations/`

**Interfaces:**
- Produces:
  - Ent entity `PersonalAccessToken` (schema `core`): `id`, `user_id` (required edge to `User`, `OnDelete(Cascade)`), `name`, `token_hash` (unique, indexed), `prefix`, `expires_at`, `revoked_at` (optional, nillable), `created_at`; index on `user_id`.
  - `type AccessTokenStatus string` with `AccessTokenActive`, `AccessTokenExpired`, `AccessTokenRevoked` (`"active"`, `"expired"`, `"revoked"`)
  - `type AccessToken struct{ ID, UserID int64; Name, TokenHash, Prefix string; ExpiresAt, CreatedAt time.Time; RevokedAt *time.Time }` and `func (t AccessToken) Status(now time.Time) AccessTokenStatus` (revoked wins over expired)
  - Errors in `domain`: `ErrTokenExpired` (`TOKEN_EXPIRED`, 401), `ErrTokenInvalid` (`TOKEN_INVALID`, 401), `ErrTokenLimitReached` (`TOKEN_LIMIT_REACHED`, 422), `ErrTokenInvalidExpiry` (`TOKEN_INVALID_EXPIRY`, 422), `ErrTokenInvalidName` (`TOKEN_INVALID_NAME`, 422), registered in `init()` with `base.AddToErrorMap`
  - `type AccessTokenRepository struct`; `func NewAccessTokenRepository(cli *ent.Client) AccessTokenRepository`; methods:
    - `Create(ctx, t domain.AccessToken) (domain.AccessToken, error)`
    - `FindByHash(ctx, hash string) (domain.AccessToken, error)` (`base.ErrNotFound` when absent)
    - `ListByUser(ctx, userID int64) ([]domain.AccessToken, error)` (newest first, non-nil slice)
    - `CountActiveByUser(ctx, userID int64, now time.Time) (int, error)`
    - `Revoke(ctx, userID, id int64, at time.Time) error` (idempotent; `base.ErrNotFound` when the token is not the user's)

- [ ] **Step 1: Write failing tests** in `access_token_test.go`, following the suite style of `session_test.go`: `Test_CreateThenFindByHash`; `Test_FindByHash_Unknown_ReturnsNotFound`; `Test_ListByUser_OnlyOwn_NewestFirst`; `Test_CountActiveByUser_ExcludesExpiredAndRevoked`; `Test_Revoke_Idempotent_KeepsFirstRevokedAt`; `Test_Revoke_OtherUsersToken_ReturnsNotFound`; `Test_DeleteUser_CascadesTokens` (RF3: after deleting the user, `FindByHash` returns `base.ErrNotFound`).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/repository/ -run AccessToken -v`. Expected: FAIL (build error).
- [ ] **Step 3: Implement** the schema (mirror `core_session.go`), run `cd apps/api && go generate ./ent`, then `go run ./cmd/migrate add_personal_access_token`; implement `domain/access_token.go`, the error registrations, and the repository (mirror `repository/session.go`, including a `toAccessTokenDomain` mapper).
- [ ] **Step 4: Run** the command of step 2 and `cd apps/api && atlas migrate validate --env local`. Expected: PASS and a valid checksum.
- [ ] **Step 5: Commit**: `feat(api): ✨ add personal access token schema and repository`.

---

### Task 3: `AccessTokenService`

**Files:**
- Create: `apps/api/internal/core/model/access_token.go`, `apps/api/internal/core/service/access_token.go`, `apps/api/internal/core/service/access_token_test.go`
- Modify: `apps/api/internal/core/module.go`

**Interfaces:**
- Consumes: `repository.AccessTokenRepository`, `hashToken(string) string` (same package).
- Produces:
  - `model.CreateAccessTokenRequest{ Name string \`json:"name"\`; ExpiresInDays int \`json:"expiresInDays"\` }` (0 means the default 90)
  - Constants: `AccessTokenPrefix = "omed_pat_"`, `MaxActiveAccessTokens = 20`, `MaxAccessTokenNameLen = 100`, `DefaultAccessTokenExpiryDays = 90`
  - `type AccessTokenService interface { Create(ctx, userID int64, req model.CreateAccessTokenRequest) (domain.AccessToken, string, error); List(ctx, userID int64) ([]domain.AccessToken, error); Revoke(ctx, userID, id int64) error; Verify(ctx, token string) (domain.AccessToken, error) }` (`Create` returns the plaintext as its second value)
  - `func NewAccessTokenService(tokens repository.AccessTokenRepository) AccessTokenService` (unexported `now func() time.Time` field, overridden in tests)
  - `CoreModule.AccessTokenService service.AccessTokenService`, built in `New`

- [ ] **Step 1: Write failing tests** (testify suite on the real test DB, as other service tests):
  - `Test_Create_ReturnsPlaintextOnceAndStoresOnlyHash`: plaintext has prefix `omed_pat_`; `FindByHash(hashToken(plaintext)).TokenHash` equals the hash; no stored field equals the plaintext; `Prefix` is a strict prefix of the plaintext.
  - `Test_Create_ExpiryDays`: `0` gives 90 days, `30`/`90`/`365` give exactly that; `45` returns `domain.ErrTokenInvalidExpiry`.
  - `Test_Create_Name` (RF2): `"   "` returns `ErrTokenInvalidName`; name is trimmed; 100 runes mixing emoji and Indonesian text accepted; 101 runes rejected.
  - `Test_Create_LimitReached`: the 21st active token returns `ErrTokenLimitReached`; after revoking one, create succeeds; expired tokens do not count.
  - `Test_Verify`: active returns the token; expired returns `ErrTokenExpired`; revoked returns `ErrTokenInvalid` (also when expired); unknown, empty, no-prefix, and `omed_pat_` plus garbage return `ErrTokenInvalid`.
  - `Test_Revoke_OtherUsers_ReturnsNotFound` and `Test_List_ReturnsOnlyOwnWithoutHash`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/service/ -run AccessToken -v`. Expected: FAIL (build error).
- [ ] **Step 3: Implement** the service. Approach: `crypto/rand` 32 bytes, `base64.RawURLEncoding`; display prefix is the first 12 characters of the full token; `Verify` rejects tokens without the prefix before any database call, then orders checks revoked, then expired.
- [ ] **Step 4: Wire** `repository.NewAccessTokenRepository(cfg.EntCli)` and `service.NewAccessTokenService(...)` into `core.New`, exposing `AccessTokenService`.
- [ ] **Step 5: Run** the command of step 2. Expected: PASS.
- [ ] **Step 6: Commit**: `feat(api): ✨ add personal access token service`.

---

### Task 4: REST controller, routes, integration tests, regenerated contract

**Files:**
- Create: `apps/api/internal/core/controller/access_token.go`, `apps/api/internal/core/test/access_token_test.go`
- Modify: `apps/api/internal/core/model/access_token.go`, `apps/api/internal/core/routes.go`; regenerated `apps/api/docs/*` and `packages/openapi/src/api.ts`

**Interfaces:**
- Consumes: `service.AccessTokenService`.
- Produces:
  - `model.AccessTokenResponse{ ID int64 \`json:"id"\`; Name, Prefix, Status string; ExpiresAt, CreatedAt time.Time }` and `model.CreatedAccessTokenResponse` (same fields plus `Token string \`json:"token"\`)
  - `controller.NewAccessTokenController(svc) AccessTokenController` with `Create`, `List`, `Revoke`
  - Routes under the existing `users` group: `POST /users/me/access-tokens` (201), `GET /users/me/access-tokens`, `DELETE /users/me/access-tokens/:id` (same status as `InvitationController.Revoke`)

- [ ] **Step 1: Write failing integration tests** in `core/test/access_token_test.go` (suite on `testutil.ApiTestSuite`):
  - `Test_Lifecycle_CreateListRevoke`: create returns a `token` starting `omed_pat_`; list shows status `active` and no `token` field; revoke, then list shows `revoked`.
  - `Test_List_NoTokens_ReturnsEmptyArray` (RF5): raw body contains `"data":[]`.
  - `Test_Create_InvalidExpiry_Returns422`, `Test_Create_LimitReached_Returns422`.
  - `Test_Revoke_OtherUsersToken_Returns404`.
  - `Test_PAT_RejectedOnRest`: the plaintext as Bearer on `GET /users/me` returns 401.
  - `Test_PAT_CannotManageTokens`: the plaintext as Bearer on `POST /users/me/access-tokens` returns 401.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/test/ -run AccessToken -v`. Expected: FAIL (404 on the new routes).
- [ ] **Step 3: Implement** the controller (Swaggo annotations like `UserController`, user from `base.UserFromContext`, `httpx.Created`/`httpx.OK`, list mapped with a non-nil slice) and register the routes in `routes.go`.
- [ ] **Step 4: Regenerate the contract**: `bunx turbo run oa:gen`, then `bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts`.
- [ ] **Step 5: Run** the command of step 2. Expected: PASS.
- [ ] **Step 6: Commit**: `feat(api): ✨ add access token endpoints`.

---

### Task 5: Boot wiring and `/mcp` mount

**Files:**
- Create: `apps/api/internal/config/mcp.go`, `apps/api/internal/core/test/mcp_test.go`
- Modify: `apps/api/internal/core/middleware/auth.go`, `apps/api/internal/core/middleware/auth_test.go`, `apps/api/internal/config/app.go`, `apps/api/testutil/api.go`

**Interfaces:**
- Consumes: `mcpx.NewRegistrar`, `mcpx.Handler`, `service.AccessTokenService`, `config.HasMCP`.
- Produces:
  - `func newMCPHandler(state State, reg *mcpx.Registrar, tokens service.AccessTokenService) fiber.Handler` in `config/mcp.go`; its verifier maps `domain.ErrTokenExpired` to `mcpx.ErrTokenExpired` and every other failure to `mcpx.ErrTokenInvalid`; `TokenSettingsURL` is `strings.TrimRight(state.Config.WebUrl, "/") + "/core/access-tokens"`
  - `func (s *ApiTestSuite[T]) EnsurePAT(userID int64) string` in `testutil` (creates a token in the test DB the way `EnsureSession` creates a session; the helper Task 7 of the `fin` MCP plan consumes)

- [ ] **Step 1: Write failing tests.**
  - In `auth_test.go`, `Test_Auth_McpPathBypass` (RF4): `/mcp` and `/mcp/` pass through without a session; `/mcpx` and `/mcp-foo` return 401.
  - In `core/test/mcp_test.go` (through the booted app, `POST /mcp` JSON-RPC): `Test_Mcp_ValidPAT_ToolsListIncludesFinTools` (result includes `fin_period_overview`); `Test_Mcp_ExpiredPAT_ReturnsLink` (token created through the repository with a past `ExpiresAt`; body code `TOKEN_EXPIRED` and message ending `/core/access-tokens`); `Test_Mcp_RevokedPAT_GenericUnauthorized`; `Test_Mcp_SessionCookie_Rejected`; `Test_Mcp_SessionTokenAsBearer_Rejected`.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/middleware/ ./internal/core/test/ -run 'Mcp|Auth' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** In `auth.go` bypass the path only when it equals `/mcp` or starts with `/mcp/`. In `config/app.go` build one `mcpx.NewRegistrar(mcpx.Config{Authorizer: az, Resolver: coreMod.MemberService, Logger: state.Logger})` in `New`; in `Boot`, after the module `Boot` loop, call `RegisterMCP` on every module implementing `HasMCP` (joining errors), then `state.FiberApp.All("/mcp", newMCPHandler(...))`. Add `EnsurePAT` to `testutil/api.go`.
- [ ] **Step 4: Run** the command of step 2 and `cd apps/api && go test ./internal/config/ -v`. Expected: PASS.
- [ ] **Step 5: Commit**: `feat(api): ✨ mount mcp endpoint with personal access token auth`.

---

### Task 6: Dashboard token page

**Files:**
- Create: `apps/web/app/modules/core/routes/access-tokens.tsx`, `apps/web/app/modules/core/routes/access-tokens.test.tsx`
- Modify: `apps/web/app/modules/core/route.ts`, `apps/web/app/modules/core/module.ts`, `apps/web/app/modules/core/queries.ts`, `apps/web/app/modules/core/lib.ts`, `apps/web/app/i18n/en.ts`, `apps/web/app/i18n/id.ts`

**Interfaces:**
- Consumes: regenerated `@omed/openapi` types (`model.AccessTokenResponse`, `model.CreatedAccessTokenResponse`) and `api`, `unwrapList`, `unwrapOne`, `unwrapVoid` from `~/lib/api`.
- Produces: route `core/access-tokens`; `accessTokenKeys.all`; `accessTokensQueryOptions()`; `type AccessToken`; `ACCESS_TOKEN_EXPIRY_DAYS = [30, 90, 365] as const`; `DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS = 90`; menu item key `access-tokens` (icon `KeyOutlined`, no `hideInPersonalWorkspace`, since tokens belong to the user).

- [ ] **Step 1: Write failing tests** in `access-tokens.test.tsx`, mirroring the mocking approach of `invitations.test.tsx`: `lists tokens with name, prefix, expiry and a status tag`; `creates a token, shows the plaintext with a copy button and a one-time warning, and no longer shows it after closing`; `sends the chosen expiry and defaults to 90`; `revokes after confirmation and refreshes the list`; `shows an empty state when there are no tokens`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/modules/core/routes/access-tokens.test.tsx`. Expected: FAIL (module not found).
- [ ] **Step 3: Implement** the route (table, create modal with name and expiry select, reveal view, `Popconfirm` revoke), the query options, the lib constants, the route entry, the menu item, and the `core.menu.accessTokens` and `core.accessTokens.*` keys in both `en.ts` and `id.ts`. Style with design tokens only.
- [ ] **Step 4: Run** the command of step 2, then `bun run typecheck` and `bun run lint`. Expected: PASS.
- [ ] **Step 5: Commit**: `feat(web): ✨ add access tokens page`.

---

### Task 7: End-to-end scenario

**Files:**
- Create: `e2e/features/core/access-tokens.feature`, `e2e/pages/core/access-tokens.page.ts`, `e2e/steps/core/access-tokens.steps.ts`
- Modify: `e2e/pages/index.ts`

**Interfaces:**
- Consumes: `BasePage`, `createPages()` grouping by module, the `@authenticated` hook.
- Produces: `AccessTokensPage` exposed as `this.pages.core.accessTokens`, with locators and `expect*` checks inside the page object only.

- [ ] **Step 1: Write the feature** (tag `@authenticated`): `Scenario: A user creates an access token and sees it only once` (create with name "Claude Desktop", expiry 90; the token text starting `omed_pat_` is visible; after closing, it is gone and the list shows the name and status active) and `Scenario: A user revokes an access token` (status becomes revoked).
- [ ] **Step 2: Run** `bun run e2e`. Expected: FAIL (undefined steps).
- [ ] **Step 3: Implement** the page object (path `/core/access-tokens`) and the steps, registering the page in `createPages()`.
- [ ] **Step 4: Run** `bun run e2e`. Expected: all scenarios PASS.
- [ ] **Step 5: Commit**: `test(e2e): ✅ cover access token creation and revocation`.

---

### Task 8: Documentation and final verification

**Files:**
- Modify: `README.md`, `docs/superpowers/specs/2026-11-01-mcp-foundation-design.md`, `docs/superpowers/plans/2026-11-01-fin-mcp-integration.md`

- [ ] **Step 1: README.** Add the `/mcp` endpoint (PAT bearer, Streamable HTTP, stateless) and the `/core/access-tokens` page to the architecture section, mention the tools registered by modules, and state that no new environment variable is needed.
- [ ] **Step 2: Spec corrections.** In spec section 5.4 change `expires_in_days` to `expiresInDays` (camelCase, as every existing DTO) and note that an omitted value means 90; set the spec status to `Approved`.
- [ ] **Step 3: Unblock the `fin` plan.** In the `fin` MCP plan, Task 7, replace "names fixed in its plan" with `EnsurePAT(userID)` from `testutil` and the `/mcp` endpoint, and remove the "Blocked" line.
- [ ] **Step 4: Verify everything.** Run `bun run lint`, `bun run typecheck`, `cd apps/api && go test ./...`, and `bun run e2e`. Expected: all pass. Run `grep -n "omed_pat_" apps/api/tmp/test.log` and confirm it prints nothing (no token was logged).
- [ ] **Step 5: Commit**: `docs(docs): 📝 document mcp endpoint and access tokens`.
