# omed

A polyglot monorepo powered by [Bun](https://bun.sh) workspaces and [Turborepo](https://turbo.build), containing an authentication service, a dashboard web app, and a finance API.

## Repository layout

```
.
├── apps/
│   ├── auth/        # @omed/auth    – Auth service (Elysia + Better Auth), runs on Bun or Cloudflare Workers
│   ├── dash/        # @omed/dash    – Dashboard (React Router v8, Ant Design, Tailwind, i18n en/id)
│   └── finance/     # @omed/finance – Finance API (Go, Fiber v3, Ent, Casbin)
├── packages/
│   ├── better-auth/ # @omed/better-auth – Shared Better Auth config, Drizzle schema & tests
│   └── tsconfig/    # @omed/tsconfig    – Shared TypeScript configs
└── .devcontainer/   # Dev container with Postgres, Redis and RustFS (S3)
```

| App       | Stack                                           | Dev port |
| --------- | ----------------------------------------------- | -------- |
| `dash`    | React Router, Vite, Ant Design, Tailwind CSS    | `3001`   |
| `auth`    | Elysia, Better Auth, Drizzle, Wrangler          | `8001`   |
| `finance` | Go, Fiber, Ent, Casbin (JWT verified via JWKS)  | `8002`   |

## Prerequisites

- [Bun](https://bun.sh) `1.4.x`
- [Go](https://go.dev) `1.27+` and [air](https://github.com/air-verse/air) (for `finance` hot reload)
- PostgreSQL 18 (Redis and an S3-compatible store are provided in the dev container)

The easiest way to get everything is to open the repo in the **Dev Container** (`.devcontainer/`), which starts:

| Service    | Port          | Notes                                              |
| ---------- | ------------- | -------------------------------------------------- |
| PostgreSQL | `5432`        | user/pass/db: `omed`; schemas `auth`, `finance`, `blog` |
| Redis      | `6379`        |                                                    |
| RustFS     | `9000`/`9001` | S3 API / web console (`rustfsadmin` / `rustfsadmin`) |

## Getting started

```bash
# install dependencies
bun install

# generate the Better Auth schema and push it to the database
bun run auth:schema
bun run db:push

# start all apps in dev mode
bun run dev
```

Run a single app with Turbo filters, e.g. `bunx turbo run dev --filter=@omed/dash`.

## Scripts

| Command               | Description                                  |
| --------------------- | -------------------------------------------- |
| `bun run dev`         | Run all apps in development mode             |
| `bun run build`       | Build all apps                               |
| `bun run test`        | Run tests (Vitest)                           |
| `bun run typecheck`   | Type-check all TypeScript packages           |
| `bun run check`       | Lint & format check with Biome               |
| `bun run check:write` | Apply Biome fixes                            |
| `bun run format`      | Format with Biome                            |
| `bun run auth:schema` | Generate Better Auth Drizzle schema          |
| `bun run db:push`     | Push the auth schema to the database         |
| `bun run clean`       | Clean build outputs and `node_modules`       |

## Configuration

### auth (`packages/better-auth/src/authEnv.ts`)

| Variable                 | Default                                       |
| ------------------------ | --------------------------------------------- |
| `AUTH_SECRET`            | **required**                                  |
| `AUTH_URL`               | `http://localhost:8001`                       |
| `AUTH_PORT`              | `8001` (Bun runtime only)                     |
| `AUTH_DB_DRIVER`         | `node` (`node` \| `neon` \| `neon-http`)      |
| `AUTH_DB_URL`            | `postgresql://omed:omed@db:5432/omed`         |
| `AUTH_TRUSTED_ORIGINS`   | `http://localhost:3001 http://localhost:8001` |
| `AUTH_GOOGLE_ID` / `AUTH_GOOGLE_SECRET` | optional – Google OAuth        |
| `AUTH_GITHUB_ID` / `AUTH_GITHUB_SECRET` | optional – GitHub OAuth        |

### dash (`apps/dash/app/lib/appEnv.ts`)

| Variable         | Default                 |
| ---------------- | ----------------------- |
| `VITE_AUTH_URL`  | `http://localhost:8001` |
| `VITE_AUTH_PATH` | `""`                    |

### finance (`apps/finance/internal/config/config.go`)

| Variable              | Default                                       |
| --------------------- | --------------------------------------------- |
| `FIN_PORT`            | `8002`                                        |
| `FIN_DB_URL`          | `postgresql://omed:omed@localhost:5432/omed`  |
| `FIN_TRUSTED_ORIGINS` | `http://localhost:3001 http://localhost:8002` |
| `AUTH_JWKS_URL`       | `http://localhost:8001/jwks`                  |

## Deployment

- **Docker** – each app has its own `Dockerfile` (built from the repo root). The `docker` workflow publishes images to `ghcr.io/<owner>/<repo>/{auth,dash,finance}`: `nightly` / `sha-*` on pushes to `main`, and semver + `latest` tags on releases.
- **Cloudflare Workers** – `auth` can be deployed with `bun run --filter @omed/auth deploy` (see `apps/auth/wrangler.toml`).
- **Releases** are managed by [release-please](https://github.com/googleapis/release-please).

## Contributing

- Commits must follow [Conventional Commits](https://www.conventionalcommits.org) (enforced by commitlint via Husky), e.g. `feat(auth): ...`, `fix(dash): ...`, `chore(ci): ...`.
- Biome runs on pre-commit; run `bun run check:write` to fix issues before committing.

## License

[MIT](./LICENSE) © Anthonius

