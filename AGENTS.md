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

Commits follow [Conventional Commits](https://www.conventionalcommits.org), validated by commitlint, with one addition: **every commit subject ends with an emoji shortcode**.

```
<type>(<scope>): <subject> <emoji>
```

Examples:

```
feat(api): add auth :rocket:
fix(dash): handle empty workspace list :bug:
chore(deps): bump biome :wrench:
```

Rules:

- The scope is optional. Allowed scopes (see `commitlint.config.ts`): `auth`, `dash`, `api`, `e2e`, `deps`, `ci`, `release`, `tools`, `docs`.
- The subject is lowercase, imperative, and has no trailing period. The emoji shortcode comes last, separated by a space.
- Use shortcodes (`:rocket:`), not the Unicode character.
- `deps` is a scope, not a type: use `chore(deps): ...`.
- Keep the header within 100 characters, emoji included.

Emoji by type:

| Type | Emoji |
| --- | --- |
| `feat` | `:rocket:` |
| `fix` | `:bug:` |
| `docs` | `:memo:` |
| `refactor` | `:recycle:` |
| `perf` | `:zap:` |
| `test` | `:white_check_mark:` |
| `style` | `:art:` |
| `build` | `:package:` |
| `ci` | `:construction_worker:` |
| `revert` | `:rewind:` |
| `chore` | `:wrench:` |

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

Module names match the dashboard modules: `sys`, `user`, `fin`, `web`.

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

### Module structure (`apps/dash`)

Each dashboard module is self-contained under `app/modules/{module_name}/`:

```
apps/dash/app/modules/{module_name}/
├── route.ts       # Route definitions for the module
├── hooks/         # Module-specific React hooks
└── ...            # Other module-local code (components, utils, types)
```

- Module names match the Go modules: `sys`, `user`, `fin`, `web`.
- A module registers itself through its own `route.ts`. There is no central registry file: adding a module means adding `app/modules/{module_name}/route.ts`.
- API types come from `@omed/openapi` and are consumed through `openapi-fetch`. Do not hand-write types for API responses.
- Style with the design tokens and the Ant Design theme described in `BRANDING.md`. Do not hardcode hex values in components.
- Unit tests sit next to the file under test, the same convention as Go.
