# Development Guide

## Prerequisites

- [Bun](https://bun.sh) `1.4.2` and Node.js `>= 22`
- [Go](https://go.dev) `1.27` (see `apps/api/go.mod`)
- Docker & Docker Compose (for PostgreSQL and supporting services)
- **Go tools:** [`air`](https://github.com/air-verse/air), [`swag`](https://github.com/swaggo/swag) v2, and [`atlas`](https://atlasgo.io)

> **Tip:** The easiest way to get started is to open the repository in the **Devcontainer** (or GitHub Codespaces). All tools, services, and environment variables come pre-configured, and `bun install` runs automatically.

## Local Setup

```bash
git clone https://github.com/kilip/omed.git
cd omed

bun install                            # installs dependencies and git hooks (husky)
cd apps/api && go mod tidy && cd ../.. # sync Go dependencies

bun run db:migrate                     # apply versioned multi-schema Atlas migrations
bun run dev                            # start all apps in watch mode
```

Run a single app:

```bash
bunx turbo run dev --filter=@omed/api
bunx turbo run dev --filter=@omed/web
```

Once running:
- **Web Dashboard:** <http://localhost:3000>
- **API Server:** <http://localhost:8001>

See [Configuration & Environment Variables](./configuration.md) if you are running outside the devcontainer.

## Common Commands

```bash
bun run dev          # run all apps in watch mode
bun run build        # build packages/apps that define a build script
bun run typecheck    # TypeScript typecheck
bun run lint         # Biome check (lint + format)
bun run format       # Biome auto-fix (check --write)
bun run test         # run frontend/package unit tests via turbo
mise run e2e         # run end-to-end tests (Cucumber + Playwright)
mise run e2e:local   # start postgres + rustfs from compose, then run e2e (see Testing)
bun run db:diff      # generate a versioned migration from Ent schemas
bun run db:migrate   # apply pending Atlas database migrations
cd apps/api && go mod tidy     # sync Go dependencies
cd apps/api && go test ./...   # run Go unit and integration tests
```

## Database Migrations (Ent + Atlas)

PostgreSQL tables are organized into module-scoped schemas matching the backend modules: **`core`**, **`fin`**, **`cms`**, and **`agent`**. Each entity under `apps/api/ent/schema/` declares its target schema via `entsql.Schema("<module>")`.

Versioned SQL migrations live in `apps/api/migrations/` alongside `atlas.sum` and are configured via `apps/api/atlas.hcl`:

```bash
cd apps/api && go generate ./ent                  # regenerate Ent after editing apps/api/ent/schema/*
cd apps/api && go run ./cmd/migrate <name>        # generate a new versioned migration in apps/api/migrations
bun run db:migrate                                # apply pending migrations via Atlas CLI
cd apps/api && atlas migrate status --env local   # inspect migration status
cd apps/api && atlas migrate validate --env local # validate checksums & replay on omed_dev
```

Always commit both `apps/api/ent/*` and `apps/api/migrations/*` when changing schemas.

## OpenAPI & Types

The OpenAPI v3.1 specification is generated from Swaggo annotations in `apps/api`, then converted to TypeScript types in `packages/openapi`:

```bash
bunx turbo run oa:gen   # runs swag init in apps/api
bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts
```

Commit `apps/api/docs/*` and `packages/openapi/src/api.ts` whenever controller annotations or response DTOs change.

## Testing

- **API (Go):**
  ```bash
  cd apps/api && go test ./...
  ```
  Unit tests sit next to the file under test (`*_test.go`), and endpoint integration tests live under `apps/api/internal/<module>/test/`.
- **Web (Vitest):**
  ```bash
  bun run test
  ```
- **End-to-End (Cucumber + Playwright):** run through mise so the env vars from `mise.toml` and `mise.local.toml` are loaded.
  ```bash
  mise run e2e:setup                                   # once: install the Playwright chromium browser
  mise run e2e                                         # run every feature (postgres + rustfs must already be reachable, as in the devcontainer)
  mise run e2e:local                                   # outside the devcontainer: start postgres + rustfs from compose, then run every feature
  mise run e2e:local -- --name "financial agent"       # run scenarios whose name matches the pattern
  mise run e2e:infra:down                              # stop postgres + rustfs (volumes are kept)
  ```
  `e2e:local` starts the compose services with their ports published to the host (`.devcontainer/compose.yaml` plus `.github/compose.e2e.yaml`) through `e2e:infra:up` and waits until they are healthy before running the tests. Extra arguments after `--` go to `cucumber-js`. `--name` filters by scenario name, not feature name. Feature paths are merged with the paths in `e2e/cucumber.json`, so they do not narrow the run.

  The e2e defaults in `e2e/support/config.ts` already point at `localhost`. To override them, put the variables in `mise.local.toml` (git-ignored, env vars only; tasks live in `mise.toml`):
  ```toml
  [env]
  OMED_DB_URL = "postgres://omed:omed@localhost:5432/omed?sslmode=disable"
  OMED_S3_DRIVER = "s3"
  OMED_S3_ENDPOINT = "http://localhost:9000"
  OMED_S3_ACCESS_KEY = "rustfsadmin"
  OMED_S3_SECRET_KEY = "rustfsadmin"
  ```
  The runner starts `api` and `web` automatically if they are not already running (disable with `E2E_AUTO_START=false`). Pass `HEADED=true` to watch the browser.
