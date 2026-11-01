# Passwordless OTP Login via Email — Design Specification

- **Issue:** [kilip/omed#35](https://github.com/kilip/omed/issues/35)
- **Date:** 2026-11-02
- **Status:** Draft, pending written-spec review
- **Scope:** `apps/api` (Phase 1), `apps/web` and `e2e` (Phase 2)

## 1. Context

Omed signs users in with Google and GitHub only (`oauthx` drivers, `POST /auth/sign-in/social`, `GET|POST /auth/callback/:id`). Users without a Google or GitHub account, or who prefer not to use them, cannot sign in.

Existing building blocks this feature reuses:

- `SessionService.CreateSession` returns a raw session token. The same value is the `omed_session` cookie and is accepted as `Authorization: Bearer <token>` (`AuthController.extractToken`).
- `platform/mailx` provides SMTP and in-memory drivers, an async queue, and embedded `templates/*.{html,txt}.tmpl`.
- The Fiber rate limiter in `config/fiber.go` already treats every `/auth/*` path as a sensitive route (per-IP, default 10 requests per minute).
- User find-or-create with personal-workspace provisioning exists as private logic in `oauthService.resolveUser`.
- `testctl` exposes `GET /test/mailer/last`, `GET /test/mailer/messages`, and `POST /test/mailer/reset` in `development` and `test` environments.

## 2. Goals and Non-Goals

### Goals

1. Add `POST /auth/otp/request` and `POST /auth/otp/verify` for passwordless email login.
2. Auto-provision the user on first successful verification with `email_verified = true`.
3. Produce a session identical to the OAuth flow (cookie `omed_session`, same token accepted as Bearer).
4. Enforce single-use, TTL, bounded attempts, constant-time comparison, and per-email and per-IP rate limits.
5. Share user provisioning between OAuth and OTP through one method on `UserService`.
6. Add an email form and code step to the web login page, plus e2e coverage.

### Non-Goals

- Magic links, SMS, or any channel other than email.
- Redis or any new runtime dependency.
- Per-email account lockout.
- Returning the session token in a response body (see decision D8).
- Hiding the OTP form in the web UI when OTP is unavailable (follow-up: `OMED_WEB_OTP_ENABLED`).
- Using OTP to link additional identities to an existing account.

## 3. Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | OTP state lives in a new Ent table `core.login_otps`, not in a cache. | No cache layer exists in the API. The in-memory limiter does not hold across replicas (Helm). Redis is provisioned in devcontainer and CI only. |
| D2 | `code_hash = hex(HMAC-SHA256(AuthSecret, "omed:otp:v1:" + email + ":" + code))`. | A six-digit code is trivially brute-forced from a plain hash if the table leaks. Keying with `AuthSecret` and binding to the email prevents offline guessing and cross-email reuse. No new secret is needed. |
| D3 | At most one active OTP per email. Issuing a new one marks older unconsumed rows as consumed in the same transaction. | Bounds the guess budget per email and removes ambiguity about which code is valid. |
| D4 | `attempts` is incremented atomically before the code comparison. | Parallel requests cannot exceed the attempt limit by racing the comparison. |
| D5 | Consumption is `UPDATE ... SET consumed_at = now WHERE id = ? AND consumed_at IS NULL`; success requires exactly one affected row. | Guarantees single use under concurrency. |
| D6 | Cooldown and hourly limits are soft under concurrency; attempts and single use are strict. | Two simultaneous requests can both pass the cooldown check and send one extra email. That is harmless, so no row locking is added. |
| D7 | Every failure in verify maps to one error, `AUTH_OTP_INVALID` (400). Request always returns the same `202` body regardless of whether the email is registered or the send was skipped by a limit. | Prevents account enumeration and prevents learning which limit applied. |
| D8 | The raw session token is not returned in the response body. The session is delivered by `Set-Cookie`, and the same token is usable as Bearer, exactly as with OAuth today. | Identical to the OAuth flow. Native and CLI clients are out of scope; a `token` field can be added later. |
| D9 | OTP is available only when `OMED_MAILER_DRIVER=smtp` or the environment is `development` or `test`. Otherwise both endpoints return `503 AUTH_OTP_UNAVAILABLE`. | The default mailer driver is `memory`, so in production without SMTP codes would never be delivered. This mirrors OAuth, where a provider is active only when its credentials exist. No new env var. |
| D10 | Cleanup is lazy: each request deletes that email's rows older than 24 hours. | No scheduler exists. Retention of 24 hours is enough for the one-hour request cap. |
| D11 | The request response carries `codeLength`, `ttlSeconds`, and `resendAfterSeconds`. | The UI renders the right input and countdown without hardcoding configuration. These are server configuration values, not per-email data, so nothing leaks. |
| D12 | The email language follows the request `locale`, falling back to `Accept-Language`, normalized with the existing `NormalizeLocaleAndCurrency`. | Omed is bilingual (`en`, `id`). |

## 4. Data Model

`apps/api/ent/schema/core_login_otp.go`, type `LoginOTP`, `entsql.Schema("core")`:

| Field | Type | Notes |
|-------|------|-------|
| `id` | `int64` | Same convention as other core schemas. |
| `email` | `string`, not empty | Normalized: trimmed and lowercased. |
| `code_hash` | `string`, not empty | See D2. |
| `expires_at` | `time` | `created_at + OTPTTL`. |
| `attempts` | `int`, default 0 | Incremented per verify attempt. |
| `consumed_at` | `time`, optional, nillable | Set when used or superseded. |
| `ip_address` | `string`, optional | Requesting IP. |
| `created_at` | `time`, default now, immutable | |

Indexes: `(email, created_at)` and `(expires_at)`.

Migration: `cd apps/api && go generate ./ent`, then `bun run db:diff` with the migration name `add_core_login_otp`. Commit `apps/api/ent/*` and `apps/api/migrations/*`.

## 5. Configuration and Package Layout

### 5.1 Environment variables

New fields in `apps/api/internal/config/config.go`:

| Env | Field | Default | Validation |
|-----|-------|---------|------------|
| `OMED_OTP_LENGTH` | `OTPLength` | `6` | 6 to 10 |
| `OMED_OTP_TTL` | `OTPTTL` | `10m` | greater than 0 |
| `OMED_OTP_MAX_ATTEMPTS` | `OTPMaxAttempts` | `5` | at least 1 |
| `OMED_OTP_RESEND_COOLDOWN` | `OTPResendCooldown` | `60s` | greater than 0 |
| `OMED_OTP_HOURLY_LIMIT` | `OTPHourlyLimit` | `5` | at least 1 |

`Config.Validate()` rejects values outside these ranges. The core module receives an `OTPConfig` struct built at the composition root that assembles `core.Config`; `OTPConfig.Enabled` is computed per D9.

Per AGENTS.md section 6, all five variables are non-secret: add them to `helm/values.yaml`, `helm/values.schema.json`, and the `omed.api.env` helper in `helm/templates/_helpers.tpl`, mirroring the existing `OMED_RATE_LIMIT_*` entries. Also document them in `docs/configuration.md` and `apps/api/.env.example`.

### 5.2 Package layout

```
apps/api/
├── ent/schema/core_login_otp.go
├── internal/core/
│   ├── domain/otp.go            # OTP entity, ErrOTPInvalid, ErrOTPUnavailable
│   ├── model/otp.go             # request/response DTOs
│   ├── repository/otp.go        # + otp_test.go
│   ├── service/otp.go           # + otp_test.go
│   ├── service/user.go          # + FindOrCreateVerified
│   ├── controller/otp.go        # + otp_test.go
│   ├── controller/auth.go       # cookie helper extracted
│   ├── routes.go                # two new routes
│   └── test/otp_test.go         # integration
└── internal/platform/mailx/templates/
    ├── otp.html.tmpl
    └── otp.txt.tmpl
```

## 6. Interfaces

### 6.1 Domain and errors

```go
// domain/otp.go
type OTP struct {
    ID         int64
    Email      string
    CodeHash   string
    ExpiresAt  time.Time
    Attempts   int
    ConsumedAt *time.Time
    IPAddress  string
    CreatedAt  time.Time
}

var ErrOTPInvalid     = errors.New("AUTH_OTP_INVALID")     // 400 "invalid or expired code"
var ErrOTPUnavailable = errors.New("AUTH_OTP_UNAVAILABLE") // 503 "email sign-in is not available"
```

Both are registered with `base.AddToErrorMap` in the domain `init()`.

### 6.2 DTOs

```go
// model/otp.go
type OTPRequest struct {
    Email  string `json:"email" validate:"required,email"`
    Locale string `json:"locale,omitempty"`
}

type OTPRequestResponse struct {
    Success            bool `json:"success"`
    CodeLength         int  `json:"codeLength"`
    TTLSeconds         int  `json:"ttlSeconds"`
    ResendAfterSeconds int  `json:"resendAfterSeconds"`
}

type OTPVerifyRequest struct {
    Email string `json:"email" validate:"required,email"`
    Code  string `json:"code" validate:"required"`
}
```

The verify response is the existing `model.SessionResponse`.

### 6.3 Repository contract (defined in `service/otp.go`)

```go
type OTPRepository interface {
    // Issue marks the email's unconsumed OTPs as consumed and inserts o, in one transaction.
    Issue(ctx context.Context, o domain.OTP) (domain.OTP, error)
    // LastIssuedAt returns the creation time of the email's newest OTP; ok is false if none.
    LastIssuedAt(ctx context.Context, email string) (t time.Time, ok bool, err error)
    // CountSince counts OTPs created for the email at or after since.
    CountSince(ctx context.Context, email string, since time.Time) (int, error)
    // FindActive returns the newest unconsumed, unexpired OTP; base.ErrNotFound if none.
    FindActive(ctx context.Context, email string, now time.Time) (domain.OTP, error)
    // IncrementAttempts atomically increments attempts only if the row is unconsumed
    // and attempts < max. It returns base.ErrNotFound when no row was updated.
    IncrementAttempts(ctx context.Context, id int64, max int) (int, error)
    // Consume marks the OTP consumed; it reports true only if this call consumed it.
    Consume(ctx context.Context, id int64, now time.Time) (bool, error)
    // DeleteOlderThan removes the email's OTPs created before the cutoff.
    DeleteOlderThan(ctx context.Context, email string, before time.Time) error
}
```

### 6.4 Service

```go
type OTPService interface {
    Request(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error)
    Verify(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error)
}
```

`NewOTPService` takes the repository, `UserService`, `SessionService`, `mailx.Mailer`, `AuthSecret`, `OTPConfig`, and a logger.

**Request flow**

1. If `!Enabled`, return `ErrOTPUnavailable`.
2. Normalize the email (trim, lowercase) and resolve the locale (D12).
3. `DeleteOlderThan(email, now - 24h)`.
4. If `LastIssuedAt` is within `OTPResendCooldown`, or `CountSince(now - 1h) >= OTPHourlyLimit`, skip sending and go to step 7.
5. Generate the code with `crypto/rand`, one uniform digit per position (`rand.Int(rand.Reader, 10)`), of length `OTPLength`.
6. `Issue` the hashed OTP, then `SendTemplateAsync(ctx, "otp", data, msg)`. A send failure is logged and never returned.
7. Return `OTPRequestResponse{Success: true, CodeLength, TTLSeconds, ResendAfterSeconds}` built from configuration only.

A repository failure in step 3, 4, or 6 returns an internal error (500). It carries no information about the email.

**Verify flow**

1. If `!Enabled`, return `ErrOTPUnavailable`.
2. Normalize the email. If the code is not exactly `OTPLength` ASCII digits, return `ErrOTPInvalid` without touching the database.
3. `FindActive`; not found returns `ErrOTPInvalid`.
4. `IncrementAttempts(id, OTPMaxAttempts)`; no row updated returns `ErrOTPInvalid`.
5. Compare with `hmac.Equal` against the stored hash; mismatch returns `ErrOTPInvalid`.
6. `Consume`; `false` returns `ErrOTPInvalid`.
7. `UserService.FindOrCreateVerified`, then `SessionService.CreateSession`.

The code and its hash are never logged.

Known property: step 3 returns faster than step 5, so timing reveals whether an unexpired OTP exists for the email. Because request creates an OTP for any email, registered or not, this does not reveal whether an account exists.

### 6.5 Provisioning refactor

```go
// service/user.go
func (s UserService) FindOrCreateVerified(ctx context.Context, in ProvisionInput) (domain.User, error)

type ProvisionInput struct {
    Email, Name, Avatar, Locale string
}
```

Behavior moved verbatim from `oauthService.resolveUser` steps 3 and 4: find by email; otherwise create with `EmailVerified: true`, locale and currency from `NormalizeLocaleAndCurrency`, and a name defaulting to the email local part. A create that fails with `ErrEmailTaken` re-fetches by email. `oauthService` keeps its account lookup (step 1) and its `ErrEmailNotVerified` check (step 2), then delegates. Wiring of the dependency is an implementation-plan detail; OAuth behavior is unchanged.

### 6.6 Controller and routes

`controller/otp.go` defines `OTPController`. The session cookie set and clear logic in `AuthController` moves to a small shared unexported type holding the cookie name and the Secure flag, used by both controllers, so `Callback` and OTP verify set the cookie identically (HttpOnly, `SameSite=Lax`, `Path=/`, expiry from the session).

Routes added to the `/auth` group in `core/routes.go`:

- `POST /auth/otp/request`: `202` with `OTPRequestResponse`; `400` for invalid body or email; `503` when unavailable.
- `POST /auth/otp/verify`: `200` with `SessionResponse` and `Set-Cookie: omed_session`; `400 AUTH_OTP_INVALID`; `503` when unavailable.

Both carry Swaggo annotations under the `Authentication` tag.

### 6.7 Mail template

`otp.html.tmpl` and `otp.txt.tmpl` receive `{Code, TTLMinutes, Locale}` and branch on `Locale` for `en` and `id`. The service sets the subject per locale. The code appears in the text body so e2e can extract it.

## 7. Rate Limiting Summary

| Layer | Mechanism | Limit |
|-------|-----------|-------|
| Per IP | Existing sensitive-route limiter, applies to `/auth/otp/*` | 10 requests per minute per IP (`OMED_RATE_LIMIT_SENSITIVE_*`) |
| Per email, request | Cooldown plus hourly cap in the service (D6) | 1 per 60 s, 5 per hour |
| Per OTP, verify | Atomic attempts counter (D4) | 5 attempts |

Worst-case guessing per email is 5 OTPs per hour times 5 attempts, which is 25 guesses per hour against 10^6 codes, under 0.003% per hour.

## 8. Web (Phase 2)

- `lib/auth-client.ts`: add `authClient.otp.request({ email, locale })` and `authClient.otp.verify({ email, code })`, using `api.POST` and types from `@omed/openapi`, following the `signIn.social` error-handling pattern. Types are available after the Phase 1 regeneration.
- `components/OtpLoginForm.tsx` (with a test next to it): two steps inside the login card.
  - Email step: email input and submit.
  - Code step: Ant Design `Input.OTP` with `codeLength` from the request response, auto-submit when complete, a resend button disabled for `resendAfterSeconds` with a visible countdown, and a "change email" action.
- `routes/login.tsx`: render `OtpLoginForm` below the social buttons, separated by an "or" divider. The existing `clientLoader` redirect for signed-in users stays.
- On verify success the cookie is already set; navigate to `/` and let the root loader fetch the session.
- `AUTH_OTP_INVALID` and `AUTH_OTP_UNAVAILABLE` map to translated messages.
- i18n keys `login.otp.*` in `i18n/en.ts` and `i18n/id.ts`.

## 9. Backward Compatibility

- OAuth endpoints, cookies, and responses are unchanged. The provisioning refactor preserves behavior and is covered by the existing OAuth tests.
- The migration is additive (one new table).
- The OpenAPI change is additive (two new paths and three new schemas).
- No existing environment variable changes meaning.

## 10. Testing Strategy

**Go unit tests** (next to the file under test)

- `repository/otp_test.go`: `Issue` supersedes older rows; `IncrementAttempts` stops at max and on consumed rows; `Consume` succeeds once; `DeleteOlderThan` keeps newer rows.
- `service/otp_test.go`: happy path for new and existing users; wrong code increments attempts; attempts exhausted rejects even the correct code; expired; replay after success; cooldown and hourly cap skip sending yet return the same response; email normalization; malformed code does not consume attempts; mailer failure does not fail the request; unavailable returns `ErrOTPUnavailable`.
- `service/user_test.go`: `FindOrCreateVerified` finds existing, creates new, handles the `ErrEmailTaken` race.
- `service/oauth_test.go`: unchanged assertions pass after the refactor.
- `controller/otp_test.go`: status codes, response shapes, cookie attributes.

**Integration** (`core/test/otp_test.go`)

- Full request then verify through Fiber with the memory mailer; the returned cookie and the same token as Bearer both succeed on `GET /auth/get-session`.
- Concurrent verify with the correct code yields exactly one success.
- Per-IP limit with the `X-Test-Rate-Limit: true` header.

**Web** (vitest): the otp methods in `auth-client.test.ts`; `OtpLoginForm` step transitions, countdown, and error display.

**E2E** (`e2e/`)

- Feature `features/core/otp-login.feature` with scenarios: new user signs in and lands on home; existing user signs in; wrong code shows an error; resend is disabled during the countdown.
- Steps in `steps/core/authentication.steps.ts`, locators in `pages/core/login.page.ts`, and a helper in `support/test-client.ts` that calls `POST /test/mailer/reset` before the request and extracts the code from `GET /test/mailer/last`.
- Created users are removed with the existing `DELETE /test/users/:id`.
- AGENTS.md section 5 currently says sign-in is social only; reword it to say the `@authenticated` hook creates sessions through the API test endpoints and does not exercise a sign-in method.

## 11. Documentation

- `docs/architecture.md`: new "Passwordless OTP" section covering the flow, storage, and availability rule.
- `docs/configuration.md` and `apps/api/.env.example`: the five `OMED_OTP_*` variables.
- `README.md`: change the "Social Sign-In" bullet to cover email OTP.
- `AGENTS.md` section 5: the wording fix above.
- `helm/README.md`: only if it lists non-secret variables explicitly.

## 12. Phases, Commits, and Branch Conventions

Branch off `main` (for example `feat/otp-login`). Commits follow `type(scope): <emoji> subject` and the allowed scopes in AGENTS.md.

**Phase 1: API**

1. `refactor(auth): ♻️ extract verified user provisioning`
2. `feat(api): ✨ add login otp schema and migration`
3. `feat(api): ✨ add passwordless otp login endpoints` (repository, service, controller, routes, config, mail template, and the regenerated `apps/api/docs/*` and `packages/openapi/src/api.ts`, because the generated contract belongs with the annotations that produce it)
4. `feat(deploy): ✨ add otp settings to helm chart`, validated with `helm lint helm --strict -f helm/ci/full-values.yaml` and `helm template omed helm -f helm/ci/full-values.yaml | kubeconform -strict -summary`
5. `docs(docs): 📝 document passwordless otp login`

**Phase 2: web and e2e**

6. `feat(web): ✨ add email otp sign-in to login page`
7. `test(e2e): ✅ cover passwordless otp login`

Before each push run `bun run lint`, `bun run typecheck`, `bun run test`, `cd apps/api && go test ./...`, and `bun run e2e` for Phase 2.

## 13. Deviations from Issue #35

| Issue text | This design | Why |
|------------|-------------|-----|
| "Stores hashed OTP in database/cache" | Database table only | D1 |
| "OpenAPI spec definition in `packages/openapi`" | Swaggo annotations in the controller, then regenerate `apps/api/docs/*` and `packages/openapi/src/api.ts` | `packages/openapi` is generated, per AGENTS.md section 3 |
| "Creates active session (`omed_session` cookie and Bearer token) identical to OAuth flow" | Cookie is set; the same token works as Bearer; the token is not in the response body | D8: that is what OAuth does today |
| "Rate limiting per IP and email" | Per-IP from the existing limiter; per-email in the service | Section 7 |
| No UI, e2e, config, Helm, or availability rule | Added | The feature is not usable or deployable without them |
| "Accepts email and OTP code ... verifies retry limits" | Single `AUTH_OTP_INVALID` error for every failure | D7 |

## 14. Acceptance Criteria Mapping

| Issue requirement | Where covered |
|-------------------|---------------|
| `POST /auth/otp/request`: validate email, rate limit, secure OTP with configurable TTL, hashed storage with attempt counter, send email | 6.4 request flow, 4, 5, 7 |
| `POST /auth/otp/verify`: code, TTL, retry limits, upsert user with verified email, session | 6.4 verify flow, 6.5, 6.6 |
| Rate limiting on both endpoints | 7 |
| Constant-time or hash verification | D2, 6.4 step 5 |
| Single use and expiry invalidation | D3, D5, 6.4 |
| OpenAPI definition | 6.6, 12, 13 |
| Unit and integration test coverage | 10 |
