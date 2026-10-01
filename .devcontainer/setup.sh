#!/usr/bin/env bash
set -euo pipefail

cd /workspaces/omed
echo "==> Omed dev container setup"

# --- JS/TS (auth, dash, packages/*) ---
if [ -f bun.lock ] || [ -f bun.lockb ]; then
  bun install
elif [ -f package.json ]; then
  bun install
fi

# --- Go (finance) ---
if [ -f apps/finance/go.mod ]; then
  (cd apps/finance && go mod download)
fi

# --- Rust (blog) ---
if [ -f apps/blog/Cargo.toml ]; then
  (cd apps/blog && cargo fetch)
fi

# --- Playwright browsers (e2e dash) ---
if [ -d apps/dash ]; then
  (cd apps/dash && bunx playwright install chromium) || echo "playwright install skipped"
fi

echo "==> Done. Postgres: postgres://omed:omed@db:5432/omed"
