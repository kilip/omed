# AGENTS.md — Omed

Omed is a deliberately over-engineered demo monorepo (a polyglot "on steroids" stack). Prefer correctness, clear boundaries and testability over minimalism, but do not add abstractions nobody asked for.

## Repository layout

| Path | Stack | Notes |
|---|---|---|
| `apps/auth` | Bun, ElysiaJS, better-auth | Auth server. Plugins: `admin`, `organization` (teams enabled), `jwt`. Dev port `9001`. |
| `apps/dash` | React Router 8 (SPA, `ssr: false`), Ant Design 6, Tailwind 4, TanStack Query | Dashboard. Dev port `3001`. Alias `~/*` → `app/*`. |
| `apps/finance` | Go, Fiber v3, ent, Casbin | Double-entry accounting API. Dev port `9002`. |
| `apps/blog` | Rust, axum, sqlx | Planned, not in the repo yet. Follow the same auth contract as `finance`. |
| `packages/better-auth` | TypeScript, drizzle, tsdown, vitest | `@omed/better-auth`: shared auth config, DB adapter, schema, env. |
| `packages/tsconfig` | JSON | `@omed/tsconfig` base config (`customConditions: ["dev-source"]`). |

Language of code, comments, commit messages and docs: English. Some user-facing dashboard copy and test case names are in informal Bahasa Indonesia; keep the existing tone when editing those.

## Architecture: auth flow (read this before touching auth, dash or finance)

1. **Sign-up** (`packages/better-auth/src/auth.ts`): `databaseHooks.user.create.after` calls `AuthService.createPersonalWorkspace`, which creates a personal **organization** and a personal **team** ("Personal Workspace", `personal: true`) and stores the team id in `user.activeWorkspace`.
2. **"Workspace" = better-auth team.** `workspaceId` is the team id. Do not confuse it with the organization id.
3. **Session creation** (`databaseHooks.session.create.before`): `findActiveTeam` resolves the last active team (if the user is still a member), otherwise the personal team. It writes `activeTeamId`, `activeOrganizationId`, `activeWorkspaceId`, `activeWorkspaceName`, `activeWorkspaceRoles` onto the session. Roles come from the org `member.role` (split on whitespace/commas).
4. **JWT payload** (`jwt.definePayload` in `auth.options.ts`): `id`, `name`, `avatar`, `activeWorkspaceId`, `activeWorkspaceName`, `activeWorkspaceRoles`. JWKS is served at `/jwks` on the auth server. If you change the payload, update `shared.AuthenticatedUser` in `apps/finance/internal/shared/auth.go` and the test JWT builder in `apps/finance/testutil/api.go` in the same change.
5. **Dash** (`app/routes/_dash.tsx`): a client middleware calls `getSession()` then `token()`, caches the result in `shared/auth/cache.ts` (re-fetched when the JWT is within 30s of `exp`), and exposes it via `authContext` and `AuthProvider`. Unauthenticated users are redirected to `/login`. Social login: Google and GitHub.
6. **Finance** (`internal/config/bootstrap.go`), middleware order matters:
   1. `jwtware` validates the Bearer token against `JWKS_URL` (default `http://localhost:9001/jwks`).
   2. `middleware.UserInjector` checks `iss` and `aud` equal `AUTH_BASE_URL`, then stores `shared.AuthenticatedUser` in `c.Locals(shared.AUTH_USER_CONTEXT_KEY)`.
   3. `middleware.AuthSnapshotMiddleware` upserts local `finance.user` and `finance.workspace` snapshots via `AuthService.CheckSnapshot` (re-synced when older than 7 days).
7. **Tenant isolation** (`internal/config/ent.go`): `WorkspaceHook` stamps `workspace_id`, `created_by(_name)`, `updated_by(_name)` on create and scopes update/delete by workspace; `WorkspaceInterceptor` scopes queries. Both read the user from the context, so **always pass the `fiber.Ctx` (or a context derived from it) into services and repositories**. Never query ent with `context.Background()` in request paths.
8. **Authorization**: Casbin (`internal/shared/authz`), model `sub, dom, obj, act`, with `model.conf` and `policy.csv` embedded via `go:embed`. `http.RequirePermission(resource, action)` checks every role in `activeWorkspaceRoles` against `(role, workspaceId, resource, action)`. Workspace roles: `owner`, `admin`, `member`. New resource → add a constant in `authz/resource.go` **and** policy lines in `policy.csv`.

## Conventions per stack

### apps/finance (Go)

- Layering: `controller → service → repository`. Interfaces are declared in the consumer package (controller declares `XService`, service declares `XRepository`). Constructors return concrete structs (`NewAccountService(...)`).
- DTOs live in `internal/model` (`Account`, `CreateAccountRequest`, `UpdateAccountRequest`, `ListAccountRequest`). Repositories map `ent` entities to DTOs (`toAccount`); never leak ent types past the repository.
- Validation: `go-playground/validator` tags on request structs (`validate:"..."`), bound with `c.Bind().Body/Query`. Errors for invalid input return `422`, bad IDs `400`.
- Errors: define sentinel errors in `internal/shared/errors.go` and map them in `internal/config/fiber.go` (`mapError`). Repositories wrap not-found as `errors.Join(shared.ErrItemNotFound, err)`. Never leak internal error details to clients.
- Responses: always use `http.OK`, `http.Created`, `http.WithCursor` (envelope `WebResponse{data, meta}`). Delete returns `204`. Pagination is cursor-based (`encodeCursor`/`decodeCursor` in `repository/helper.go`).
- Writes go through `repository.WithTx`.
- Every controller method has swag godoc comments (`@Summary`, `@Param`, `@Security BearerAuth`, `@Failure` …). Keep them in sync with behavior.
- IDs are UUIDv7 via `shared.GenerateID()`. Money uses `shopspring/decimal` stored as `numeric(20,8)` (`decimalField` helper in `ent/schema/field.go`). Never use `float64` for money.
- ent schemas (`ent/schema`): compose `IDV7Mixin`, `AuditMixins`, `WorkspaceMixin`; annotate with `entsql.Annotation{Schema: "finance", Table: "..."}`. Files are named `sch_*.go` (schemas) and `mix_*.go` (mixins). After schema changes run `go generate ./ent` and register the new entity in `WorkspaceInterceptor` and in `SchemaConfig` (`GetEntClient`). Missing either silently breaks tenant isolation or the Postgres schema.
- Some request DTOs accept both camelCase and snake_case JSON through custom `UnmarshalJSON`. Keep that behavior when extending them.

### apps/dash (React Router 8 SPA)

- Routes use `flatRoutes()`. A route file in `app/routes/` should be a thin re-export: `export { default, meta } from "~/features/<domain>/<Page>"`. Pages live in `app/features/<domain>/`, feature-only components in `components/` or `component/` beside them, cross-feature code in `app/shared/{ui,auth,providers,contexts,hooks}`.
- Authenticated routes are prefixed `_dash.` (layout `_dash.tsx`); public ones `_auth.` (layout `_auth.tsx`).
- New page → add the route file and the menu entry in `shared/ui/menu-items.tsx` (menu keys are the URL paths). Unfinished pages render `<UnderConstruction />`.
- UI is Ant Design 6. Theme tokens live only in `shared/ui/theme.tsx`; do not hardcode brand colors in components (primary is `#2B4C7E`). Tailwind is available for layout utilities.
- Server state uses TanStack Query. Call the finance API with `openapi-fetch` and the JWT from `useOutletContext<AuthContext>()`/`authContext`; do not store tokens in `localStorage`.
- Env vars must be `VITE_`-prefixed and validated in `app/env.ts` (`@t3-oss/env-core`).
- SPA mode: no server loaders. Use `clientLoader`, `clientMiddleware` and `HydrateFallback`.
- `verbatimModuleSyntax` is on: use `import type` for types.

### apps/auth (Elysia)

- `src/server.ts` is the single app definition, shared by `src/dev.ts` (local, `server.listen(9001)`) and `src/index.ts` (Cloudflare Worker adapter, `wrangler.toml`). Do not put runtime-specific code in `server.ts`.
- All better-auth behavior (plugins, hooks, additional fields) belongs in `packages/better-auth`, not in `apps/auth`.
- Schema changes: edit `auth.options.ts`, run `bun run auth:schema` (regenerates `packages/better-auth/src/drizzle/schema/better-auth.ts`, so do not hand-edit that file), then `bun run db:push` or generate migrations with drizzle-kit.
- Env is validated in `packages/better-auth/src/config`. Add new variables there (server-side, zod).
- `testUtils()` is only enabled when `DEVELOPMENT` is true. Never enable it in production.

### apps/blog (Rust, when added)

- axum + sqlx, verify the same JWT/JWKS contract and `activeWorkspaceId` scoping as `finance`. Use compile-time-checked sqlx queries where possible.

## Finance domain rules (accounting invariants — do not weaken)

- **Entry** (journal): at least 2 postings; `sum(debit) == sum(credit)`; amounts are non-negative; each posting is either debit or credit (`debit_amount = 0 OR credit_amount = 0`, also enforced by a DB check); `entryDate` must fall inside an **open** ledger period; every posting account must be `active`.
- Entries are append-only via the API (no update/delete endpoints). Corrections go through `adjustment` entries. Entry types: `normal`, `opening_balance`, `adjustment`, `closing`, `fx_adjustment`.
- **Ledger period**: `endDate > startDate`; periods must not overlap; a new period must start `open`; status only moves `open → closed → locked`; locked periods cannot be deleted.
- **Account** (chart of accounts): `code`, `type`, `currency` are immutable after creation; `(workspace_id, code)` is unique; types are `asset|liability|equity|revenue|expense`; status `active|archived`; hierarchy via `parentId`.
- **Exchange rate**: `fromCurrency != toCurrency`, `rate > 0`, unique per `(workspace_id, fromCurrency, toCurrency, rateDate)`. `ConvertToBase` uses the latest rate on or before the date.
- Currency codes are 3 uppercase letters (default working currency is `IDR`).
- Workspace isolation: a resource from another workspace must look like it does not exist (`404`, not `403`).

## Testing and commands

### Finance (Go)

```bash
cd apps/finance
go run ./cmd/api                      # needs DATABASE_URL, auth server up (JWKS)
go generate ./...                     # ent + mockgen
go test ./internal/... -race          # unit tests (gomock, testify suites)
go test ./tests/... -p 1              # integration tests: MUST run serially
```

- Integration tests (`apps/finance/tests`) use `testutil.ApiTestSuite[T]`, in-memory SQLite (ent auto-migration) and a fake JWKS server bound to a **fixed port (`localhost:4321`)**. Never use `t.Parallel()` and always pass `-p 1`.
- Each suite registers the controllers it needs in `SetupSuite` against `testutil.GetState().Api`. Use `s.User.WorkspaceRoles` and `s.User.WorkspaceID` to switch role/tenant.
- New endpoint checklist: create/list/get/update/delete happy paths, validation (`422`), not found (`404`), invalid ID (`400`), an RBAC matrix (owner, admin, member, user, no role), and workspace isolation. Copy the pattern from `tests/account_test.go`.
- Generated mocks (`service/mocks`, `testutil/cmocks`) are produced by mockgen; regenerate instead of editing.
- Note: SQLite in tests ignores the Postgres schema names and `numeric(20,8)`, so also verify DB-specific behavior manually against Postgres when touching schema or constraints.

### Auth package and server

```bash
cd packages/better-auth
bun run test        # vitest + PGlite, reads .env.test
bun run build       # tsdown (ESM + d.ts)

cd apps/auth
bun run dev         # http://localhost:9001
```

### Dash

```bash
cd apps/dash
npm run dev         # http://localhost:3001
npm run typecheck   # react-router typegen && tsc
npm run build
```

Auth server, finance API and dash must be started together for the full login → dashboard → API flow. Google/GitHub OAuth credentials are optional env vars; providers are only enabled when both id and secret are set.

## Working rules for agents

- Make the smallest change that satisfies the request, but complete the whole vertical slice (schema → repository → service → controller → tests → swagger comments → dash route/menu when relevant).
- Do not edit generated files: `ent/` generated code, `drizzle/schema/better-auth.ts`, `mocks/`, `.react-router/`.
- Never commit secrets or `.env` files. Configuration goes through the validated env modules (`config.go`, `packages/better-auth/src/config`, `app/env.ts`).
- Run the relevant tests/typecheck before declaring work done, and state what you could not run.
- When unsure about a cross-app contract (JWT claims, response envelope, role names), check the existing implementation first and update every consumer together.
