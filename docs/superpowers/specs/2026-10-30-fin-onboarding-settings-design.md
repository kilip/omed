# Design Spec: Finance Onboarding Wizard (`/fin/onboarding`) & Finance Settings (`/fin/settings`)

- **Date:** 2026-10-30
- **Status:** Draft / Under Review
- **Module:** `fin` (Financial Accounting)
- **Author:** Antigravity (Pair Programming with Pak Bos)

---

## 1. Intent & The Why

Previously, new workspaces entering the Finance (`fin`) module landed on the `/fin` launcher or `/fin/overview` without any guided configuration, and had to discover `/fin/setup` via a banner to seed their Chart of Accounts (CoA) and initial fiscal period. Furthermore, number/currency formatting (`1,234.56` vs `1.234,56`) was hardcoded to the UI locale (`en-US` vs `id-ID`) rather than a workspace-level financial preference, and there was no dedicated `/fin/settings` page to manage currency, decimal formatting, fiscal periods, and FX accounts after initial setup.

This design replaces `/fin/setup` with a step-by-step **Finance Onboarding Wizard (`/fin/onboarding`)** guarded by an explicit `onboarding` boolean flag in `fin.fin_settings`, and introduces a **Finance Settings (`/fin/settings`)** page for ongoing configuration.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **Extend `fin.fin_settings` Schema**:
  - Add `onboarding` (`bool`, default `false`) and `decimal_style` (`enum: "dot" | "comma"`, default `"comma"`) to `FinSetting` (`apps/api/ent/schema/fin_setting.go`).
- **Rename `/fin/setup` to `/fin/onboarding` (2-Step Wizard Style)**:
  - Multi-step wizard using Ant Design `Steps` and reusable form components shared with `/fin/settings`:
    1. **Step 1 — General & Fiscal Period**:
       - **Base Currency**: `IDR` default for `id`, `USD` default for `en`, selectable from supported ISO currencies.
       - **Decimal/Number Format**: `comma` (`1.234,56`) default for `id`, `dot` (`1,234.56`) default for `en`, with live formatted preview.
       - **Initial Fiscal Period**: Fiscal year, period name (`FY YYYY`), start/end date range, and optional description.
    2. **Step 2 — Chart of Accounts (CoA)**:
       - Template selection (`freelancer` template with language defaulted to user's UI language `id`/`en` and searchable preview table) OR option to start with an empty/blank Chart of Accounts.
- **Onboarding Redirect Guard**:
  - Visiting `/fin` (or any `/fin/*` route other than `/fin/onboarding`) when `settings.onboarding === false` automatically redirects to `/fin/onboarding`.
  - Visiting `/fin/onboarding` when `settings.onboarding === true` automatically redirects to `/fin`.
- **Workspace-Wide Decimal Style Formatting**:
  - `formatMoney` and `formatNumber` in `apps/web/app/lib/format.ts` (and `MoneyCell` / finance views) respect the workspace's configured `decimalStyle` (`"dot"` -> `en-US` separators `1,234.56`, `"comma"` -> `id-ID` separators `1.234,56`) regardless of the active UI display language.
- **Dedicated `/fin/settings` Route & Launcher Tile**:
  - Add `/fin/settings` page and a "Settings" tile in the `/fin` launcher (`apps/web/app/modules/fin/routes/launcher.tsx`).
  - Allows viewing and updating:
    - **General & Formatting**: Base Currency (calls `PUT /workspaces/{wid}/fin/settings/base-currency`, locked if posted entries exist) and Decimal Style (`dot` / `comma`).
    - **Fiscal Periods**: Quick management/navigation for fiscal periods (create new period, view open/closed status, link to `/fin/periods`).
    - **Foreign Exchange (FX) Accounts**: Configure FX Gain, FX Loss, Unrealized Gain/Loss, and Rounding accounts (`PUT /workspaces/{wid}/fin/settings/fx-accounts`).

### Non-Goals
- Cross-module generic settings table in `core` (each module manages its own settings table in its own PostgreSQL schema).
- Retroactive conversion of historical posted journal amounts when base currency changes (base currency stays locked once posted entries exist, per existing `ErrBaseCurrencyLocked` invariant).

### Anti-Goals
- Do not cause infinite redirect loops when a user completes onboarding with a blank Chart of Accounts (`onboarding: true` is persisted in `fin.fin_settings` independently of account count).
- Do not hardcode hex colors in onboarding or settings UI components; use design tokens from `BRANDING.md` (`var(--omed-*)`).

---

## 3. Constraints

- **Database & Migrations**:
  - Schema changes in `apps/api/ent/schema/fin_setting.go` must keep `entsql.Schema("fin")`.
  - Regenerate Ent (`cd apps/api && go generate ./ent`) and create a versioned Atlas migration (`bun run db:diff`).
- **API Contract**:
  - Update Swaggo annotations in `apps/api/internal/fin/controller/setting.go`, regenerate OpenAPI docs (`apps/api/docs/*`) and TypeScript types (`packages/openapi/src/api.ts`).
- **Architecture**:
  - Strictly follow `controller -> service -> repository` in `apps/api/internal/fin/`.

---

## 4. Approaches Considered

### Approach A: Shared `core.workspace_settings` Table per `(workspace_id, module)`
- *Description*: Store `onboarding` and JSON settings in `core` schema while keeping FX account FKs in `fin.fin_settings`.
- *Trade-off*: Reusable across future modules, but splits `fin` settings across two database schemas (`core` and `fin`) and adds cross-module coupling.

### Approach B: Extend Existing `fin.fin_settings` Table (Selected by Pak Bos)
- *Description*: Add `onboarding` (`bool`) and `decimal_style` (`enum: "dot", "comma"`) columns directly to `fin.fin_settings`. Each module owns its own settings table within its schema.
- *Trade-off*: Cleanest single-table model for `fin`, strongly typed in Ent and OpenAPI, zero cross-schema coupling.

### Approach C: Key-Value `core.workspace_settings` Table
- *Description*: Store each setting as a `(workspace_id, module, key, value)` row.
- *Trade-off*: Loses static typing and requires string/JSON parsing per key.

**Decision**: **Approach B**, keeping module settings self-contained per module schema.

---

## 5. Detailed Design

### 5.1 Database Schema (`apps/api/ent/schema/fin_setting.go`)

Add two fields with safe defaults for existing rows:
```go
field.Bool("onboarding").Default(false),
field.Enum("decimal_style").Values("dot", "comma").Default("comma"),
```

### 5.2 Backend Layer (`apps/api/internal/fin/`)

1. **Domain (`internal/fin/domain/setting.go`)**:
   - Add `DecimalStyleDot = "dot"` and `DecimalStyleComma = "comma"`.
   - Extend `FinSetting` struct with:
     - `Onboarding bool`
     - `DecimalStyle string` (`"dot"` | `"comma"`)
2. **DTOs (`internal/fin/model/setting.go`)**:
   - `FinSettingResponse`:
     ```go
     type FinSettingResponse struct {
         Onboarding                bool   `json:"onboarding" example:"true"`
         DecimalStyle              string `json:"decimalStyle" example:"comma"`
         BaseCurrency              string `json:"baseCurrency" example:"IDR"`
         FXGainAccountID           *int64 `json:"fxGainAccountId,omitempty"`
         FXLossAccountID           *int64 `json:"fxLossAccountId,omitempty"`
         FXUnrealizedGainAccountID *int64 `json:"fxUnrealizedGainAccountId,omitempty"`
         FXUnrealizedLossAccountID *int64 `json:"fxUnrealizedLossAccountId,omitempty"`
         FXRoundingAccountID       *int64 `json:"fxRoundingAccountId,omitempty"`
     }
     ```
   - `UpdateFinSettingRequest`:
     ```go
     type UpdateFinSettingRequest struct {
         Onboarding   *bool  `json:"onboarding" validate:"omitempty"`
         DecimalStyle string `json:"decimalStyle" validate:"required,oneof=dot comma"`
     }
     ```
3. **Repository & Service (`internal/fin/repository/setting.go`, `internal/fin/service/setting.go`)**:
   - `SettingRepository.Get`: returns `FinSetting` (if not found in DB, `SettingService.GetSettings` returns default `Onboarding: false`, `DecimalStyle: "comma"`, plus `BaseCurrency` from `currencyRepo.GetBaseCurrency`).
   - `SettingRepository.Save`: upserts `onboarding`, `decimal_style`, and FX account fields without overwriting existing fields when updating partial settings.
   - `seed.Seed` (`internal/fin/seed/seeder.go`): when seeding accounts, also sets `Onboarding: true` on `FinSetting` so seeded workspaces are marked onboarded automatically.
4. **Endpoints (`internal/fin/routes.go` & `internal/fin/controller/setting.go`)**:
   - `GET /workspaces/{wid}/fin/settings` (`GetSettings`) — returns `FinSettingResponse`.
   - `PUT /workspaces/{wid}/fin/settings` (`UpdateSettings`) — updates `decimalStyle` and optional `onboarding` flag, returns `FinSettingResponse`.
   - Existing `PUT /workspaces/{wid}/fin/settings/base-currency` and `GET|PUT /workspaces/{wid}/fin/settings/fx-accounts` remain intact.

### 5.3 Frontend Layer (`apps/web/app/modules/fin/`)

1. **Routes (`apps/web/app/modules/fin/route.ts`)**:
   - Replace `route("setup", "modules/fin/routes/setup.tsx")` with:
     - `route("onboarding", "modules/fin/routes/onboarding.tsx")`
     - `route("settings", "modules/fin/routes/settings.tsx")`
2. **Onboarding Guard & Queries (`apps/web/app/modules/fin/queries.ts`)**:
   - Add `finSettingsQueryOptions(wid)` calling `GET /workspaces/{wid}/fin/settings`.
   - Add helper `requireFinOnboarded(wid)` used in `clientLoader` of `/fin` routes: fetches `finSettingsQueryOptions(wid)` and throws `redirect("/fin/onboarding")` if `!settings.onboarding`.
   - In `/fin/onboarding` `clientLoader`: if `settings.onboarding` is already `true`, return `redirect("/fin")`.
3. **Reusable Settings Form Components (`apps/web/app/modules/fin/components/`)**:
   - Extract shared form components so `/fin/onboarding` and `/fin/settings` share the exact same UI and field state/validation logic:
     - `CurrencyFormatFields` (`currency-format-fields.tsx`): Base Currency selector (`CurrencySelect` or currency options) and Decimal Style radio cards (`dot` `1,234,567.89` vs `comma` `1.234.567,89`) with live formatted preview.
     - `FiscalPeriodFields` (`fiscal-period-fields.tsx`): Fiscal Year, Period Name (`FY YYYY`), Start/End Date Range, and Description fields.
     - `CoaTemplateFields` (`coa-template-fields.tsx`): Mode selector (**Use Standard Template (`freelancer`)** vs **Start from Scratch (Empty CoA)**), template language radio (`id`/`en`), and searchable/filterable account preview table.
4. **Onboarding Wizard (`apps/web/app/modules/fin/routes/onboarding.tsx`)**:
   - Uses Ant Design `Steps` with **2 steps**:
     - **Step 1 (Currency, Decimal Format & Initial Fiscal Period)**:
       - Renders `CurrencyFormatFields` + `FiscalPeriodFields`.
       - Default `baseCurrency`: `i18n.language.startsWith("en") ? "USD" : "IDR"`.
       - Default `decimalStyle`: `i18n.language.startsWith("en") ? "dot" : "comma"`.
       - Validates period name & date range before advancing to Step 2.
     - **Step 2 (Chart of Accounts)**:
       - Renders `CoaTemplateFields` with default `lang`: `i18n.language.startsWith("en") ? "en" : "id"`.
   - **Wizard Submission (`Finish / Complete Setup` on Step 2)**:
     1. Update base currency via `PUT /workspaces/{wid}/fin/settings/base-currency`.
     2. If template mode is selected, seed CoA via `POST /workspaces/{wid}/fin/accounts/seed`.
     3. Create initial fiscal period via `POST /workspaces/{wid}/fin/periods` (ignoring 409 `PERIOD_OVERLAP` if period already exists).
     4. Save settings via `PUT /workspaces/{wid}/fin/settings` with `{ decimalStyle, onboarding: true }`.
     5. Invalidate `fin` queries and navigate to `/fin`.
5. **Settings Page (`apps/web/app/modules/fin/routes/settings.tsx`)**:
   - Added to `finLauncherItems` in `launcher.tsx` (`key: "settings"`, `to: "/fin/settings"`, `icon: SettingOutlined`).
   - Reuses the shared form components across 3 panels:
     - **Currency & Number Format**: Renders `CurrencyFormatFields` (Base Currency disabled with tooltip if workspace has posted entries + Decimal Style) + Save button.
     - **Fiscal Periods**: Displays active/open fiscal periods table with quick "New Period" modal reusing `FiscalPeriodFields` and link to `/fin/periods`.
     - **Chart of Accounts & FX Account Mappings**: If workspace has 0 accounts, allows seeding CoA via `CoaTemplateFields` or linking to `/fin/accounts`; plus selectors for FX Realized Gain/Loss, Unrealized Gain/Loss, and Rounding accounts (`PUT /workspaces/{wid}/fin/settings/fx-accounts`).
6. **Decimal Style Integration (`apps/web/app/lib/format.ts` & `MoneyCell`)**:
   - Extend `formatMoney` and `formatNumber` to accept an optional `decimalStyle?: "dot" | "comma"` (or read active decimal style default set when `finSettings` loads), mapping `"dot"` -> `"en-US"` number formatting and `"comma"` -> `"id-ID"` number formatting while preserving ISO currency decimal places.

---

## 6. Left to the Builder

- Exact i18n translation keys in `apps/web/app/i18n/locales/en.json` and `id.json` under `fin.onboarding.*` and `fin.settings.*`.
- Updating existing unit tests (`setup` -> `onboarding`, `launcher.test.tsx`, `format.test.ts`, Go service/controller/integration tests) and E2E tests (`e2e/features/fin/*`, `e2e/pages/fin/*`, `e2e/steps/fin/*`) to match `/fin/onboarding`.
