# OAuth2 Provider Driver Pattern Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Google and GitHub OAuth2 logic out of `oauthService` into drivers under `internal/platform/oauthx`, resolved through a registry.

**Architecture:** `oauthx` defines a `Driver` interface, a `Profile` type, a shared `Base` helper, and a `Registry`. `google` and `github` sub-packages implement drivers by embedding `Base`. `core` builds the registry from configured credentials and `oauthService` delegates provider-specific work to the resolved driver while keeping state, linking, upsert, and session logic.

**Tech Stack:** Go 1.27.1, `golang.org/x/oauth2` v0.37.0, Fiber v3, Ent, testify.

**Spec:** `docs/superpowers/specs/2026-11-01-oauth-driver-pattern-design.md` (issue [#34](https://github.com/kilip/omed/issues/34)). Executors read both documents.

## Global Constraints

- Go module is `github.com/kilip/omed/api`. Add no new dependencies.
- `internal/platform/*` packages must not import `internal/core/...` (spec D2). Driver IDs are therefore local constants (`google.ID`, `github.ID`), pinned to `domain.ProviderGoogle` and `domain.ProviderGithub` by a test in `core`.
- When `HTTPClient` is nil, drivers use `&http.Client{Timeout: 10 * time.Second}`.
- `GetAuthURL` uses `oauth2.AccessTypeOffline`. `ExchangeCode` injects the driver HTTP client through the `oauth2.HTTPClient` context key.
- `GetJSON` sends `Authorization: Bearer <token>` and `Accept: application/json`. A non-2xx response returns the error text `provider returned status <code>: <body>`.
- The redirect URI stays `{apiURL}/auth/callback/{provider}`, where `apiURL` is trimmed of a trailing `/` and defaults to `http://localhost:8001`.
- No new environment variables. No changes to OpenAPI (`apps/api/docs`, `packages/openapi`), migrations, Helm, `AGENTS.md`, or `docs/configuration.md`.
- Commits use `type(scope): <emoji> subject` with scopes `api` or `docs`; emojis `feat` ✨, `refactor` ♻️, `docs` 📝; lowercase imperative subject, at most 72 characters.
- Unit tests sit next to the file under test. Format with `gofmt`. Documentation is English Markdown.
- All commands run from `apps/api` unless stated otherwise.

## Review Focus

Failure modes the spec implies but its main flow does not exercise. Each has a pinning test in the owning task.

1. GitHub `/user/emails` returns a non-200 status or malformed JSON: the profile email is kept, `EmailVerified` stays false, and no error is returned. (Task 4)
2. An empty or differently cased provider string (`""`, `"GOOGLE"`) yields `domain.ErrInvalidProvider`, never a panic. (Task 5)
3. Token endpoint rejection maps to `domain.ErrOAuthExchangeFailed`; a user info 5xx maps to `domain.ErrFetchUserInfoFailed`. (Task 5)
4. A service built with a nil or empty registry (no credentials configured) rejects every provider and does not panic. (Task 5)
5. A caller mutating the slice returned by `Scopes()` does not change the driver's scopes. (Task 2)

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/platform/oauthx/driver.go` | `Driver`, `Profile`, `ErrUnsupportedProvider` |
| `internal/platform/oauthx/registry.go` | `Registry` |
| `internal/platform/oauthx/base.go` | `Base`, `BaseConfig`, `NewBase` |
| `internal/platform/oauthx/google/google.go` | Google driver |
| `internal/platform/oauthx/github/github.go` | GitHub driver |
| `internal/core/oauth_drivers.go` | `newOAuthRegistry(cfg Config) *oauthx.Registry` |
| `internal/core/module.go` | Pass `Drivers` to `NewOAuthService` |
| `internal/core/service/oauth.go` | Orchestration only |
| `internal/core/service/oauth_test.go` | Service tests on driver-based mocks |
| `docs/architecture.md` | `oauthx` documentation |

Each source file has a `_test.go` beside it.

---

### Task 1: oauthx contract and registry

**Files:**
- Create: `internal/platform/oauthx/driver.go`, `internal/platform/oauthx/registry.go`
- Test: `internal/platform/oauthx/registry_test.go` (package `oauthx_test`)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Profile struct { ID, Email, Name, Avatar string; EmailVerified bool; Locale string }`
  - `type Driver interface { ID() string; Scopes() []string; GetAuthURL(state, redirectURL string) string; ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error); FetchUserProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) }`
  - `var ErrUnsupportedProvider = errors.New("oauthx: unsupported provider")`
  - `func NewRegistry(drivers ...Driver) *Registry`, `(*Registry).Register(d Driver)`, `(*Registry).Get(id string) (Driver, error)`, `(*Registry).IDs() []string`

- [ ] **Step 1: Write the failing tests** using a local `fakeDriver{id string}` that implements `Driver` with no-op methods:
  - `TestRegistry_GetReturnsRegisteredDriver`: `NewRegistry(fakeDriver{"a"}).Get("a")` returns that driver and no error.
  - `TestRegistry_GetUnknownReturnsErrUnsupportedProvider`: `Get("missing")` and `Get("")` both satisfy `errors.Is(err, oauthx.ErrUnsupportedProvider)`.
  - `TestRegistry_RegisterReplacesSameID`: registering a second driver with ID `a` makes `Get("a")` return the second one.
  - `TestRegistry_IDsSorted`: drivers registered as `google`, `github` yield `[]string{"github", "google"}`; an empty registry yields an empty slice.
  - `TestRegistry_ConcurrentRegisterAndGet`: 50 goroutines mixing `Register` and `Get` finish without a data race.
- [ ] **Step 2: Run to verify failure:** `go test ./internal/platform/oauthx/ -race`. Expected: FAIL (build error, undefined `oauthx.NewRegistry`). Drop `-race` only if the toolchain has no cgo.
- [ ] **Step 3: Implement** `driver.go` and `registry.go` per the Produces block. `Registry` holds `map[string]Driver` behind a `sync.RWMutex`; `Get` returns `ErrUnsupportedProvider` when absent.
- [ ] **Step 4: Run to verify pass:** same command. Expected: PASS.
- [ ] **Step 5: Commit**

```bash
git add internal/platform/oauthx
git commit -m "feat(api): ✨ add oauthx driver interface and registry"
```

---

### Task 2: oauthx Base helper

**Files:**
- Create: `internal/platform/oauthx/base.go`
- Test: `internal/platform/oauthx/base_test.go` (package `oauthx`, internal, so it can read the default client)

**Interfaces:**
- Consumes: package `oauthx` from Task 1.
- Produces:
  - `type BaseConfig struct { ClientID, ClientSecret string; Endpoint oauth2.Endpoint; Scopes []string; HTTPClient *http.Client }`
  - `func NewBase(id string, cfg BaseConfig) *Base`
  - `(*Base).ID() string`, `(*Base).Scopes() []string` (returns a copy)
  - `(*Base).GetAuthURL(state, redirectURL string) string`
  - `(*Base).ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error)`
  - `(*Base).GetJSON(ctx context.Context, url, accessToken string, out any) error`

- [ ] **Step 1: Write the failing tests** using `httptest.Server`:
  - `TestBase_GetAuthURL_IncludesParams`: URL starts with the endpoint `AuthURL` and contains `client_id=cid`, `state=st`, `access_type=offline`, and `redirect_uri=` equal to the URL-encoded redirect passed in.
  - `TestBase_ExchangeCode_UsesInjectedHTTPClient`: a custom `http.Client` whose transport counts requests; after `ExchangeCode` the count is 1, the token has `AccessToken` from the server, and the token request form `redirect_uri` equals the redirect passed in.
  - `TestBase_ExchangeCode_ProviderRejects`: token endpoint returns 400 `{"error":"invalid_grant"}`; the error is non-nil.
  - `TestBase_GetJSON_Success`: server asserts headers `Authorization: Bearer tok` and `Accept: application/json`; the body decodes into `out`.
  - `TestBase_GetJSON_NonSuccessStatus`: server returns 502 with body `boom`; error text equals `provider returned status 502: boom`.
  - `TestBase_GetJSON_InvalidJSON`: 200 with body `not-json`; error is non-nil.
  - `TestBase_Scopes_ReturnsCopy` (Review Focus 5): set `got[0] = "mutated"` on the returned slice; a second `Scopes()` call still returns the original scopes.
  - `TestNewBase_DefaultsHTTPClientTimeout`: with nil `HTTPClient`, the stored client has `Timeout == 10*time.Second`.
- [ ] **Step 2: Run to verify failure:** `go test ./internal/platform/oauthx/ -race`. Expected: FAIL (undefined `NewBase`).
- [ ] **Step 3: Implement** `Base` with signatures above. `GetAuthURL` and `ExchangeCode` copy an `oauth2.Config` per call, set `RedirectURL`, then call `AuthCodeURL(state, oauth2.AccessTypeOffline)` or `Exchange` on a context carrying the client. `GetJSON` reads the body once; non-2xx returns the spec error text.
- [ ] **Step 4: Run to verify pass:** same command. Expected: PASS.
- [ ] **Step 5: Commit**

```bash
git add internal/platform/oauthx
git commit -m "feat(api): ✨ add oauthx base helper for oauth2 drivers"
```

---

### Task 3: Google driver

**Files:**
- Create: `internal/platform/oauthx/google/google.go`
- Test: `internal/platform/oauthx/google/google_test.go`

**Interfaces:**
- Consumes: `oauthx.NewBase`, `oauthx.BaseConfig`, `*oauthx.Base`, `oauthx.Profile`.
- Produces:
  - `const ID = "google"`
  - `type Config struct { ClientID, ClientSecret string; HTTPClient *http.Client }`
  - `type Option func(*settings)` with `WithEndpoint(oauth2.Endpoint) Option` and `WithUserInfoURL(string) Option`
  - `type Driver struct { *oauthx.Base; ... }`, `func New(cfg Config, opts ...Option) *Driver`
  - `(*Driver).FetchUserProfile(ctx context.Context, token *oauth2.Token) (*oauthx.Profile, error)`
  - Compile-time check `var _ oauthx.Driver = (*Driver)(nil)`.

- [ ] **Step 1: Write the failing tests** (user info served by `httptest.Server` through `WithUserInfoURL`):
  - `TestDriver_Defaults`: `ID() == "google"`; `Scopes()` equals `openid`, `https://www.googleapis.com/auth/userinfo.email`, `https://www.googleapis.com/auth/userinfo.profile`; `GetAuthURL("s", "http://x/cb")` starts with `golang.org/x/oauth2/google.Endpoint.AuthURL` (imported as `oauth2google`).
  - `TestFetchUserProfile_MapsFields`: payload `id`, `email`, `name`, `picture`, `locale: "id"`, `verified_email: true` maps to `ID`, `Email`, `Name`, `Avatar`, `Locale`, `EmailVerified == true`.
  - `TestFetchUserProfile_EmailVerifiedAlias`: payload with `email_verified: true` and no `verified_email` gives `EmailVerified == true`.
  - `TestFetchUserProfile_Unverified`: `verified_email: false` gives `EmailVerified == false`.
  - `TestFetchUserProfile_SendsBearerToken`: server asserts `Authorization: Bearer tok`.
  - `TestFetchUserProfile_NonSuccessStatus`: 500 response gives an error containing `provider returned status 500`.
- [ ] **Step 2: Run to verify failure:** `go test ./internal/platform/oauthx/google/`. Expected: FAIL (undefined `New`).
- [ ] **Step 3: Implement** `New` and `FetchUserProfile`. Options mutate an unexported `settings` struct (endpoint, user info URL) that `New` applies before calling `oauthx.NewBase(ID, ...)`. Defaults: endpoint `oauth2google.Endpoint`, user info URL `https://www.googleapis.com/oauth2/v2/userinfo`, scopes as in the test. `EmailVerified = verified_email || email_verified`.
- [ ] **Step 4: Run to verify pass:** same command. Expected: PASS.
- [ ] **Step 5: Commit**

```bash
git add internal/platform/oauthx/google
git commit -m "feat(api): ✨ add google oauthx driver"
```

---

### Task 4: GitHub driver

**Files:**
- Create: `internal/platform/oauthx/github/github.go`
- Test: `internal/platform/oauthx/github/github_test.go`

**Interfaces:**
- Consumes: same `oauthx` symbols as Task 3.
- Produces: `const ID = "github"`; `Config` identical to Task 3; `Option`, `WithEndpoint`, `WithUserInfoURL`, `WithEmailsURL(string) Option`; `Driver`, `New(cfg Config, opts ...Option) *Driver`; `(*Driver).FetchUserProfile(ctx, token) (*oauthx.Profile, error)`; `var _ oauthx.Driver = (*Driver)(nil)`.

- [ ] **Step 1: Write the failing tests** (user and emails endpoints served by one `httptest.Server`):
  - `TestDriver_Defaults`: `ID() == "github"`; `Scopes()` equals `read:user`, `user:email`; auth URL starts with `golang.org/x/oauth2/github.Endpoint.AuthURL`.
  - `TestFetchUserProfile_PrefersPrimaryVerifiedEmail`: emails `[{n@x.com, primary false, verified true}, {p@x.com, primary true, verified true}]` give `Email == "p@x.com"`, `EmailVerified == true`.
  - `TestFetchUserProfile_FallsBackToVerifiedNonPrimary`: emails `[{u@x.com, primary true, verified false}, {v@x.com, primary false, verified true}]` give `Email == "v@x.com"`, verified true, with an empty profile `email`.
  - `TestFetchUserProfile_NoVerifiedEmail`: all emails unverified, profile `email` is `e@x.com`: `Email == "e@x.com"`, `EmailVerified == false`.
  - `TestFetchUserProfile_EmailsEndpointFailureIgnored` (Review Focus 1): subtests for emails status 500 and for emails body `not-json`; each returns no error, keeps the profile email, `EmailVerified == false`.
  - `TestFetchUserProfile_NameFallsBackToLogin`: empty `name`, `login: "octo"` gives `Name == "octo"`.
  - `TestFetchUserProfile_IDFormattedBase10`: numeric `id: 2002` gives `ID == "2002"`.
  - `TestFetchUserProfile_VerifiedFieldPrecedence`: with the emails endpoint returning 404, `email_verified: true` gives true; only `verified: true` gives true; `email_verified: false` with `verified: true` gives false (the first field present wins).
  - `TestFetchUserProfile_UserInfoNonSuccessStatus`: user info 500 gives an error containing `provider returned status 500`.
- [ ] **Step 2: Run to verify failure:** `go test ./internal/platform/oauthx/github/`. Expected: FAIL (undefined `New`).
- [ ] **Step 3: Implement** `New` and `FetchUserProfile` mirroring the current `fetchUserInfo` and `fetchGithubEmail` logic in `core/service/oauth.go`: defaults endpoint `github.Endpoint`, user info URL `https://api.github.com/user`, emails URL `https://api.github.com/user/emails`. The emails lookup goes through `GetJSON` and every error from it is ignored. Verified selection: primary and verified first, then any verified; a match overrides `Email` and sets `EmailVerified = true`.
- [ ] **Step 4: Run to verify pass:** same command. Expected: PASS.
- [ ] **Step 5: Commit**

```bash
git add internal/platform/oauthx/github
git commit -m "feat(api): ✨ add github oauthx driver"
```

---

### Task 5: Core wiring and service migration

Tests migrate in the same commit as the service because they compile against the new `OAuthServiceParams`, and every commit must build.

**Files:**
- Create: `internal/core/oauth_drivers.go`, `internal/core/oauth_drivers_test.go` (package `core`)
- Modify: `internal/core/module.go` (the `NewOAuthService` call), `internal/core/service/oauth.go`, `internal/core/service/oauth_test.go`

**Interfaces:**
- Consumes: Tasks 1 to 4 (`oauthx.Registry`, `google.New`, `github.New`, `google.ID`, `github.ID`, driver options).
- Produces:
  - `func newOAuthRegistry(cfg Config) *oauthx.Registry` in package `core`: registers Google only when `GoogleID` and `GoogleSecret` are both non-empty, GitHub likewise with `GithubID` and `GithubSecret`.
  - `service.OAuthServiceParams{ Accounts AccountRepository; Users repository.UserRepository; Sessions SessionService; AuthSecret string; ApiURL string; TrustedOrigins []string; Drivers *oauthx.Registry; Logger *slog.Logger }`. `NewOAuthService(p OAuthServiceParams) OAuthService` keeps its signature; a nil `Drivers` becomes an empty registry.
  - Removed: `OAuthProviderConfig`, `RegisterProvider`, and the fields `GoogleID`, `GoogleSecret`, `GithubID`, `GithubSecret`, `HTTPClient`.

- [ ] **Step 1: Write wiring tests** in `internal/core/oauth_drivers_test.go`:
  - `TestNewOAuthRegistry_RegistersConfiguredProviders`: both credential pairs set gives `IDs() == []string{"github", "google"}`.
  - `TestNewOAuthRegistry_SkipsIncompleteCredentials`: ID without secret, secret without ID, and no credentials each leave the provider absent (`IDs()` empty when nothing is configured).
  - `TestOAuthDriverIDsMatchDomainConstants`: `google.ID == domain.ProviderGoogle` and `github.ID == domain.ProviderGithub`.
- [ ] **Step 2: Migrate `oauth_test.go`.**
  - The suite keeps a `registry *oauthx.Registry` passed as `Drivers`. Add helpers `useGoogle(userInfoPath string)` and `useGithub(userInfoPath, emailsPath string)` that register drivers built with `HTTPClient: s.mockServer.Client()`, `WithEndpoint` (`AuthURL: mock/auth`, `TokenURL: mock/token`), and `WithUserInfoURL`/`WithEmailsURL` pointing at the mock server. Replace every `RegisterProvider` block, including the `failingAccountRepo` test, with these helpers.
  - Remove `Test_FetchGithubEmail_PicksVerifiedNonPrimaryWhenPrimaryUnverified` (covered by Task 4). Keep the linking, nonce, callback URL, locale, and upsert-error tests.
  - Extend `Test_GetAuthURL_Success` to assert the URL contains `redirect_uri=http%3A%2F%2Flocalhost%3A8001%2Fauth%2Fcallback%2Fgoogle`.
  - Add mock server routes `/token-error` (400 `{"error":"invalid_grant"}`) and `/userinfo/error` (500).
  - Add tests: `Test_GetAuthURL_EmptyOrMiscasedProvider` (`""` and `"GOOGLE"` give `domain.ErrInvalidProvider`); `Test_NewOAuthService_NilDrivers_RejectsAllProviders` (`GetAuthURL` for `google` returns `ErrInvalidProvider`, and `HandleCallback` with a correctly signed state also returns `ErrInvalidProvider`); `Test_HandleCallback_TokenEndpointRejects_ReturnsExchangeFailed` (`ErrOAuthExchangeFailed`); `Test_HandleCallback_UserInfo5xx_ReturnsFetchUserInfoFailed` (`ErrFetchUserInfoFailed`, message contains `status 500`); `Test_HandleCallback_ScopeFallsBackToDriverScopes` (the token response has no `scope`, so the stored account scope equals the Google driver scopes joined by a single space).
  - Add a `signedState(claims model.OAuthStateClaims) string` helper for the new tests.
- [ ] **Step 3: Run to verify failure:** `go test ./internal/core/... 2>&1 | head -30`. Expected: FAIL (build errors for unknown `Drivers` field and `newOAuthRegistry`).
- [ ] **Step 4: Implement the service migration** in `oauth.go` per spec section 7: private `driver(provider string) (oauthx.Driver, error)` mapping `oauthx.ErrUnsupportedProvider` to `domain.ErrInvalidProvider`; private `redirectURL(provider string) string`; a mapper from `*oauthx.Profile` to `model.OAuthUserInfo`; `GetAuthURL` and `HandleCallback` follow the spec flow with the existing check order (state, nonce, callback URL, then driver); exchange and profile errors keep the `%w: %v` wrapping with `ErrOAuthExchangeFailed` and `ErrFetchUserInfoFailed`; scope fallback uses `strings.Join(driver.Scopes(), " ")`. Delete `getOAuthConfig`, `fetchUserInfo`, `fetchGithubEmail`, `RegisterProvider`, `OAuthProviderConfig`, the `httpClient` field, and the imports they leave unused.
- [ ] **Step 5: Implement the wiring** in `oauth_drivers.go` and replace the provider credential fields in the `NewOAuthService` call in `module.go` with `Drivers: newOAuthRegistry(cfg)`.
- [ ] **Step 6: Run to verify pass:** `gofmt -l . && go build ./... && go vet ./... && go test ./internal/core/... ./internal/platform/...`. Expected: `gofmt` prints nothing, everything else passes (including `internal/core/controller` unchanged).
- [ ] **Step 7: Commit**

```bash
git add internal/core
git commit -m "refactor(api): ♻️ delegate oauth provider logic to oauthx drivers"
```

---

### Task 6: Documentation and final verification

**Files:**
- Modify: `docs/architecture.md` (repository root `docs/`)

**Interfaces:**
- Consumes: the finished implementation.
- Produces: documentation only.

- [ ] **Step 1: Add a `## OAuth Provider Drivers` section** to `docs/architecture.md`, directly before `## Model Context Protocol (MCP) Server`. Content: one paragraph on `internal/platform/oauthx` (interface, `Base`, registry, `google` and `github` drivers, auto-registration from `OMED_GOOGLE_*` and `OMED_GITHUB_*`), and a numbered list for adding a provider: create `oauthx/<name>` implementing `Driver` by embedding `Base`; register it in `core/oauth_drivers.go`; add the credentials to `config.go` and follow `AGENTS.md` section 6 for the Helm chart when new `OMED_*` variables are introduced.
- [ ] **Step 2: Run the full verification** from `apps/api`: `gofmt -l . && go build ./... && go vet ./... && go test ./...`. Expected: no `gofmt` output, all packages pass.
- [ ] **Step 3: Confirm the API contract is untouched** from the repository root: `git diff --stat origin/main -- apps/api/docs packages/openapi helm apps/api/migrations AGENTS.md`. Expected: empty output.
- [ ] **Step 4: Commit and push**

```bash
git add docs/architecture.md
git commit -m "docs(docs): 📝 document oauthx provider drivers"
git push
```
