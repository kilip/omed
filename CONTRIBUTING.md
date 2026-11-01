# Contributing to Omed

Thank you for your interest in contributing to Omed! This guide explains our workflow, conventions, and requirements for submitting changes.

## Development Workflow

1. **Create a Branch:** Always branch off `main` with a descriptive name (e.g., `feat/new-tool`, `fix/login-redirect`, `docs/api-guide`).
2. **Implement Changes:** Keep pull requests focused on one logical change. Avoid mixing refactors with functional changes.
3. **Follow Commit Conventions:** Commits must follow [Conventional Commits](https://www.conventionalcommits.org) with an emoji prefix and valid scope (see below).
4. **Run Checks Locally:** Ensure Biome formatting/linting and tests pass before pushing:
   ```bash
   bun run lint
   bun run format
   bun run test
   cd apps/api && go test ./...
   ```
5. **Open a Pull Request:** Push your branch and open a PR against `main`. All PRs run CI checks (linting, Go tests, unit tests, and Playwright E2E) that must pass before merging.

## Commit Message Convention

Commits follow [Conventional Commits](https://www.conventionalcommits.org) validated by Commitlint, with one addition: **every commit subject must be prefixed with an emoji**:

```
<type>(<scope>): <emoji> <subject>
```

### Examples

```
feat(api): ✨ add refresh token rotation
fix(web): 🐛 handle empty workspace list
chore(deps): 🔧 bump biome
```

### Rules

- **Emoji placement:** Directly after the colon and space (`type(scope): <emoji> subject`) so tooling parses type and scope properly.
- **Scope is required.** Allowed scopes: `auth`, `web`, `api`, `agent`, `e2e`, `deps`, `ci`, `release`, `tools`, `deploy`, `docs`.
- **Subject:** Imperative mood, lowercase after the emoji, no trailing period, max ~72 characters including the emoji.
- **Emoji mapping:**
  | Type | Emoji | Description |
  | --- | --- | --- |
  | `feat` | ✨ | New features |
  | `fix` | 🐛 | Bug fixes |
  | `refactor` | ♻️ | Code refactoring without behavior changes |
  | `perf` | ⚡ | Performance improvements |
  | `test` | ✅ | Adding or correcting tests |
  | `docs` | 📝 | Documentation additions or updates |
  | `build` | 📦 | Build system, dependency updates (e.g. `build(deps)`) |
  | `ci` | 👷 | CI/CD configuration and scripts |
  | `chore` | 🔧 | Tooling, maintenance, repo config (e.g. `chore(deps)`) |
  | `revert` | ⏪ | Reverting a previous commit |

## Coding & Style Guidelines

- **TypeScript / Frontend:** Follow `.editorconfig` and use Biome (`bun run lint`, `bun run format`). Dashboard UI conventions and design tokens are documented in [BRANDING.md](./BRANDING.md).
- **Go / Backend:** Follow standard Go formatting (`gofmt`). Architecture follows layered boundaries under `apps/api/internal/<module>/` (`controller -> service -> repository`).
- **Database Migrations:** Schema changes under `apps/api/ent/schema/` require running `go generate ./ent` and generating versioned Atlas migrations. See [docs/development.md](./docs/development.md).
- **Secrets:** Never commit secrets or `.env` files. Use `.env.example` to document environment variables.
- **AI Coding Agents:** Refer to [AGENTS.md](./AGENTS.md) for full architecture and repo guidelines.

## Releases

Releases and `CHANGELOG.md` are automated via [release-please](https://github.com/googleapis/release-please) based on Conventional Commits on `main`.
