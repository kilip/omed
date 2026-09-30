# Omed dev container

- Postgres 17 (`db`), 1 database `omed`, schema: `auth`, `finance`, `blog`
- Adminer di :8080 (server `db`, user/pass/db: `omed`)
- Toolchain: Bun, Node 24, Go 1.25 (air, ent, swag, mockgen, staticcheck, gopls, dlv), Rust stable (clippy, rustfmt, sqlx-cli), Playwright Chromium

## Ports
| Port | Service |
|------|---------|
| 3001 | dash |
| 9001 | auth |
| 9002 | finance |
| 9003 | blog |
| 5432 | postgres |
| 8080 | adminer |

## Catatan
- `AUTH_BASE_PATH` sengaja tidak di-set di compose; taruh di `.env` masing-masing app.
- Di Codespaces (browser), `localhost:3001` yang di-hardcode di `LoginForm.tsx` dan CORS `apps/auth/src/server.ts`
  harus diganti ke URL forwarded port (mis. via env).
