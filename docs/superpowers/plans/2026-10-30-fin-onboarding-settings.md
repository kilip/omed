# Finance Onboarding Wizard & Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 2-step Finance Onboarding Wizard at `/fin/onboarding` (guarded by `onboarding: true` in `fin.fin_settings`), workspace-wide `decimal_style` (`dot` vs `comma`) formatting, and a reusable `/fin/settings` configuration page.

**Architecture:** Extend `FinSetting` (`fin.fin_settings`) with `onboarding` (`bool`) and `decimal_style` (`"dot" | "comma"`), expose `GET /workspaces/{wid}/fin/settings` and `PUT /workspaces/{wid}/fin/settings` in Go/Fiber, and regenerate OpenAPI types. On the frontend, extract reusable settings form components (`CurrencyFormatFields`, `FiscalPeriodFields`, `CoaTemplateFields`) shared between the 2-step `/fin/onboarding` wizard and `/fin/settings`, and guard `/fin/*` routes via a layout loader (`modules/fin/routes/layout.tsx`) that enforces `onboarding === true` and applies the active `decimalStyle` to `formatMoney`/`formatNumber`.

**Tech Stack:** Go 1.23+, Fiber v3, Ent ORM, Atlas Migrations, Swaggo/OpenAPI, React Router v7, React 19, Ant Design 5, TanStack Query, Vitest, Playwright + Cucumber E2E.

**Spec:** `docs/superpowers/specs/2026-10-30-fin-onboarding-settings-design.md`

## Global Constraints

- Follow layered architecture in `apps/api/internal/fin`: `controller -> service -> repository`.
- Ent schema `FinSetting` must retain `entsql.Schema("fin")`; generate Ent code and versioned Atlas migration in `apps/api/migrations/`.
- Regenerate OpenAPI docs (`apps/api/docs/*`) and TypeScript client types (`packages/openapi/src/api.ts`) after modifying Swaggo annotations.
- Never hardcode hex values in frontend components; use `var(--omed-*)` CSS custom properties from `BRANDING.md`.
- Never run `git commit` or `git push` unless explicitly instructed by Pak Bos.

## Review Focus

- Completing onboarding with "Start from Scratch (Empty CoA)" (0 accounts) sets `onboarding: true` and does NOT trigger a redirect loop back to `/fin/onboarding`.
- Updating `decimalStyle` (`dot` vs `comma`) via `PUT /workspaces/{wid}/fin/settings` preserves existing FX account IDs on `fin.fin_settings` (and vice-versa when `UpdateFXAccounts` is called).
- Calling `seed.Seed` (`POST /workspaces/{wid}/fin/accounts/seed`) automatically sets `onboarding: true` on `fin.fin_settings` even if called directly by existing tests.
- Invalid `decimalStyle` values outside `"dot"` and `"comma"` are rejected with HTTP 400/422 by `PUT /workspaces/{wid}/fin/settings`.
- Base currency selector in `/fin/settings` is disabled when the workspace already has posted journal entries, preventing accidental `ErrBaseCurrencyLocked` errors.

---

### Task 1: Extend `FinSetting` Ent Schema, Migration, and Seeder

**Files:**
- Modify: `apps/api/ent/schema/fin_setting.go`
- Modify: `apps/api/internal/fin/seed/seeder.go`
- Modify: `apps/api/internal/fin/seed/seeder_test.go`
- Generated: `apps/api/ent/**`, `apps/api/migrations/*`

**Interfaces:**
- Produces:
  - Ent `FinSetting` fields: `Onboarding bool` (default `false`), `DecimalStyle finsetting.DecimalStyle` (`"dot"` | `"comma"`, default `"comma"`).
  - `seed.Seed(ctx, client, workspaceID, template, lang)` sets `Onboarding(true)` on the workspace's `FinSetting` row.

- [ ] **Step 1: Add `onboarding` and `decimal_style` fields to `apps/api/ent/schema/fin_setting.go`**

```go
func (FinSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Bool("onboarding").Default(false),
		field.Enum("decimal_style").Values("dot", "comma").Default("comma"),
		field.Int64("fx_gain_account_id").Optional().Nillable(),
		field.Int64("fx_loss_account_id").Optional().Nillable(),
		field.Int64("fx_unrealized_gain_account_id").Optional().Nillable(),
		field.Int64("fx_unrealized_loss_account_id").Optional().Nillable(),
		field.Int64("fx_rounding_account_id").Optional().Nillable(),
	}
}
```

- [ ] **Step 2: Generate Ent code and Atlas migration**

Run: `cd apps/api && go generate ./ent && cd ../.. && bun run db:diff fin_onboarding_settings`
Expected: Ent files updated in `apps/api/ent/` and new migration SQL created in `apps/api/migrations/`.

- [ ] **Step 3: Write failing test in `apps/api/internal/fin/seed/seeder_test.go` asserting `Seed` sets `Onboarding == true`**

Add assertion in `TestSeed_Success` in `apps/api/internal/fin/seed/seeder_test.go`:
```go
setting, err := client.FinSetting.Query().Where(finsetting.WorkspaceIDEQ(ws.ID)).Only(ctx)
require.NoError(t, err)
assert.True(t, setting.Onboarding)
assert.Equal(t, finsetting.DecimalStyleComma, setting.DecimalStyle)
```

- [ ] **Step 4: Update `apps/api/internal/fin/seed/seeder.go` to set `SetOnboarding(true)` when creating or updating `FinSetting`**

In `seed.Seed`, ensure `FinSetting` is always created or updated with `.SetOnboarding(true)`.

- [ ] **Step 5: Run seeder tests**

Run: `cd apps/api && go test -v ./internal/fin/seed/...`
Expected: PASS

---

### Task 2: Backend Finance Settings Domain, Repository, Service, Controller & OpenAPI

**Files:**
- Modify: `apps/api/internal/fin/domain/setting.go`
- Modify: `apps/api/internal/fin/model/setting.go`
- Modify: `apps/api/internal/fin/repository/setting.go`
- Modify: `apps/api/internal/fin/service/setting.go`
- Modify: `apps/api/internal/fin/service/setting_test.go`
- Modify: `apps/api/internal/fin/controller/setting.go`
- Modify: `apps/api/internal/fin/routes.go`
- Modify: `apps/api/internal/fin/module.go`
- Modify: `apps/api/internal/fin/test/setting_test.go`
- Generated: `apps/api/docs/**`, `packages/openapi/src/api.ts`

**Interfaces:**
- Consumes: Ent `FinSetting` (`Onboarding`, `DecimalStyle`), `CurrencyRepository.GetBaseCurrency(c, workspaceID)`.
- Produces:
  - `domain.DecimalStyleDot = "dot"`, `domain.DecimalStyleComma = "comma"`
  - `domain.FinSetting` with `Onboarding bool` and `DecimalStyle string`
  - `model.FinSettingResponse` (`onboarding`, `decimalStyle`, `baseCurrency`, `fxGainAccountId`, `fxLossAccountId`, `fxUnrealizedGainAccountId`, `fxUnrealizedLossAccountId`, `fxRoundingAccountId`)
  - `model.UpdateFinSettingRequest` (`onboarding *bool`, `decimalStyle string` validated `required,oneof=dot comma`)
  - `SettingService.GetSettings(c context.Context, workspaceID int64) (model.FinSettingResponse, error)`
  - `SettingService.UpdateSettings(c context.Context, workspaceID int64, req model.UpdateFinSettingRequest) (model.FinSettingResponse, error)`
  - HTTP `GET /workspaces/{wid}/fin/settings` and `PUT /workspaces/{wid}/fin/settings`

- [ ] **Step 1: Write failing unit tests in `apps/api/internal/fin/service/setting_test.go` and integration tests in `apps/api/internal/fin/test/setting_test.go`**

Test cases in `setting_test.go`:
1. `Test_GetSettings_Default`: returns `Onboarding: false`, `DecimalStyle: "comma"`, `BaseCurrency: "IDR"` when no `fin_settings` row exists.
2. `Test_UpdateSettings_PreservesFXAccounts`: sets `DecimalStyle: "dot"`, `Onboarding: true`, then updates FX accounts via `UpdateFXAccounts`, then updates `DecimalStyle: "comma"` via `UpdateSettings` and verifies both `Onboarding == true` and FX account IDs are preserved.
3. `Test_Settings_GetAndPut` in `internal/fin/test/setting_test.go`: exercises `GET /workspaces/{wid}/fin/settings` and `PUT /workspaces/{wid}/fin/settings` (including 422 validation error on invalid `decimalStyle: "invalid"`).

- [ ] **Step 2: Run Go tests to verify failure**

Run: `cd apps/api && go test ./internal/fin/service/... ./internal/fin/test/...`
Expected: FAIL (missing `GetSettings` / `UpdateSettings` / `FinSettingResponse`).

- [ ] **Step 3: Implement domain, model, repository, service, controller, and routes**

1. Update `apps/api/internal/fin/domain/setting.go` with `Onboarding bool`, `DecimalStyle string`, `DecimalStyleDot = "dot"`, `DecimalStyleComma = "comma"`.
2. Update `apps/api/internal/fin/model/setting.go` with `FinSettingResponse` and `UpdateFinSettingRequest`.
3. Update `apps/api/internal/fin/repository/setting.go` (`Save` and `toFinSettingDomain`) to persist `Onboarding` and `DecimalStyle` (`finsetting.DecimalStyle(setting.DecimalStyle)` defaulting to `finsetting.DecimalStyleComma` if empty).
4. Update `apps/api/internal/fin/service/setting.go` (`NewSettingService` accepts optional `CurrencyRepository` via functional option `WithSettingCurrencyRepository(currencyRepo)` or parameter) and implement `GetSettings` and `UpdateSettings`. Ensure `UpdateFXAccounts` preserves existing `Onboarding` and `DecimalStyle` values on `setting`.
5. Wire `currencyRepo` into `NewSettingService` in `apps/api/internal/fin/module.go`.
6. Add `GetSettings` (`GET /workspaces/{wid}/fin/settings`) and `UpdateSettings` (`PUT /workspaces/{wid}/fin/settings`) with Swaggo annotations in `apps/api/internal/fin/controller/setting.go` and register them in `apps/api/internal/fin/routes.go`.

- [ ] **Step 4: Regenerate OpenAPI docs and TypeScript client types**

Run: `cd apps/api && swag init -g cmd/api/main.go -o docs && cd ../.. && bun --filter @omed/openapi run generate`
Expected: `apps/api/docs/swagger.json` and `packages/openapi/src/api.ts` contain `/workspaces/{wid}/fin/settings` and `model.FinSettingResponse`.

- [ ] **Step 5: Run Go unit and integration tests**

Run: `cd apps/api && go test ./...`
Expected: PASS

---

### Task 3: Workspace Decimal Style Formatting & Finance Settings Query/Guard

**Files:**
- Modify: `apps/web/app/lib/format.ts`
- Modify: `apps/web/app/lib/format.test.ts` (or create if not present)
- Modify: `apps/web/app/modules/fin/queries.ts`
- Modify: `apps/web/app/modules/fin/queries.test.ts`
- Create: `apps/web/app/modules/fin/routes/layout.tsx`
- Create: `apps/web/app/modules/fin/routes/layout.test.tsx`

**Interfaces:**
- Consumes: `GET /workspaces/{wid}/fin/settings` from `@omed/openapi`.
- Produces:
  - `DecimalStyle = "dot" | "comma"` in `apps/web/app/lib/format.ts`
  - `setActiveDecimalStyle(style: DecimalStyle | null)` and `getActiveDecimalStyle(): DecimalStyle | null` in `apps/web/app/lib/format.ts`
  - `formatMoney(amount, currency, lang, decimalStyle?)` and `formatNumber(value, lang, decimalStyle?)` in `apps/web/app/lib/format.ts`
  - `settingKeys` and `finSettingsQueryOptions(wid: number)` in `apps/web/app/modules/fin/queries.ts`
  - `FinLayout` and `clientLoader` in `apps/web/app/modules/fin/routes/layout.tsx`

- [ ] **Step 1: Write failing tests in `apps/web/app/lib/format.test.ts`, `apps/web/app/modules/fin/queries.test.ts`, and `apps/web/app/modules/fin/routes/layout.test.tsx`**

Test cases:
- `formatMoney(1234567.89, "USD", "id", "dot")` formats with dot decimal (`$1,234,567.89`).
- `formatMoney(1234567.89, "USD", "en", "comma")` formats with comma decimal (`1.234.567,89`).
- When `setActiveDecimalStyle("dot")` is active, `formatMoney(1500000, "IDR", "id")` uses dot/comma grouping (`1,500,000`) instead of `"id"` default (`1.500.000`).
- `layout.test.tsx`: `clientLoader` redirects to `/fin/onboarding` when `settings.onboarding === false`, and returns `{ wid, settings }` + calls `setActiveDecimalStyle(settings.decimalStyle)` when `settings.onboarding === true`.

- [ ] **Step 2: Run Vitest on those test files to verify failure**

Run: `bun --filter @omed/web test app/lib/format.test.ts app/modules/fin/queries.test.ts app/modules/fin/routes/layout.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement `decimalStyle` support in `apps/web/app/lib/format.ts`, `finSettingsQueryOptions` in `apps/web/app/modules/fin/queries.ts`, and `apps/web/app/modules/fin/routes/layout.tsx`**

In `apps/web/app/lib/format.ts`:
- Add `export type DecimalStyle = "dot" | "comma";`
- Track `let activeDecimalStyle: DecimalStyle | null = null;` with `setActiveDecimalStyle` and `getActiveDecimalStyle`.
- Resolve number formatting locale via:
  ```ts
  const numberLocale = (lang: string, decimalStyle?: DecimalStyle | null) => {
    const style = decimalStyle ?? activeDecimalStyle;
    if (style === "dot") return "en-US";
    if (style === "comma") return "id-ID";
    return locale(lang);
  };
  ```
- Use `numberLocale(lang, decimalStyle)` in `formatNumber` and `formatMoney`.

In `apps/web/app/modules/fin/queries.ts`:
- Add `settingKeys = { all: (wid: number) => ["fin", "workspaces", wid, "settings"] as const }` and `finSettingsQueryOptions(wid: number)` calling `api.GET("/workspaces/{wid}/fin/settings", { params: { path: { wid } } })`.

In `apps/web/app/modules/fin/routes/layout.tsx`:
- `clientLoader`: fetches `wid = await requireWorkspaceId()`, `settings = await getQueryClient().ensureQueryData(finSettingsQueryOptions(wid))`, calls `setActiveDecimalStyle((settings.decimalStyle as DecimalStyle) ?? "comma")`, and if `!settings.onboarding`, returns `redirect("/fin/onboarding")`.
- Default export renders `<Outlet />`.

- [ ] **Step 4: Run Vitest to verify tests pass**

Run: `bun --filter @omed/web test app/lib/format.test.ts app/modules/fin/queries.test.ts app/modules/fin/routes/layout.test.tsx`
Expected: PASS

---

### Task 4: Reusable Finance Settings Components & 2-Step `/fin/onboarding` Wizard

**Files:**
- Create: `apps/web/app/modules/fin/components/currency-format-fields.tsx`
- Create: `apps/web/app/modules/fin/components/fiscal-period-fields.tsx`
- Create: `apps/web/app/modules/fin/components/coa-template-fields.tsx`
- Modify: `apps/web/app/modules/fin/components/index.ts`
- Create: `apps/web/app/modules/fin/routes/onboarding.tsx`
- Create: `apps/web/app/modules/fin/routes/onboarding.test.tsx`
- Delete: `apps/web/app/modules/fin/routes/setup.tsx`
- Modify: `apps/web/app/modules/fin/routes/dashboard.tsx`
- Modify: `apps/web/app/modules/fin/routes/accounts.tsx`
- Modify: `apps/web/app/i18n/locales/en.json`
- Modify: `apps/web/app/i18n/locales/id.json`

**Interfaces:**
- Consumes: `finSettingsQueryOptions`, `currenciesQueryOptions`, `templatePreviewQueryOptions`, `formatMoney`.
- Produces:
  - `CurrencyFormatFields` (`baseCurrency`, `onBaseCurrencyChange`, `baseCurrencyDisabled?`, `decimalStyle`, `onDecimalStyleChange`, `wid`)
  - `FiscalPeriodFields` (`fiscalYear`, `onFiscalYearChange`, `periodName`, `onPeriodNameChange`, `dateRange`, `onDateRangeChange`, `description`, `onDescriptionChange`)
  - `CoaTemplateFields` (`wid`, `coaMode`: `"template" | "blank"`, `onCoaModeChange`, `template`, `onTemplateChange`, `lang`, `onLangChange`)
  - Route component `Onboarding` + `clientLoader` in `apps/web/app/modules/fin/routes/onboarding.tsx`

- [ ] **Step 1: Write failing unit tests in `apps/web/app/modules/fin/routes/onboarding.test.tsx`**

Test cases in `onboarding.test.tsx`:
1. `clientLoader` redirects to `/fin` when `finSettings.onboarding === true`.
2. `clientLoader` returns `{ wid }` when `finSettings.onboarding === false`.
3. Renders 2-step `Steps` wizard: Step 1 shows currency, decimal style (`dot`/`comma`), and fiscal period fields defaulted by user language (`en` -> `USD` + `dot`, `id` -> `IDR` + `comma`); Step 2 shows CoA template selection + preview table and option to start with blank CoA.

- [ ] **Step 2: Run Vitest on `onboarding.test.tsx` to verify failure**

Run: `bun --filter @omed/web test app/modules/fin/routes/onboarding.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement reusable components (`CurrencyFormatFields`, `FiscalPeriodFields`, `CoaTemplateFields`), `onboarding.tsx`, i18n keys, and update `/fin/setup` links**

1. Create `apps/web/app/modules/fin/components/currency-format-fields.tsx`, `fiscal-period-fields.tsx`, and `coa-template-fields.tsx`, and export them from `apps/web/app/modules/fin/components/index.ts`.
2. Create `apps/web/app/modules/fin/routes/onboarding.tsx`:
   - `clientLoader`: checks `finSettingsQueryOptions(wid)`; if `settings.onboarding` is `true`, returns `redirect("/fin")`.
   - Component state defaults from `i18n.language`:
     - `isEn = i18n.language.startsWith("en")`
     - `baseCurrency`: `isEn ? "USD" : "IDR"`
     - `decimalStyle`: `isEn ? "dot" : "comma"`
     - `coaMode`: `"template"` (`"template" | "blank"`)
     - `lang`: `isEn ? "en" : "id"`
   - Step 0 (Step 1 in UI): renders `CurrencyFormatFields` and `FiscalPeriodFields`, plus "Next" button (`t("fin.onboarding.next")`).
   - Step 1 (Step 2 in UI): renders `CoaTemplateFields`, plus "Back" (`t("fin.onboarding.back")`) and "Complete Setup" / "Apply template" (`t("fin.onboarding.finish")`) buttons.
   - On finish:
     1. Calls `PUT /workspaces/{wid}/fin/settings/base-currency` with `{ baseCurrency }` (ignoring 409 if already locked).
     2. If `coaMode === "template"`, calls `POST /workspaces/{wid}/fin/accounts/seed` with `{ template, lang }` (ignoring 409 if accounts already exist).
     3. Calls `POST /workspaces/{wid}/fin/periods` with fiscal period payload (ignoring 409 `PERIOD_OVERLAP`).
     4. Calls `PUT /workspaces/{wid}/fin/settings` with `{ decimalStyle, onboarding: true }`.
     5. Calls `setActiveDecimalStyle(decimalStyle)`, invalidates `settingKeys.all(wid)`, `accountKeys.all(wid)`, `periodKeys.all(wid)`, and navigates to `/fin`.
3. Update `/fin/setup` links in `apps/web/app/modules/fin/routes/dashboard.tsx` and `apps/web/app/modules/fin/routes/accounts.tsx` to point to `/fin/onboarding` (or `/fin/settings`), and remove `apps/web/app/modules/fin/routes/setup.tsx`.
4. Add i18n keys in `apps/web/app/i18n/locales/en.json` and `apps/web/app/i18n/locales/id.json`.

- [ ] **Step 4: Run Vitest on `onboarding.test.tsx`**

Run: `bun --filter @omed/web test app/modules/fin/routes/onboarding.test.tsx`
Expected: PASS

---

### Task 5: `/fin/settings` Page, Route Registration, and Launcher Tile

**Files:**
- Create: `apps/web/app/modules/fin/routes/settings.tsx`
- Create: `apps/web/app/modules/fin/routes/settings.test.tsx`
- Modify: `apps/web/app/modules/fin/route.ts`
- Modify: `apps/web/app/modules/fin/module.ts`
- Modify: `apps/web/app/modules/fin/routes/launcher.tsx`
- Modify: `apps/web/app/modules/fin/routes/launcher.test.tsx`

**Interfaces:**
- Consumes: `CurrencyFormatFields`, `FiscalPeriodFields`, `CoaTemplateFields`, `finSettingsQueryOptions`, `periodsQueryOptions`, `accountsQueryOptions`, `recentEntriesQueryOptions`.
- Produces:
  - Route `/fin/settings` (`apps/web/app/modules/fin/routes/settings.tsx`)
  - Updated `apps/web/app/modules/fin/route.ts` wrapping `/fin/*` in `layout("modules/fin/routes/layout.tsx", [...])` and exposing `route("onboarding", "modules/fin/routes/onboarding.tsx")` + `route("settings", "modules/fin/routes/settings.tsx")`.
  - Updated `finLauncherItems` in `launcher.tsx` and `finModule.menu` in `module.ts` with `settings`.

- [ ] **Step 1: Write failing tests in `apps/web/app/modules/fin/routes/settings.test.tsx` and update `apps/web/app/modules/fin/routes/launcher.test.tsx`**

Test cases:
- `settings.test.tsx`: renders Currency & Number Format panel (`CurrencyFormatFields`), Fiscal Periods panel with periods list and "New Period" modal (`FiscalPeriodFields`), and Chart of Accounts / FX Accounts panel; disables Base Currency selector when posted entries exist.
- `launcher.test.tsx`: verifies `/fin/settings` tile (`fin.menu.settings`) is rendered in `FinLauncher` and included in `finModule.menu`.

- [ ] **Step 2: Run Vitest on `settings.test.tsx` and `launcher.test.tsx` to verify failure**

Run: `bun --filter @omed/web test app/modules/fin/routes/settings.test.tsx app/modules/fin/routes/launcher.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement `apps/web/app/modules/fin/routes/settings.tsx` and update `route.ts`, `module.ts`, and `launcher.tsx`**

1. In `apps/web/app/modules/fin/routes/settings.tsx`:
   - `clientLoader`: ensures `finSettingsQueryOptions(wid)`, `periodsQueryOptions(wid)`, `accountsQueryOptions(wid)`, and `recentEntriesQueryOptions(wid, 1)` (with `status: "POSTED"` check).
   - Panel 1 (**Currency & Decimal Format**): reuses `CurrencyFormatFields`, disables `baseCurrency` if posted entries exist, and saves via `PUT /workspaces/{wid}/fin/settings/base-currency` (if changed) + `PUT /workspaces/{wid}/fin/settings` (`{ decimalStyle, onboarding: true }`).
   - Panel 2 (**Fiscal Periods**): displays existing periods table (name, fiscal year, start/end dates, status tag) + button to open a `RecordModal` containing `FiscalPeriodFields` to create a new fiscal period, plus link to `/fin/periods`.
   - Panel 3 (**Chart of Accounts & Foreign Exchange Accounts**): if `accounts.length === 0`, embeds `CoaTemplateFields` with a button to seed template; always renders FX Gain, FX Loss, Unrealized Gain/Loss, and Rounding `AccountSelect` fields saved via `PUT /workspaces/{wid}/fin/settings/fx-accounts`.
2. Update `apps/web/app/modules/fin/route.ts` to register `onboarding`, `layout`, and `settings`.
3. Add `settings` item to `apps/web/app/modules/fin/routes/launcher.tsx` and `apps/web/app/modules/fin/module.ts`.

- [ ] **Step 4: Run Vitest and typecheck across web**

Run: `bun run typecheck && bun --filter @omed/web test`
Expected: PASS

---

### Task 6: Update E2E Tests for `/fin/onboarding` Wizard & Redirect Guard

**Files:**
- Modify: `e2e/pages/fin/setup.page.ts`
- Modify: `e2e/steps/fin/setup.steps.ts`
- Modify: `e2e/features/fin/setup.feature`
- Modify: `e2e/features/fin/dashboard.feature`
- Modify: `e2e/features/fin/entries.feature`
- Modify: `e2e/features/fin/income-expense.feature`
- Modify: `e2e/features/fin/ledger.feature`
- Modify: `e2e/features/fin/opening-balance.feature`
- Modify: `e2e/features/fin/periods.feature`
- Modify: `e2e/features/fin/transactions.feature`
- Modify: `e2e/features/fin/trial-balance.feature`

**Interfaces:**
- Consumes: `/fin/onboarding` 2-step wizard and `/fin/*` redirect guard.
- Produces: Passing E2E suite verifying automatic redirect from `/fin` to `/fin/onboarding` when un-onboarded, 2-step wizard completion, redirect from `/fin/onboarding` to `/fin` once onboarded, and `/fin/settings` updates.

- [ ] **Step 1: Update `e2e/pages/fin/setup.page.ts`, `e2e/steps/fin/setup.steps.ts`, and `e2e/features/fin/*.feature`**

1. Update `FinSetupPage.path` in `e2e/pages/fin/setup.page.ts` to `"/fin/onboarding"`.
2. Update `clickApplyTemplate()` in `FinSetupPage` so that if the user is on Step 1 of the wizard ("Next" button visible), it clicks "Next" to advance to Step 2 and then clicks "Apply template" / "Complete setup".
3. Update `e2e/features/fin/*.feature` where `"/fin/setup"` or post-setup redirect `"/fin/accounts"` is asserted so they reflect automatic redirect to `"/fin/onboarding"` and completion redirect to `"/fin"`.
4. Add scenario in `e2e/features/fin/setup.feature` testing:
   - Opening `"/fin"` in a fresh workspace redirects to `"/fin/onboarding"`.
   - Selecting decimal style and completing the 2-step wizard lands on `"/fin"`.
   - Re-opening `"/fin/onboarding"` after completion redirects back to `"/fin"`.

- [ ] **Step 2: Run full lint, typecheck, unit tests, and E2E tests**

Run: `bun run lint && bun run typecheck && bun run test && cd apps/api && go test ./...`
Expected: PASS
