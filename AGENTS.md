# AGENTS.md

Guidance for AI coding agents (Claude, Gemini, and others) working in this repository. This file is the single source of truth: `CLAUDE.md` and `GEMINI.md` only point here.

Read [README.md](./README.md) for architecture and [BRANDING.md](./BRANDING.md) for dashboard visual guidelines.

## 1. General rules

- Many paths in the README describe planned structure and may not exist yet. Check the filesystem before assuming a file, package, or script exists.
- The root `package.json` is the source of truth for root scripts. Do not document or call scripts that are not defined there.
- Formatting follows `.editorconfig`. Use Biome for TypeScript, JSON, and CSS (`bun run lint`, `bun run format`) and `gofmt` for Go.
- Write documentation in English, in Markdown.
- Never commit secrets. `.env` files are git-ignored; use `.env.example` for documented defaults.

### Common commands

```bash
bun run dev          # all apps, watch mode
bun run build        # build packages/apps that define a build script
bun run typecheck    # TypeScript typecheck
bun run lint         # Biome check (lint + format)
bun run format       # Biome auto-fix
bun run test         # tests via turbo
bun run e2e          # end-to-end tests
cd apps/api && go test ./...   # Go unit + integration tests
```

## 2. Commit convention

Commits follow [Conventional Commits](https://www.conventionalcommits.org), validated by commitlint, with one addition: **every commit subject prefixed with an emoji**: `type(scope): <emoji> subject`.

```
<type>(<scope>): <emoji> <subject>
```

Examples:

```
feat(api): ✨ add auth
fix(web): 🐛 handle empty workspace list
chore(deps): 🔧 bump biome
```

Rules:

- The emoji goes right after the colon, so the `type(scope):` prefix stays parseable by tooling.
- The scope is optional. Allowed scopes (see `commitlint.config.ts`): `auth`, `web`, `api`, `e2e`, `deps`, `ci`, `release`, `tools`, `docs`.
- Subject: imperative mood, lowercase (after the emoji), no trailing period, max ~72 chars including the emoji.
- Body (when needed): explain *why*, not *what*; wrap at ~72 chars.
- Breaking changes: add `!` after type/scope and a `BREAKING CHANGE:` footer. Breaking changes keep the same emoji for their type.
- `deps` is a scope, not a type: use `chore(deps): ...` or `build(deps): ...`.
- One logical change per commit. Don't mix refactors with behavior changes.
- Use exactly this mapping. Don't pick a different emoji and don't deliberate about it:
    | Type       | Emoji | Example                                        |
    | ---------- | ----- | ---------------------------------------------- |
    | `feat`     | ✨    | `feat(auth): ✨ add refresh token rotation`     |
    | `fix`      | 🐛    | `fix(api): 🐛 handle empty response body`       |
    | `refactor` | ♻️    | `refactor(db): ♻️ extract query builder`        |
    | `perf`     | ⚡    | `perf(search): ⚡ cache parsed index`           |
    | `test`     | ✅    | `test(cart): ✅ cover discount edge cases`      |
    | `docs`     | 📝    | `docs(readme): 📝 document local setup`         |
    | `build`    | 📦    | `build(deps): 📦 bump typescript to 5.6`        |
    | `ci`       | 👷    | `ci(actions): 👷 cache node_modules`            |
    | `chore`    | 🔧    | `chore(repo): 🔧 update gitignore`              |
    | `revert`   | ⏪    | `revert(auth): ⏪ revert refresh token rotation` |

## 3. Go (`apps/api`)

### Layered architecture

Every module uses the same layers, under `internal/{module_name}/`:

```
apps/api/internal/{module_name}/
├── controller/    # HTTP handlers (Fiber), Swaggo annotations, request/response DTOs
├── middleware/    # Module-specific Fiber middleware
├── service/       # Business logic and orchestration
├── repository/    # Data access (Ent)
└── test/          # Integration tests (API endpoint testing)
```

Dependency direction is strictly `controller -> service -> repository`:

- Controllers never touch Ent or repositories directly.
- Only repositories talk to the database.
- A module reaches another module through that module's service, never its repository.

### Testing

- **Unit tests** live next to the file under test: `repository/user.go` and `repository/user_test.go`.
- **Integration tests** live in the module's `test/` directory and exercise API endpoints end to end.

### API contract

Controllers carry the Swaggo annotations. After changing annotations or response models, regenerate and commit `apps/api/docs/*` and `packages/openapi/src/api.ts` (see the README, section "OpenAPI & Types").

## 4. TypeScript (React Router)

The React Router agent skill is located at [`apps/web/.agents/skills/react-router`](./apps/web/.agents/skills/react-router). Refer to it for React Router conventions and best practices.

### Module structure (`apps/web`)

Each dashboard module is self-contained under `app/modules/{module_name}/`:

```
apps/web/app/modules/{module_name}/
├── module.ts      # Module metadata for the launcher (label, icon, etc.)
├── route.ts       # Route definitions for the module
├── hooks/         # Module-specific React hooks
└── ...            # Other module-local code (components, utils, types)
```

- Module names match the Go modules: `sys`, `user`, `fin`, `cms`.
- A module registers itself through its own files: `route.ts` for routes and `module.ts` for launcher metadata. There is no central registry file: adding a module means adding `app/modules/{module_name}/` with both files.
- The launcher discovers modules with Vite's `import.meta.glob("./modules/*/module.ts")`. Do not import modules into a hand-maintained list.
- API types come from `@omed/openapi` and are consumed through `openapi-fetch`. Do not hand-write types for API responses.
- Style with the design tokens and the Ant Design theme described in `BRANDING.md`. Do not hardcode hex values in components.
- Unit tests sit next to the file under test, the same convention as Go.

## 5. End-to-end testing (`e2e`)

End-to-end tests live under `e2e/` (Cucumber + Playwright).

- Run tests with `bun run e2e`.
- The runner starts `api` and `web` automatically if they are not already running (disable with `E2E_AUTO_START=false`).
- Add `HEADED=true` to watch the browser.
- Run `bunx playwright install chromium` once beforehand.

### Structure

```
e2e/
├── features/{module}/   # Gherkin features, grouped by module
├── steps/
│   ├── common/          # Steps shared by every module
│   └── {module}/        # Step definitions of that module only
├── pages/
│   ├── base.page.ts     # Base class for page objects
│   ├── index.ts         # createPages(): page objects grouped by module
│   └── {module}/        # Page objects (POM) of that module
└── support/             # World, hooks, config, server runner, test client
```

- Group names match the dashboard modules (`sys`, `user`, `fin`, `cms`). `dashboard` is the one extra group: it covers the shell itself (login, homepage, invitations), which lives in `apps/web/app/routes` and is not a module.
- Use the Page Object Model: locators and `expect*` checks live in `pages/{module}/*.page.ts`; steps only call page objects through `this.pages.{module}`. Do not put selectors in steps.
- Put a step in `steps/common/` only when more than one module uses it.
- Tag a feature or scenario with `@authenticated` to start it signed in. Sign-in is social only, so the hook creates the user and session with the API test endpoints (`support/test-client.ts`), and removes the user afterwards.

