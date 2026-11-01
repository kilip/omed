# Configuration & Environment Variables

In the **devcontainer**, all environment variables are preconfigured through `.devcontainer/compose.yaml`.

Outside the devcontainer, each application reads its own git-ignored `.env` file. Copy the example files before running locally:

```bash
cp apps/api/.env.example apps/api/.env
cp apps/web/.env.example apps/web/.env
```

## API Environment Variables (`apps/api`)

| Variable | Default | Description |
| --- | --- | --- |
| `OMED_ENV` | `production` | Runtime environment (`development`, `test`, `production`) |
| `OMED_API_PORT` | `8001` | HTTP server listening port |
| `OMED_API_URL` | `http://localhost:8001` | Base URL for the API service |
| `OMED_WEB_URL` | `http://localhost:3000` | Base URL for the web frontend client |
| `OMED_TRUSTED_ORIGINS` | `http://localhost:3000` | Comma-separated allowed CORS origins |
| `OMED_DB_URL` | - | PostgreSQL connection URL |
| `OMED_DB_DEV_URL` | - | Clean dev PostgreSQL database used by Atlas for diff/validation (`omed_dev`) |
| `OMED_DB_MIGRATE_ON_START` | `false` | Apply pending versioned migrations from `apps/api/migrations` on startup |
| `OMED_DB_AUTO_MIGRATE` | `false` | Run Ent `Schema.Create` automatically on startup |
| `OMED_AUTH_PATH` | `/auth` | Subpath mounted for authentication endpoints |
| `OMED_AUTH_SECRET` | - | Secret for session tokens and OAuth state signing (min 32 bytes in `production`) |
| `OMED_SESSION_COOKIE_NAME` | `omed_session` | Session cookie name |
| `OMED_GOOGLE_ID` / `OMED_GOOGLE_SECRET` | - | Optional Google OAuth2 credentials |
| `OMED_GITHUB_ID` / `OMED_GITHUB_SECRET` | - | Optional GitHub OAuth2 credentials |
| `OMED_SECRET_KEY` | - | Base64-encoded 32-byte master key for encrypting LLM provider credentials |
| `OMED_LLM_ALLOW_PRIVATE_URLS` | `false` | Allow private/loopback IP addresses for custom LLM endpoints |
| `OMED_S3_DRIVER` | `memory` | Object storage driver (`memory` or `s3`) |
| `OMED_S3_ENDPOINT` | `http://localhost:9000` | S3 / RustFS endpoint URL |
| `OMED_S3_ACCESS_KEY` / `OMED_S3_SECRET_KEY` | - | S3 credentials |
| `OMED_S3_REGION` | `us-east-1` | S3 region name |
| `OMED_S3_FORCE_PATH_STYLE` | `true` | Enable path-style S3 addressing (required for RustFS/MinIO) |
| `OMED_S3_BUCKET_PREFIX` | `omed-ws-` | Prefix prepended to workspace IDs for workspace bucket names |
| `OMED_ATTACHMENT_MAX_SIZE_MB` | `10` | Maximum allowed attachment file size in MB |
| `OMED_ATTACHMENT_ALLOWED_TYPES` | `.pdf,.png,.jpg,.jpeg,.webp,.csv,.xls,.xlsx` | Comma-separated allowed file extensions for attachments |
| `OMED_CMS_MEDIA_MAX_SIZE_MB` | `25` | Maximum allowed CMS media file size in MB |
| `OMED_CMS_MEDIA_ALLOWED_TYPES` | `.png,.jpg,.jpeg,.webp,.gif,.pdf` | Comma-separated allowed file extensions for CMS media uploads |
| `OMED_MAILER_DRIVER` | `memory` | Email transport driver (`memory` or `smtp`) |
| `OMED_MAILER_HOST` / `OMED_MAILER_PORT` | `localhost` / `1025` | SMTP server host and port |
| `OMED_MAILER_USER` / `OMED_MAILER_PASSWORD` | - | Optional SMTP authentication credentials |
| `OMED_MAILER_FROM_ADDRESS` | `no-reply@omed.local` | Default sender email address |
| `OMED_MAILER_FROM_NAME` | `Omed` | Default sender display name |
| `OMED_MAILER_INSECURE_TLS` | `false` | Allow self-signed TLS certificates for SMTP |
| `OMED_READ_TIMEOUT` | `10s` | HTTP server read timeout |
| `OMED_WRITE_TIMEOUT` | `30s` | HTTP server write timeout |
| `OMED_IDLE_TIMEOUT` | `120s` | HTTP server keep-alive idle timeout |
| `OMED_RATE_LIMIT_ENABLED` | `true` | Toggle HTTP rate limiting |
| `OMED_RATE_LIMIT_GENERAL_MAX` / `OMED_RATE_LIMIT_GENERAL_EXPIRATION` | `100` / `1m` | Rate limit window for general routes |
| `OMED_RATE_LIMIT_SENSITIVE_MAX` / `OMED_RATE_LIMIT_SENSITIVE_EXPIRATION` | `10` / `1m` | Rate limit window for sensitive routes |
| `OMED_REDIS_URL` | - | Planned (not yet used by application code) |
| `OMED_KAFKA_BROKERS` | - | Planned (not yet used by application code) |

## Web Environment Variables (`apps/web`)

Build-time `VITE_*` variables can be overridden at container startup using the corresponding `OMED_WEB_*` variables (which `apps/web/docker-entrypoint.sh` writes into `/config.js` as `window.__OMED_CONFIG__`).

| Build-Time Variable | Container Runtime Override | Default | Description |
| --- | --- | --- | --- |
| `VITE_API_URL` | `OMED_WEB_API_URL` | `http://localhost:8001` | Base URL of the API server |
| `VITE_PUBLIC_URL` | `OMED_WEB_PUBLIC_URL` | `http://localhost:3000` | Public dashboard URL |
| `VITE_ATTACHMENT_MAX_SIZE_MB` | `OMED_WEB_ATTACHMENT_MAX_SIZE_MB` | `10` | Max attachment upload size in MB |
| `VITE_ATTACHMENT_ALLOWED_TYPES` | `OMED_WEB_ATTACHMENT_ALLOWED_TYPES` | `.pdf,.png,.jpg,.jpeg,.webp,.csv,.xls,.xlsx` | Allowed attachment file extensions |
| `VITE_CMS_MEDIA_MAX_SIZE_MB` | `OMED_WEB_CMS_MEDIA_MAX_SIZE_MB` | `10` | Max CMS media upload size in MB |
| `VITE_CMS_MEDIA_ALLOWED_TYPES` | `OMED_WEB_CMS_MEDIA_ALLOWED_TYPES` | `.png,.jpg,.jpeg,.webp,.gif,.pdf` | Allowed CMS media file extensions |
