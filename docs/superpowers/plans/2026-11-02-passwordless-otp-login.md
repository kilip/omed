# Passwordless OTP Login Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users sign in with an emailed one-time code, with the same session as the OAuth flow, plus the login form and e2e coverage.

**Architecture:** A new `core.login_otps` table stores HMAC-hashed codes. `OTPService` (core module) issues and verifies codes, sends mail through `mailx`, and reuses a new `UserService.FindOrCreateVerified` shared with OAuth. Two controller handlers under `/auth/otp/*` set the existing `omed_session` cookie. The web login card gains an `OtpLoginForm`.

**Tech Stack:** Go, Fiber v3, Ent, Atlas, Swaggo, testify, gomock, Helm; React 19, React Router, Ant Design 6, openapi-fetch, Vitest; Cucumber and Playwright.

**Spec:** `docs/superpowers/specs/2026-11-02-passwordless-otp-login-design.md` (issue kilip/omed#35). Executors read the spec and this plan together.

## Global Constraints

- Branch: create `feat/otp-login` from `docs/otp-login-design` so the spec and plan are present.
- Commits: `type(scope): <emoji> subject`, scope from `auth|web|api|agent|e2e|deps|ci|release|tools|deploy|docs`, imperative lowercase subject. Each commit message ends with these trailer lines:
  `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_017hFcA9dAmzuvnfJdjpfQi4`
- Layers: `controller -> service -> repository`. Controllers never touch Ent or repositories.
- Table `core.login_otps`, Ent type `LoginOTP`, file `core_login_otp.go`, indexes `(email, created_at)` and `(expires_at)`.
- Hash: `hex(HMAC-SHA256(AuthSecret, "omed:otp:v1:" + email + ":" + code))`. Compare with `hmac.Equal`. Never log the code or the hash.
- Defaults: length 6 (allowed 6 to 10), TTL `10m`, max attempts 5, resend cooldown `60s`, hourly limit 5. Env: `OMED_OTP_LENGTH`, `OMED_OTP_TTL`, `OMED_OTP_MAX_ATTEMPTS`, `OMED_OTP_RESEND_COOLDOWN`, `OMED_OTP_HOURLY_LIMIT`.
- Errors: `AUTH_OTP_INVALID` is HTTP 400 "invalid or expired code"; `AUTH_OTP_UNAVAILABLE` is HTTP 503 "email sign-in is not available". Every verify failure is `AUTH_OTP_INVALID`.
- Availability: enabled only when `OMED_MAILER_DRIVER=smtp` or `OMED_ENV` is `development` or `test`.
- Request always returns `202` with `{success, codeLength, ttlSeconds, resendAfterSeconds}` built from configuration only. JSON keys are camelCase.
- The session token is never returned in a response body. Verify returns `model.SessionResponse` plus `Set-Cookie`.
- Go tests: `cd apps/api && go test ./...`. Web tests: `bun run test`. Run `bun run lint`, `bun run typecheck` before pushing web work. Documentation is English Markdown.
- Commits here are finer than spec section 12. The rule that generated OpenAPI files travel with the annotation change is kept in Task 6.

## Review Focus

Inputs the spec implies but its listed tests do not reach. Each line has a test in the owning task.

- Email typed with different case or padding at request and at verify (`" Alice@Example.COM "` then `alice@example.com`) must hit the same OTP. Task 5.
- A code pasted with surrounding whitespace (`" 123456\n"`) should verify; trim before the digit check. Task 5.
- An existing user stored with a mixed-case email (OAuth keeps provider casing) must be found, not duplicated. Task 1.
- An unsupported locale (`"fr-FR"`) sends the English email and never errors. Task 5.
- Non-ASCII digits (Arabic-Indic `"١٢٣٤٥٦"`) and very long codes are rejected without touching the database or attempts. Task 5.

---

## Phase 1: API

### Task 1: Shared verified-user provisioning

**Files:**
- Modify: `apps/api/internal/core/service/user.go`, `apps/api/internal/core/service/oauth.go`, `apps/api/internal/core/repository/user.go`
- Test: `apps/api/internal/core/service/user_test.go`, `apps/api/internal/core/repository/user_test.go`, existing `service/oauth_test.go` (must stay green)

**Interfaces:**
- Produces:
  - `type ProvisionInput struct { Email, Name, Avatar, Locale string }` in `service/user.go`
  - `func (s UserService) FindOrCreateVerified(ctx context.Context, in ProvisionInput) (domain.User, error)`
  - `type UserProvisioner interface { FindOrCreateVerified(ctx context.Context, in ProvisionInput) (domain.User, error) }` in `service/oauth.go`
  - `OAuthServiceParams.Provisioner UserProvisioner` (optional; `NewOAuthService` defaults it to `NewUserService(p.Users)`)
  - `UserRepository.FindByEmail` becomes case-insensitive

- [ ] **Step 1: Write the failing tests**
  - `repository/user_test.go`: `Test_FindByEmail_IsCaseInsensitive`: create user `Alice@Example.com`; `FindByEmail("alice@example.com")` returns that user.
  - `service/user_test.go` (gomock repo): `Test_FindOrCreateVerified_ExistingUser` (FindByEmail returns user, `Create` never called); `Test_FindOrCreateVerified_NewUser` (FindByEmail returns `base.ErrNotFound`; `Create` called with `CreateUserRequest{Name: "bob", Email: "bob@example.com", EmailVerified: true, Locale: "id", Currency: "IDR"}` for input locale `"id-ID"` and empty name); `Test_FindOrCreateVerified_EmailTakenRace` (Create returns `domain.ErrEmailTaken`, second FindByEmail returns the user); `Test_FindOrCreateVerified_EmptyEmailSkipsLookup` (no FindByEmail call when Email is empty).
- [ ] **Step 2: Run to verify failure**
  Run: `cd apps/api && go test ./internal/core/service ./internal/core/repository -run 'Test_FindOrCreateVerified|Test_FindByEmail_IsCaseInsensitive' -count=1`
  Expected: compile failure on `FindOrCreateVerified`, then FAIL on case-insensitivity.
- [ ] **Step 3: Implement**
  - `FindOrCreateVerified` moves steps 3 and 4 of `oauthService.resolveUser` verbatim: a nil error from `FindByEmail` returns that user (skipped when Email is empty); otherwise create with `EmailVerified: true`, locale and currency from `NormalizeLocaleAndCurrency`, name defaulting to the email local part; `ErrEmailTaken` re-fetches by email.
  - `oauthService.resolveUser` keeps steps 1 and 2, computes the final name (including its `User-<accountID>` fallback), then calls `s.provisioner.FindOrCreateVerified`.
  - `repository.UserRepository.FindByEmail` uses `user.EmailEqualFold(email)` with `.Order(ent.Asc(user.FieldID)).First(c)` and maps `ent.IsNotFound` to `base.ErrNotFound`.
  - Regenerate the mock: `cd apps/api/internal/core/service && go generate ./...` only if the `UserRepository` interface changed (it should not).
- [ ] **Step 4: Run full core tests**
  Run: `cd apps/api && go test ./internal/core/... -count=1`
  Expected: PASS, including every existing OAuth test.
- [ ] **Step 5: Commit**
  ```bash
  git add apps/api/internal/core
  git commit -m "refactor(auth): ♻️ extract verified user provisioning"
  ```

### Task 2: `LoginOTP` schema and migration

**Files:**
- Create: `apps/api/ent/schema/core_login_otp.go`, `apps/api/migrations/<timestamp>_add_core_login_otp.sql`
- Modify: generated `apps/api/ent/*`, `apps/api/migrations/atlas.sum`

**Interfaces:**
- Produces: Ent entity `ent.LoginOTP` with builders and predicates in package `ent/loginotp`. Fields: `id int64`, `email string`, `code_hash string`, `expires_at time`, `attempts int` (default 0), `consumed_at *time` (optional, nillable), `ip_address string` (optional), `created_at time` (default now, immutable). Annotation `entsql.Schema("core")` and an explicit `entsql.Annotation{Table: "login_otps"}`. Indexes `(email, created_at)` and `(expires_at)`.

- [ ] **Step 1: Write the schema** `LoginOTP` in `core_login_otp.go` following `core_session.go` (no edges, no mixins).
- [ ] **Step 2: Generate** `cd apps/api && go generate ./ent && go build ./...`. Expected: builds.
- [ ] **Step 3: Generate the migration** `cd apps/api && go run ./cmd/migrate add_core_login_otp` (needs the devcontainer Postgres at `OMED_DB_DEV_URL`).
  Expected: the new `.sql` contains `CREATE TABLE "core"."login_otps"` and both indexes; `atlas.sum` is updated.
- [ ] **Step 4: Commit**
  ```bash
  git add apps/api/ent apps/api/migrations
  git commit -m "feat(api): ✨ add login otp schema and migration"
  ```

### Task 3: Domain, DTOs, and `OTPRepository`

**Files:**
- Create: `apps/api/internal/core/domain/otp.go`, `apps/api/internal/core/model/otp.go`, `apps/api/internal/core/repository/otp.go`
- Test: `apps/api/internal/core/repository/otp_test.go`

**Interfaces:**
- Consumes: `ent.LoginOTP` from Task 2.
- Produces (all names exact):
  - `domain.OTP{ID int64; Email, CodeHash string; ExpiresAt time.Time; Attempts int; ConsumedAt *time.Time; IPAddress string; CreatedAt time.Time}`
  - `domain.ErrOTPInvalid` (`"AUTH_OTP_INVALID"`, 400) and `domain.ErrOTPUnavailable` (`"AUTH_OTP_UNAVAILABLE"`, 503), registered with `base.AddToErrorMap` in the domain `init()`
  - `model.OTPRequest{Email string "json:email validate:required,email"; Locale string "json:locale,omitempty"}`
  - `model.OTPRequestResponse{Success bool; CodeLength, TTLSeconds, ResendAfterSeconds int}` with json `success`, `codeLength`, `ttlSeconds`, `resendAfterSeconds`
  - `model.OTPVerifyRequest{Email string "validate:required,email"; Code string "validate:required"}`
  - `repository.NewOTPRepository(cli *ent.Client) OTPRepository` with methods:
    `Issue(ctx, o domain.OTP) (domain.OTP, error)`, `LastIssuedAt(ctx, email string) (time.Time, bool, error)`, `CountSince(ctx, email string, since time.Time) (int, error)`, `FindActive(ctx, email string, now time.Time) (domain.OTP, error)`, `IncrementAttempts(ctx, id int64, max int) (int, error)`, `Consume(ctx, id int64, now time.Time) (bool, error)`, `DeleteOlderThan(ctx, email string, before time.Time) error`

- [ ] **Step 1: Write the failing tests** (suite on `testutil.CreateTestDB` with its own DSN, like `session_test.go`):
  - `Test_Issue_SupersedesPreviousUnconsumed`: issue A then B for one email; `FindActive` returns B; A's `consumed_at` is set.
  - `Test_Issue_DoesNotTouchOtherEmails`.
  - `Test_FindActive_IgnoresExpiredAndConsumed`: returns `base.ErrNotFound`.
  - `Test_IncrementAttempts_StopsAtMax`: with max 3 returns 1, 2, 3, then `base.ErrNotFound`.
  - `Test_IncrementAttempts_IgnoresConsumed`: `base.ErrNotFound`.
  - `Test_Consume_OnlyOnce`: first call `true`, second `false`.
  - `Test_LastIssuedAt_And_CountSince`: `ok=false` for unknown email; counts only rows at or after `since`.
  - `Test_DeleteOlderThan_ScopedToEmail`.
- [ ] **Step 2: Run to verify failure**
  Run: `cd apps/api && go test ./internal/core/repository -run OTP -count=1`
  Expected: compile FAIL.
- [ ] **Step 3: Implement** the repository. `Issue` runs in one `cli.Tx`: mark the email's unconsumed rows consumed with `time.Now()`, then insert. `IncrementAttempts` is a conditional `Update().Where(idEQ, consumedAtIsNil, attemptsLT(max)).AddAttempts(1)`; zero rows affected returns `base.ErrNotFound`, otherwise re-read and return `attempts`. `Consume` is a conditional update on `consumed_at IS NULL` and reports `rowsAffected == 1`. Map `ent.IsNotFound` to `base.ErrNotFound`.
- [ ] **Step 4: Run to verify pass** (same command). Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add apps/api/internal/core/domain apps/api/internal/core/model apps/api/internal/core/repository
  git commit -m "feat(api): ✨ add login otp repository"
  ```

### Task 4: OTP configuration

**Files:**
- Modify: `apps/api/internal/config/config.go`
- Test: `apps/api/internal/config/config_test.go`

**Interfaces:**
- Produces on `config.Config`: `OTPLength int` (`OMED_OTP_LENGTH,default=6`), `OTPTTL time.Duration` (`default=10m`), `OTPMaxAttempts int` (`default=5`), `OTPResendCooldown time.Duration` (`default=60s`), `OTPHourlyLimit int` (`default=5`); `func (c Config) IsOTPEnabled() bool`; `func (c Config) ValidateOTP() error`. `GetConfig` calls `ValidateOTP` after `Validate` and exits through `logFatalf` on error. `ValidateOTP` is separate from `Validate` so hand-built `Config{}` values in existing tests are unaffected.

- [ ] **Step 1: Write the failing tests**: `Test_GetConfig_OTPDefaults` (6, 10m, 5, 60s, 5); `Test_ValidateOTP` table (length 5 and 11 rejected; 6 and 10 accepted; TTL 0, MaxAttempts 0, ResendCooldown 0, HourlyLimit 0 rejected); `Test_IsOTPEnabled` table: smtp+production true, memory+production false, memory+development true, memory+test true, smtp+test true.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/config -run 'OTP' -count=1`. Expected: FAIL.
- [ ] **Step 3: Implement** the fields, `IsOTPEnabled` (`MailerDriver == "smtp" || IsTestRoutesEnabled()`), and `ValidateOTP` (length 6 to 10, other values at least 1 or greater than 0). Error text names the env var, for example `OMED_OTP_LENGTH must be between 6 and 10`.
- [ ] **Step 4: Run** the same command plus `go test ./internal/config -count=1`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add apps/api/internal/config
  git commit -m "feat(api): ✨ add otp configuration"
  ```

### Task 5: `OTPService` and the `otp` email template

**Files:**
- Create: `apps/api/internal/core/service/otp.go`, `apps/api/internal/platform/mailx/templates/otp.html.tmpl`, `apps/api/internal/platform/mailx/templates/otp.txt.tmpl`
- Test: `apps/api/internal/core/service/otp_test.go`

**Interfaces:**
- Consumes: `repository.OTPRepository` (Task 3), `service.UserProvisioner` (Task 1), `SessionService`, `mailx.Mailer`.
- Produces:
  - `type OTPRepository interface` in `service/otp.go` with exactly the seven methods listed in Task 3
  - `type OTPConfig struct { Enabled bool; Length int; TTL time.Duration; MaxAttempts int; ResendCooldown time.Duration; HourlyLimit int }`
  - `type OTPServiceParams struct { Repo OTPRepository; Users UserProvisioner; Sessions SessionService; Mailer mailx.Mailer; AuthSecret string; Config OTPConfig; Logger *slog.Logger; Now func() time.Time }` (`Now` optional, defaults to `time.Now`)
  - `type OTPService interface { Request(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error); Verify(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error) }`
  - `func NewOTPService(p OTPServiceParams) OTPService`
- Template data: `{Code string, TTLMinutes int, Locale string}`. Both bodies branch on `Locale` (`en`, `id`). The text body contains the code as a standalone token and the TTL minutes; the `id` variant contains the word `menit`. Subjects: en `Your Omed sign-in code`, id `Kode masuk Omed Anda`. The template name passed to `SendTemplateAsync` is `otp`.

- [ ] **Step 1: Write the failing tests.** Real SQLite repos via `testutil.CreateTestDB`, real `UserService` and `SessionService`, `mailx.New(mailx.Config{Driver: "memory"})`; read mail from `mailer.Driver().(*mailx.MemoryDriver).SentMessages()` inside `s.Eventually`; extract the code from `TextBody` with `\b\d{6}\b`. Use `OTPConfig{Enabled: true, Length: 6, TTL: 10m, MaxAttempts: 3, ResendCooldown: 1ms, HourlyLimit: 5}` unless noted, and an injected `Now`.
  - Request: `Test_Request_SendsCodeAndStoresOnlyHash` (6 digits in mail; stored `code_hash` is 64 hex chars and not equal to the code); `Test_Request_ResponseIdenticalForKnownAndUnknownEmail`; `Test_Request_CooldownSkipsSend` (cooldown 60s: two calls, one mail, equal responses); `Test_Request_HourlyLimitSkipsSend` (limit 2: third call sends no mail, still success); `Test_Request_Unavailable` (`ErrOTPUnavailable`); `Test_Request_MailerFailureStillSucceeds` (stub embedding `mailx.Mailer` whose `SendTemplateAsync` errors); `Test_Request_UnsupportedLocaleFallsBackToEnglish` (locale `fr-FR`: subject is the English one, no error).
  - Verify: `Test_Verify_NewUser_ProvisionsVerifiedUserAndSession` (user created with `EmailVerified` true; `ValidateSession(token)` succeeds); `Test_Verify_ExistingUser_ReusesUser`; `Test_Verify_WrongCodeCountsAttemptsThenLocks` (3 wrong codes, then the correct code still returns `ErrOTPInvalid`); `Test_Verify_ReplayRejected`; `Test_Verify_ExpiredRejected` (advance `Now` past TTL); `Test_Verify_SupersededCodeRejected` (code from the first request fails after a second request); `Test_Verify_NoActiveOTP`; `Test_Verify_ConcurrentExactlyOneSuccess` (20 goroutines, correct code, exactly one nil error and one session row).
  - Review Focus: `Test_Request_ThenVerify_WithDifferentlyCasedEmail`; `Test_Verify_TrimsWhitespaceAroundCode` (`" 123456\n"` form of the real code succeeds); `Test_Verify_RejectsNonASCIIDigitsAndLongCodes` (`"١٢٣٤٥٦"` and a 10,000 character string return `ErrOTPInvalid` and leave `attempts` at 0).
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/service -run OTP -count=1`. Expected: compile FAIL.
- [ ] **Step 3: Implement the templates** (HTML plus text, branching on `.Locale`).
- [ ] **Step 4: Implement `NewOTPService`.**
  - Normalize email with `strings.ToLower(strings.TrimSpace(...))`; resolve locale as `NormalizeLocaleAndCurrency(req.Locale)` first element.
  - Generate digits with `rand.Int(rand.Reader, big.NewInt(10))` per position.
  - Hash with `hmac.New(sha256.New, []byte(authSecret))` over the Global Constraints message, hex-encoded.
  - Follow spec section 6.4 step by step for both flows; verify trims whitespace, then requires exactly `Config.Length` bytes in `'0'..'9'` before any database call.
  - Send with `SendTemplateAsync(ctx, "otp", data, msg)` and log (never return) a send error.
- [ ] **Step 5: Run** the same command. Expected: PASS (run once with `-race`).
- [ ] **Step 6: Commit**
  ```bash
  git add apps/api/internal/core/service apps/api/internal/platform/mailx/templates
  git commit -m "feat(api): ✨ add otp service and email template"
  ```

### Task 6: Controller, routes, wiring, and OpenAPI

**Files:**
- Create: `apps/api/internal/core/controller/otp.go`
- Modify: `apps/api/internal/core/controller/auth.go`, `apps/api/internal/core/routes.go`, `apps/api/internal/core/module.go`, `apps/api/internal/config/app.go`
- Test: `apps/api/internal/core/controller/otp_test.go`
- Regenerate: `apps/api/docs/*`, `packages/openapi/src/api.ts`

**Interfaces:**
- Consumes: `service.OTPService`, `service.OTPConfig` (Task 5), `config.Config.IsOTPEnabled` and OTP fields (Task 4).
- Produces:
  - unexported `sessionCookies{name string; secure bool}` in `controller/auth.go` with `set(c fiber.Ctx, sess domain.Session)` and `clear(c fiber.Ctx)`; `AuthController` uses it for `Callback` and `SignOut`
  - `func NewOTPController(otp service.OTPService, cookieName string, cookieSecure bool) OTPController`
  - `func (o OTPController) Request(c fiber.Ctx) error` (202 with `model.OTPRequestResponse`) and `func (o OTPController) Verify(c fiber.Ctx) error` (200 with `model.SessionResponse`, `Set-Cookie`)
  - `core.Config.OTP service.OTPConfig`; `CoreModule.OTPService service.OTPService`
  - routes `POST /auth/otp/request` and `POST /auth/otp/verify`

- [ ] **Step 1: Write the failing controller tests** with a mock `OTPService` and an app built by `config.GetFiber(config.Config{Env: "test"}, slog.Default())`: `Test_Request_Returns202WithBody`; `Test_Request_InvalidEmail400`; `Test_Request_DefaultsLocaleFromAcceptLanguage`; `Test_Request_Unavailable503` (error code `AUTH_OTP_UNAVAILABLE`); `Test_Verify_SetsSessionCookieAndReturnsSession` (cookie `omed_session`, HttpOnly, SameSite Lax, body `user.email`); `Test_Verify_InvalidCode400` (error code `AUTH_OTP_INVALID`); `Test_Verify_PassesIPAndUserAgent`. Confirm the existing `auth_test.go` still passes after the cookie helper extraction.
- [ ] **Step 2: Run** `cd apps/api && go test ./internal/core/controller -count=1`. Expected: compile FAIL.
- [ ] **Step 3: Implement** the controller with Swaggo annotations (`@Tags Authentication`, `@Router /auth/otp/request [post]` and `/auth/otp/verify [post]`, success and failure types as in `SignInSocial`). Wire `repository.NewOTPRepository`, `service.NewOTPService` (with `Users: userSvc`, `Sessions: sessionSvc`, `Mailer: cfg.Mailer`, `AuthSecret`), the two routes, and `OTP: service.OTPConfig{Enabled: state.Config.IsOTPEnabled(), Length: state.Config.OTPLength, TTL: ..., MaxAttempts: ..., ResendCooldown: ..., HourlyLimit: ...}` in `config/app.go`.
- [ ] **Step 4: Run** `cd apps/api && go build ./... && go test ./internal/core/... ./internal/config/... -count=1`. Expected: PASS.
- [ ] **Step 5: Regenerate the contract**
  ```bash
  bunx turbo run oa:gen
  bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts
  grep -c "otp" apps/api/docs/swagger.json packages/openapi/src/api.ts
  ```
  Expected: both paths and schemas `model.OTPRequest`, `model.OTPRequestResponse`, `model.OTPVerifyRequest` present.
- [ ] **Step 6: Commit**
  ```bash
  git add apps/api packages/openapi
  git commit -m "feat(api): ✨ add otp login endpoints"
  ```

### Task 7: Integration tests and per-IP limit coverage

**Files:**
- Create: `apps/api/internal/core/test/otp_test.go`
- Modify: `apps/api/testutil/init.go`, `apps/api/internal/config/fiber_test.go`

**Interfaces:**
- Consumes: the running test app from `testutil` (`GetState().FiberApp`, `GetMailer()`, `GetEntCli()`), `ApiTestSuite.RequestWithToken`.

- [ ] **Step 1: Make OTP available under test.** In `testutil/init.go`, default `OMED_ENV` to `test` when unset (same pattern as the other defaults). Run `cd apps/api && go test ./... -count=1`. Expected: PASS with no other regressions; if a test regresses, fix that test's expectation rather than the default.
- [ ] **Step 2: Write the failing tests** in `core/test/otp_test.go` (suite embedding `testutil.ApiTestSuite`, unauthenticated calls via `RequestWithToken(path, method, body, "")`, decode the raw JSON body):
  - `Test_RequestThenVerify_NewUser_CookieAndBearer`: request returns 202 with `codeLength` 6, `ttlSeconds` 600, `resendAfterSeconds` 60; code read from the memory mailer; verify returns 200 with an `omed_session` cookie; `GET /auth/get-session` succeeds with the cookie and with `Authorization: Bearer <same value>`; the user row has `email_verified` true.
  - `Test_Verify_WrongCode_400_AUTH_OTP_INVALID`.
  - `Test_Request_InvalidEmail_400`.
  - `Test_Request_SameResponseForKnownAndUnknownEmail`.
  - `Test_Verify_ConcurrentExactlyOneSession`: 10 goroutines call `testutil.GetState().FiberApp.Test(req)` with the correct code; exactly one 200.
  In `config/fiber_test.go` add `Test_SensitiveRoute_OTPRequest_RateLimited` mirroring the existing sensitive-route test for `/auth/sign-in/social`, with path `/auth/otp/request` (the shared test app disables the limiter, so the per-IP check lives here).
- [ ] **Step 3: Run** `cd apps/api && go test ./internal/core/test ./internal/config -run 'OTP' -count=1`. Expected: PASS once Task 6 is in place; any failure points at wiring.
- [ ] **Step 4: Commit**
  ```bash
  git add apps/api
  git commit -m "test(api): ✅ cover otp login end to end"
  ```

### Task 8: Helm chart

**Files:**
- Modify: `helm/values.yaml`, `helm/values.schema.json`, `helm/templates/_helpers.tpl`, `helm/ci/full-values.yaml`, `helm/README.md` (only if it lists non-secret settings)

**Interfaces:**
- Produces: values under `api.config.otp` with keys `length` (6), `ttl` (`10m`), `maxAttempts` (5), `resendCooldown` (`60s`), `hourlyLimit` (5), mapped to the five env vars in the `omed.api.env` helper next to `OMED_RATE_LIMIT_ENABLED`. Schema types: `length`, `maxAttempts`, `hourlyLimit` integer (`length` minimum 6, maximum 10, others minimum 1); `ttl`, `resendCooldown` string.

- [ ] **Step 1: Edit** values, schema, helper, and `ci/full-values.yaml` (set every key explicitly there).
- [ ] **Step 2: Verify**
  ```bash
  helm lint helm --strict -f helm/ci/full-values.yaml
  helm template omed helm -f helm/ci/full-values.yaml | grep OMED_OTP_
  helm template omed helm -f helm/ci/full-values.yaml | kubeconform -strict -summary
  ```
  Expected: lint clean, five `OMED_OTP_*` env entries, kubeconform reports 0 invalid.
- [ ] **Step 3: Commit**
  ```bash
  git add helm
  git commit -m "feat(deploy): ✨ add otp settings to helm chart"
  ```

### Task 9: Documentation

**Files:**
- Modify: `docs/configuration.md`, `apps/api/.env.example`, `docs/architecture.md`, `README.md`

- [ ] **Step 1: Edit.**
  - `docs/configuration.md`: five rows after the rate-limit rows, defaults from Global Constraints, plus a note on the availability rule.
  - `.env.example`: the five variables with defaults.
  - `docs/architecture.md`: a "Passwordless OTP" section (flow, storage in `core.login_otps`, HMAC hashing, rate limits, availability rule) placed before "Model Context Protocol (MCP) Server".
  - `README.md`: change the "Social Sign-In" bullet to "Social & Email OTP Sign-In" and mention one-time codes.
- [ ] **Step 2: Verify** `grep -n "OMED_OTP" docs/configuration.md apps/api/.env.example | wc -l` prints 10, and `grep -n "Passwordless OTP" docs/architecture.md` matches once.
- [ ] **Step 3: Commit**
  ```bash
  git add docs apps/api/.env.example README.md
  git commit -m "docs(docs): 📝 document passwordless otp login"
  ```

---

## Phase 2: Web and e2e

### Task 10: `authClient.otp`

**Files:**
- Modify: `apps/web/app/lib/auth-client.ts`
- Test: `apps/web/app/lib/auth-client.test.ts`

**Interfaces:**
- Consumes: regenerated `@omed/openapi` types (Task 6).
- Produces in `auth-client.ts`:
  - `export type OtpRequestBody = components["schemas"]["model.OTPRequest"]`, `OtpRequestResponse = components["schemas"]["model.OTPRequestResponse"]`, `OtpVerifyBody = components["schemas"]["model.OTPVerifyRequest"]`
  - `authClient.otp.request(body: OtpRequestBody): Promise<{ data?: OtpRequestResponse; error?: { message: string; code?: string } }>`
  - `authClient.otp.verify(body: OtpVerifyBody): Promise<{ data?: SessionResponse; error?: { message: string; code?: string } }>`
  - Both call `api.POST("/auth/otp/request" | "/auth/otp/verify", { body })`, read `error.error.code` and `error.error.message` from the error body, and convert thrown errors into `{ error: { message } }`, matching `signIn.social`.

- [ ] **Step 1: Write the failing tests** in the existing mocked-`api` style: `otp.request returns data on success`; `otp.request surfaces AUTH_OTP_UNAVAILABLE code and message`; `otp.verify returns session data`; `otp.verify surfaces AUTH_OTP_INVALID code`; `otp.verify converts a thrown network error`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/lib/auth-client.test.ts`. Expected: FAIL (`authClient.otp` undefined).
- [ ] **Step 3: Implement** the two methods.
- [ ] **Step 4: Run** the same command plus `bun run typecheck`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add apps/web/app/lib
  git commit -m "feat(web): ✨ add otp methods to auth client"
  ```

### Task 11: `OtpLoginForm` and login page

**Files:**
- Create: `apps/web/app/components/OtpLoginForm.tsx`, `apps/web/app/components/OtpLoginForm.test.tsx`
- Modify: `apps/web/app/routes/login.tsx`, `apps/web/app/i18n/en.ts`, `apps/web/app/i18n/id.ts`

**Interfaces:**
- Consumes: `authClient.otp` (Task 10).
- Produces from `OtpLoginForm.tsx` (vitest runs in the `node` environment, so logic is pure and exported):
  - `type OtpState = { step: "email" | "code"; email: string; codeLength: number; resendAt: number }` and `type OtpAction = { type: "requested"; email: string; codeLength: number; resendAfterSeconds: number; now: number } | { type: "changeEmail" }`
  - `otpStepReducer(state: OtpState, action: OtpAction): OtpState`
  - `remainingSeconds(resendAt: number, now: number): number` (ceiling of the difference in seconds, minimum 0)
  - `mapOtpErrorKey(code?: string): "login.otp.invalid" | "login.otp.unavailable" | "login.otp.failed"`
  - `OtpLoginForm(): JSX.Element`: email step (Ant Design `Form`, `Input`, `Button`), code step (`Input.OTP` with `length={codeLength}`, submits when the last digit is entered and clears on error), resend button disabled while `remainingSeconds > 0` with a one-second ticking countdown, "use a different email" action, success calls `navigate("/", { replace: true })`, errors shown with `message.error(t(mapOtpErrorKey(code)))`; sends `locale: i18n.language`.
- i18n keys under `login.otp` in both files (same keys): `divider`, `emailLabel`, `emailPlaceholder`, `sendCode`, `codeHint`, `resend`, `resendIn`, `changeEmail`, `invalid`, `unavailable`, `failed`. English copy fixed because e2e locators depend on it: `or`, `Email`, `you@example.com`, `Send code`, `Enter the {{length}}-digit code we sent to {{email}}.`, `Resend code`, `Resend code in {{seconds}}s`, `Use a different email`, `That code is invalid or expired.`, `Email sign-in is not available right now.`, `Could not send the code. Try again.`. Indonesian: `atau`, `Email`, `anda@contoh.com`, `Kirim kode`, `Masukkan kode {{length}} digit yang kami kirim ke {{email}}.`, `Kirim ulang kode`, `Kirim ulang kode dalam {{seconds}} dtk`, `Gunakan email lain`, `Kode tidak valid atau sudah kedaluwarsa.`, `Masuk dengan email belum tersedia saat ini.`, `Gagal mengirim kode. Coba lagi.`.

- [ ] **Step 1: Write the failing tests** (mock `react-i18next`, `react-router` `useNavigate`, and `~/lib/auth-client` like `provider-form.test.tsx`): `otpStepReducer moves to the code step on requested`; `otpStepReducer returns to the email step on changeEmail`; `remainingSeconds rounds up and never goes below zero`; `mapOtpErrorKey maps AUTH_OTP_INVALID, AUTH_OTP_UNAVAILABLE, and anything else`; `renders the email step first` (`renderToString` contains `login.otp.emailLabel` and `login.otp.sendCode`); `en and id define the same login.otp keys`.
- [ ] **Step 2: Run** `cd apps/web && bunx vitest run app/components/OtpLoginForm.test.tsx`. Expected: FAIL.
- [ ] **Step 3: Implement** the form and add the i18n keys. In `login.tsx`, render `<OtpLoginForm />` after the GitHub button, preceded by a divider showing `t("login.otp.divider")`, and keep the `invitationHint` paragraph last.
- [ ] **Step 4: Run** `bun run test && bun run lint && bun run typecheck`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add apps/web
  git commit -m "feat(web): ✨ add email otp sign-in to login page"
  ```

### Task 12: e2e coverage

**Files:**
- Create: `e2e/features/core/otp-login.feature`
- Modify: `e2e/support/test-client.ts`, `e2e/support/world.ts`, `e2e/pages/core/login.page.ts`, `e2e/steps/core/authentication.steps.ts`, `AGENTS.md` (section 5)

**Interfaces:**
- Consumes: web copy from Task 11; existing `testClient.getSentEmails`, `resetEmails`, and the deletion call used by the `@authenticated` After hook.
- Produces:
  - `TestClient.waitForOtpCode(email: string, timeoutMs = 10_000): Promise<string>`: polls `getSentEmails()` for the newest message addressed to `email` and returns the first match of `\b\d{6,10}\b` in `TextBody`; throws on timeout.
  - `LoginPage`: `emailInput` (label `Email`), `sendCodeButton` (button `Send code`), `codeInput` (the OTP input group), `resendButton` (button starting with `Resend code`), `invalidCodeMessage` (text `That code is invalid or expired.`), `fillCode(code: string)`.
  - `OmedWorld.otpEmail?: string`

- [ ] **Step 1: Write the feature** with four scenarios tagged `@otp`, using the existing steps `I open "<path>"` and `I am on "<path>"`: a new user signs in and lands on `/`; an existing user (created first through the test client) signs in; a wrong code shows the invalid-code message and stays on `/login`; after requesting a code the resend button is disabled while the countdown runs.
- [ ] **Step 2: Add steps** in `authentication.steps.ts`: `the mailbox is empty` (calls `resetEmails`), `I request a sign-in code for a new email`, `I request a sign-in code for the existing user`, `I enter the code from the email`, `I enter the code "{string}"`, `I see the invalid code message`, `the resend button is disabled`. Emails are unique per run (`otp-<Date.now()>@example.com`) and stored in `this.otpEmail`. Add an `After({ tags: "@otp" })` hook that removes the user for `otpEmail` using the same deletion helper as the `@authenticated` hook.
- [ ] **Step 3: Run to see it fail** `bunx playwright install chromium` once, then `bun run e2e -- --tags @otp`. Expected: FAIL on missing steps or page members before they exist.
- [ ] **Step 4: Implement** the helper, page members, and world field; run again. Expected: PASS (the e2e servers start with `OMED_ENV=test` and the memory mailer, so OTP is enabled).
- [ ] **Step 5: Reword AGENTS.md section 5.** Replace the sentence "Sign-in is social only, so the hook creates the user and session with the API test endpoints (`support/test-client.ts`), and removes the user afterwards." with "The hook creates the user and session with the API test endpoints (`support/test-client.ts`) instead of going through a sign-in method, and removes the user afterwards."
- [ ] **Step 6: Run the whole suite** `bun run e2e`. Expected: PASS.
- [ ] **Step 7: Commit**
  ```bash
  git add e2e AGENTS.md
  git commit -m "test(e2e): ✅ cover passwordless otp login"
  ```
