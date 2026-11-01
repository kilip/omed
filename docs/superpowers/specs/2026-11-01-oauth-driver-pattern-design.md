# OAuth2 Provider Driver Pattern — Design Specification

- **Issue:** [kilip/omed#34](https://github.com/kilip/omed/issues/34)
- **Date:** 2026-11-01
- **Status:** Approved for implementation planning
- **Scope:** `apps/api` (Go backend only)

## 1. Context

`apps/api/internal/core/service/oauth.go` currently contains provider-specific logic inside `oauthService`:

- Google and GitHub are initialized inline in `NewOAuthService` from `GoogleID/GoogleSecret/GithubID/GithubSecret`.
- `fetchUserInfo` switches on the provider ID to parse each provider's profile payload.
- `fetchGithubEmail` performs the GitHub-specific verified-email lookup.
- `OAuthProviderConfig` and `RegisterProvider` exist so tests can override endpoints.

Adding a provider (GitLab, Apple, Discord, Keycloak/OIDC) requires editing the core service. The goal is to move all provider-specific behavior behind a driver interface so that a new provider is a self-contained package.

## 2. Goals and Non-Goals

### Goals

1. Define a clean `Driver` interface for OAuth2 identity providers.
2. Implement Google and GitHub drivers with behavior identical to the current implementation.
3. Provide a registry that resolves a driver by provider ID and registers drivers from configured credentials.
4. Reduce `OAuthService` to orchestration: state/nonce verification, callback URL validation, account linking, user upsert, and session creation.
5. Keep `POST /auth/sign-in/social` and `GET /auth/callback/:provider` backward compatible.

### Non-Goals

- Adding new providers (GitLab, Apple, Discord, generic OIDC).
- OIDC discovery, PKCE, ID token verification, or sending the nonce to the provider.
- Token refresh.
- Any change to database schema, migrations, OpenAPI contract, environment variables, or the Helm chart.

## 3. Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Drivers live in `apps/api/internal/platform/oauthx`, not `internal/core/driver/oauth`. | AGENTS.md section 3 defines no `driver/` layer inside modules. External adapters already live in `internal/platform/` (`s3x`, `mailx`, `llmx`). |
| D2 | `oauthx` defines its own `Profile` type. | No package under `internal/platform` imports a module. The core service maps `Profile` to `model.OAuthUserInfo`. |
| D3 | A shared `Base` helper provides `GetAuthURL` and `ExchangeCode`. | Both providers differ only in endpoint, scopes, and profile parsing. A new provider then requires one file. |
| D4 | The `nonce` parameter is removed from `Driver.GetAuthURL`. | The nonce is held in the signed state claims and verified by the service. It is never sent to the provider today, so removal changes no behavior. |
| D5 | `redirectURL` is passed per call to `GetAuthURL` and `ExchangeCode`. | The service already normalizes `apiURL` and derives `{apiURL}/auth/callback/{provider}`. Drivers stay free of API URL handling. |
| D6 | The registry returns `oauthx.ErrUnsupportedProvider`; the service maps it to `domain.ErrInvalidProvider`. | Keeps `platform` independent of `core/domain`. |
| D7 | HTTP client ownership moves from the service to the drivers. | Drivers perform all provider HTTP calls. The default client keeps the current 10 second timeout. |

## 4. Package Layout

```
apps/api/internal/platform/oauthx/
├── driver.go          # Driver interface, Profile, ErrUnsupportedProvider
├── base.go            # Base helper (oauth2.Config wrapper, bearer GET helper)
├── registry.go        # Registry
├── registry_test.go
├── base_test.go
├── google/
│   ├── google.go
│   └── google_test.go
└── github/
    ├── github.go
    └── github_test.go

apps/api/internal/core/
├── oauth_drivers.go   # builds the registry from core.Config credentials
└── service/oauth.go   # reduced to orchestration
```

## 5. Interfaces

### 5.1 Driver and Profile

```go
package oauthx

// ErrUnsupportedProvider is returned when no driver is registered for a provider ID.
var ErrUnsupportedProvider = errors.New("oauthx: unsupported provider")

// Profile is the normalized user profile returned by a provider.
type Profile struct {
	ID            string
	Email         string
	Name          string
	Avatar        string
	EmailVerified bool
	Locale        string
}

// Driver encapsulates all provider-specific OAuth2 behavior.
type Driver interface {
	// ID returns the provider identifier, e.g. "google".
	ID() string
	// Scopes returns the scopes requested from the provider.
	Scopes() []string
	// GetAuthURL builds the provider authorization URL.
	GetAuthURL(state, redirectURL string) string
	// ExchangeCode exchanges an authorization code for a token.
	ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error)
	// FetchUserProfile retrieves and normalizes the user profile.
	FetchUserProfile(ctx context.Context, token *oauth2.Token) (*Profile, error)
}
```

### 5.2 Base

`Base` is embedded by concrete drivers.

```go
type BaseConfig struct {
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
	Scopes       []string
	HTTPClient   *http.Client // nil defaults to a client with a 10 second timeout
}

func NewBase(id string, cfg BaseConfig) *Base

func (b *Base) ID() string
func (b *Base) Scopes() []string // returns a copy
func (b *Base) GetAuthURL(state, redirectURL string) string
func (b *Base) ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error)
func (b *Base) GetJSON(ctx context.Context, url, accessToken string, out any) error
```

Behavior contract:

- `GetAuthURL` builds an `oauth2.Config` with the given `redirectURL` and calls `AuthCodeURL(state, oauth2.AccessTypeOffline)`, matching current behavior.
- `ExchangeCode` injects the driver's HTTP client into the context under `oauth2.HTTPClient` before calling `Exchange`, matching current behavior.
- `GetJSON` performs a GET with `Authorization: Bearer <token>` and `Accept: application/json`. A non-2xx response returns an error formatted as `provider returned status <code>: <body>`, matching the current message. A 2xx body is decoded into `out`.

### 5.3 Registry

```go
func NewRegistry(drivers ...Driver) *Registry
func (r *Registry) Register(d Driver)              // replaces an existing driver with the same ID
func (r *Registry) Get(id string) (Driver, error)  // ErrUnsupportedProvider when absent
func (r *Registry) IDs() []string                  // sorted
```

The registry is safe for concurrent use (`sync.RWMutex`). Registration normally happens once at boot; tests may re-register drivers.

## 6. Drivers

Concrete drivers expose a constructor with a config struct and functional options. Options exist so tests can point a driver at a mock server.

### 6.1 Google (`oauthx/google`)

```go
type Config struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

func New(cfg Config, opts ...Option) *Driver
// Options: WithEndpoint(oauth2.Endpoint), WithUserInfoURL(string)
```

| Item | Value |
|------|-------|
| ID | `google` (`domain.ProviderGoogle`) |
| Endpoint | `golang.org/x/oauth2/google.Endpoint` |
| Scopes | `openid`, `https://www.googleapis.com/auth/userinfo.email`, `https://www.googleapis.com/auth/userinfo.profile` |
| UserInfo URL | `https://www.googleapis.com/oauth2/v2/userinfo` |

Profile mapping: `id`, `email`, `name`, `picture` (to `Avatar`), `locale`; `EmailVerified = verified_email || email_verified`.

The package imports `golang.org/x/oauth2/google` under the alias `oauth2google` to avoid a name clash with the package name.

### 6.2 GitHub (`oauthx/github`)

```go
func New(cfg Config, opts ...Option) *Driver
// Options: WithEndpoint(oauth2.Endpoint), WithUserInfoURL(string), WithEmailsURL(string)
```

| Item | Value |
|------|-------|
| ID | `github` (`domain.ProviderGithub`) |
| Endpoint | `golang.org/x/oauth2/github.Endpoint` |
| Scopes | `read:user`, `user:email` |
| UserInfo URL | `https://api.github.com/user` |
| Emails URL | `https://api.github.com/user/emails` |

Profile mapping preserves the current logic exactly:

1. `ID` is the numeric `id` formatted in base 10.
2. `Name` falls back to `login` when `name` is empty.
3. `EmailVerified` starts from the optional `email_verified` field, then the optional `verified` field, otherwise `false`.
4. The driver queries the emails URL. A failure of that request (transport error, non-200 status, or decode error) is ignored and the profile email is kept. On success, the primary verified email is preferred, then any verified email. A match overrides `Email` and sets `EmailVerified = true`.

## 7. Service Changes

### 7.1 Parameters

```go
type OAuthServiceParams struct {
	Accounts       AccountRepository
	Users          repository.UserRepository
	Sessions       SessionService
	AuthSecret     string
	ApiURL         string
	TrustedOrigins []string
	Drivers        *oauthx.Registry // nil results in an empty registry
	Logger         *slog.Logger
}
```

Removed: `GoogleID`, `GoogleSecret`, `GithubID`, `GithubSecret`, `HTTPClient`, `OAuthProviderConfig`, and `RegisterProvider`.

### 7.2 Flow

`GetAuthURL`:

1. Resolve the driver; map `oauthx.ErrUnsupportedProvider` to `domain.ErrInvalidProvider`.
2. Validate `CallbackURL` against trusted origins (unchanged).
3. Generate signed state and nonce (unchanged).
4. Return `driver.GetAuthURL(state, redirectURL(provider))`.

`HandleCallback` (order of checks is preserved):

1. Verify state, nonce, and callback URL (unchanged).
2. Resolve the driver (`domain.ErrInvalidProvider` when unsupported).
3. `driver.ExchangeCode`; wrap failures with `domain.ErrOAuthExchangeFailed`.
4. `driver.FetchUserProfile`; wrap failures with `domain.ErrFetchUserInfoFailed`.
5. Map `oauthx.Profile` to `model.OAuthUserInfo`; apply the locale claim override.
6. `resolveUser`, account upsert, and session creation (unchanged).
7. When the token carries no `scope` extra, the stored scope falls back to `strings.Join(driver.Scopes(), " ")`.

`redirectURL(provider)` remains `{normalized apiURL}/auth/callback/{provider}`, so redirect URIs registered with providers stay valid.

### 7.3 Retained in the service

State signing and verification, trusted-origin checks, `resolveUser`, `NormalizeLocaleAndCurrency`, and the `AccountRepository` interface.

## 8. Wiring

`apps/api/internal/core/oauth_drivers.go` exposes a helper that builds the registry from `core.Config`:

- When `GoogleID` and `GoogleSecret` are both non-empty, register `google.New(...)`.
- When `GithubID` and `GithubSecret` are both non-empty, register `github.New(...)`.

`core.New` passes the result as `OAuthServiceParams.Drivers`. `core.Config`, `config/app.go`, environment variables, and documentation of those variables are unchanged.

## 9. Backward Compatibility

| Surface | Change |
|---------|--------|
| `POST /auth/sign-in/social` request and response | None |
| `GET /auth/callback/:provider` | None |
| OAuth state format and signing | None |
| Redirect URI sent to providers | None |
| Error codes (`OAUTH_INVALID_PROVIDER`, `OAUTH_INVALID_STATE`, `OAUTH_EXCHANGE_FAILED`, `OAUTH_USERINFO_FAILED`) | None |
| Stored account fields | None |
| Environment variables, Helm chart, OpenAPI, migrations | None |

## 10. Testing Strategy

Unit tests live next to the file under test, per AGENTS.md.

| Test file | Coverage |
|-----------|----------|
| `oauthx/registry_test.go` | Get hit, get miss returns `ErrUnsupportedProvider`, register overrides by ID, `IDs` sorted, concurrent access. |
| `oauthx/base_test.go` | Auth URL contains client ID, state, redirect URL, and offline access; `ExchangeCode` uses the injected HTTP client; `GetJSON` success, non-2xx error message, and decode error. |
| `oauthx/google/google_test.go` | Verified and unverified profiles, `email_verified` alias, locale and avatar mapping, defaults for endpoint, scopes, and user info URL. |
| `oauthx/github/github_test.go` | Verified primary email, unverified primary email, fallback to a verified non-primary email, empty profile email, name fallback to login, emails endpoint failure ignored, `verified` field handling. |
| `core/service/oauth_test.go` | Existing state, nonce, callback URL, linking, locale, and upsert-error tests retained. The suite builds a registry whose drivers are overridden to the mock server through driver options. Provider-specific parsing cases move to the driver tests. Add cases for an empty registry and for scope fallback to `driver.Scopes()`. |
| `core/controller/auth_test.go` and `core/test/` | Run unchanged to confirm endpoint compatibility. Adjust only construction code if it references removed parameters. |

Verification commands:

```bash
cd apps/api && gofmt -l . && go build ./... && go vet ./... && go test ./...
```

## 11. Documentation

- `docs/architecture.md`: add a short subsection describing `internal/platform/oauthx` and the steps to add a provider (implement `Driver`, embed `Base`, register in `core/oauth_drivers.go`).
- `AGENTS.md`: no change.
- `docs/configuration.md`: no change.

## 12. Commit and Branch Conventions

- Branch: `refactor/oauth-driver-pattern`, created from `main`.
- Commit scopes: `api` for code, `docs` for documentation. Messages follow the emoji convention in AGENTS.md section 2, for example `refactor(api): ♻️ extract oauth providers into oauthx drivers`.
- One logical change per commit: platform package, service migration, test migration, documentation.

## 13. Deviations from Issue #34

| Issue text | This specification |
|------------|--------------------|
| Drivers in `internal/core/driver/oauth/{google,github}` | `internal/platform/oauthx/{google,github}` (D1) |
| Interface named `OAuthDriver` | `oauthx.Driver` |
| `GetAuthURL(state, nonce, opts)` | `GetAuthURL(state, redirectURL)` (D4, D5) |
| `ExchangeCode(ctx, code)` | `ExchangeCode(ctx, code, redirectURL)` (D5) |
| `ProviderProfile` | `oauthx.Profile`, mapped by the service to `model.OAuthUserInfo` (D2) |
| Not mentioned | `ID()` and `Scopes()` on the interface; shared `Base` helper (D3); `ErrUnsupportedProvider` sentinel (D6); HTTP client owned by drivers (D7) |

## 14. Acceptance Criteria Mapping

| Issue criterion | Satisfied by |
|-----------------|--------------|
| Clean interface definition | Section 5 |
| Google and GitHub drivers functional | Section 6 and driver tests |
| Registry lookup returns driver or unsupported provider error | Section 5.3 and `registry_test.go` |
| Existing endpoints backward compatible | Sections 7 and 9, controller and integration tests |
| Unit and integration tests for registry and driver resolution | Section 10 |
