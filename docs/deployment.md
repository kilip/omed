# Deployment Guide

## Docker

All container images are built from the repository root (context `.`):

```bash
docker build -f apps/api/Dockerfile -t omed-api .
docker build -f apps/web/Dockerfile -t omed-web .
```

The `.github/workflows/build.yml` workflow publishes images to `ghcr.io/<owner>/omed-<app>` (for example `ghcr.io/kilip/omed-api`), tagging pushes to `main` with `nightly`/`sha-*` and releases with semver + `latest`.

### Web Runtime Configuration

`VITE_*` variables are baked in at build time, but the `omed-web` image can override them at container startup via `OMED_WEB_*` environment variables (`OMED_WEB_API_URL`, `OMED_WEB_PUBLIC_URL`, `OMED_WEB_ATTACHMENT_MAX_SIZE_MB`, `OMED_WEB_ATTACHMENT_ALLOWED_TYPES`, `OMED_WEB_CMS_MEDIA_MAX_SIZE_MB`, `OMED_WEB_CMS_MEDIA_ALLOWED_TYPES`).

`apps/web/docker-entrypoint.sh` renders those values into `/config.js`, which the browser loads before falling back to build-time defaults:

```bash
docker run -p 3000:3000 -e OMED_WEB_API_URL=https://api.example.com omed-web
```

### Running Migrations in Production

The `omed-api` image embeds `apps/api/migrations` in `/api` and also includes `/atlas`, `/migrations`, and `/atlas.hcl`, supporting two deployment patterns:

1. **Dedicated Migration Init-Container / Job (recommended for multi-replica deployments):**
   Run the `omed-api` image with `["/api", "migrate"]` (or `["/atlas", "migrate", "apply", "--config", "file:///atlas.hcl", "--env", "local"]`) before starting the API service.
2. **Single-Container Auto-Migrate on Startup:**
   Set `OMED_DB_MIGRATE_ON_START=true` on the `omed-api` container to automatically apply pending versioned migrations (guarded by a PostgreSQL advisory lock) before listening on port `8001`.

## Kubernetes (Helm)

The Helm chart in [`helm`](../helm) deploys `api` and `web`, runs database migrations as a pre-install/pre-upgrade hook Job, and exposes both services through an Ingress. PostgreSQL, S3, and SMTP are managed externally.

```bash
helm install omed oci://ghcr.io/kilip/charts/omed \
  --set secrets.existingSecret=omed-secrets \
  --set ingress.web.host=app.example.com \
  --set ingress.api.host=api.example.com
```

See the [Helm chart README](../helm/README.md) for full details on secrets, values, and URL derivation.
