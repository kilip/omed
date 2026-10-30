# omed Helm chart

Deploys the Omed **API** (`ghcr.io/kilip/omed-api`) and **web** dashboard (`ghcr.io/kilip/omed-web`).

PostgreSQL, S3-compatible storage, and SMTP are **external**. The chart does not deploy them.

## What gets deployed

| Resource | Notes |
| --- | --- |
| `Deployment` + `Service` for `api` and `web` | Probes on `/healthz` and `/readyz` (api) and `/` (web) |
| `Job` `<release>-migrate` | Helm `pre-install,pre-upgrade` hook that runs `/api migrate`. Toggle with `migrations.enabled` |
| `Secret` | Only when `secrets.existingSecret` is empty. Rendered as a hook (see [Secrets](#secrets)) |
| `Ingress` | One Ingress with two host rules: `ingress.web.host` and `ingress.api.host` |
| `ServiceAccount` | Optional (`serviceAccount.create`) |
| `HorizontalPodAutoscaler`, `PodDisruptionBudget` | Optional, off by default, per component |
| `NetworkPolicy` | Optional, off by default |

## Install

With a Secret you manage yourself (recommended, e.g. via External Secrets or Sealed Secrets):

```bash
kubectl create secret generic omed-secrets \
  --from-literal=OMED_DB_URL='postgres://user:pass@db:5432/omed?sslmode=require' \
  --from-literal=OMED_AUTH_SECRET="$(openssl rand -hex 32)"

helm install omed deploy/helm/omed \
  --set secrets.existingSecret=omed-secrets \
  --set ingress.className=nginx \
  --set ingress.web.host=app.example.com \
  --set ingress.api.host=api.example.com
```

With chart-managed secrets (prefer a values file over `--set` so secrets stay out of shell history):

```yaml
# my-values.yaml
secrets:
  dbUrl: postgres://user:pass@db:5432/omed?sslmode=require
  authSecret: <at least 32 random characters>
ingress:
  className: nginx
  web: { host: app.example.com }
  api: { host: api.example.com }
  tls:
    - secretName: omed-tls
      hosts: [app.example.com, api.example.com]
```

```bash
helm install omed deploy/helm/omed -f my-values.yaml
```

## Secrets

Both containers load the Secret with `envFrom`, so its keys are the env var names:

| Key | Required | Purpose |
| --- | --- | --- |
| `OMED_DB_URL` | yes | PostgreSQL connection string |
| `OMED_AUTH_SECRET` | yes | Session signing secret, at least 32 bytes in production |
| `OMED_GOOGLE_ID`, `OMED_GOOGLE_SECRET` | no | Google OAuth |
| `OMED_GITHUB_ID`, `OMED_GITHUB_SECRET` | no | GitHub OAuth |
| `OMED_MAILER_USER`, `OMED_MAILER_PASSWORD` | no | SMTP credentials |
| `OMED_S3_ACCESS_KEY`, `OMED_S3_SECRET_KEY` | no | S3 credentials |

> **Note:** The chart-managed Secret is a Helm hook (`pre-install,pre-upgrade`, weight `-10`) so the migration Job can read it. Helm does **not** delete hook resources on `helm uninstall`. Delete it manually: `kubectl delete secret <release>-omed`. With `secrets.existingSecret` this does not apply.

## Migrations

When `migrations.enabled=true` (default), every `helm install` and `helm upgrade` first runs a Job with the api image and `args: ["migrate"]`. If it fails, the release fails and the running pods are not replaced. The api containers always run with `OMED_DB_MIGRATE_ON_START=false`.

## Public URLs

These are derived from the ingress hosts, using `https://` when the host is listed in `ingress.tls` and `http://` otherwise:

- `OMED_API_URL` comes from `api.apiUrl`, else from `ingress.api.host`.
- `OMED_WEB_URL` comes from `api.webUrl`, else from `ingress.web.host`.
- `OMED_TRUSTED_ORIGINS` comes from `api.trustedOrigins`, else `[webUrl, apiUrl]`.
- The web runtime `OMED_WEB_API_URL` / `OMED_WEB_PUBLIC_URL` come from `web.config.apiUrl` / `web.config.publicUrl`, else the URLs above.

## Web runtime config

The web image renders `/config.js` at container start from `OMED_WEB_*` env vars (`apps/web/docker-entrypoint.sh`). The file is written to `/tmp/omed`, so the container runs with `readOnlyRootFilesystem: true` and an `emptyDir` on `/tmp`.

## Values

See [`values.yaml`](./values.yaml) for every option with comments. Most-used keys:

| Key | Default | Description |
| --- | --- | --- |
| `global.imageRegistry` | `ghcr.io` | Registry prefix for both images |
| `api.image.tag` / `web.image.tag` | `""` (chart `appVersion`) | Image tags |
| `api.replicaCount` / `web.replicaCount` | `2` | Replicas when HPA is off |
| `api.config.*` | see file | Non-secret `OMED_*` settings (mailer, S3, limits) |
| `api.extraEnv` / `web.extraEnv` | `[]` | Extra env vars (`api.extraEnv` also applies to the migration Job) |
| `secrets.existingSecret` | `""` | Use an existing Secret instead of chart-managed values |
| `migrations.enabled` | `true` | Run the migration hook Job |
| `ingress.enabled` | `true` | Create the Ingress |
| `ingress.className` | `""` | IngressClass name |
| `ingress.tls` | `[]` | TLS entries (also switches derived URLs to https) |
| `api.hpa.enabled` / `web.hpa.enabled` | `false` | HorizontalPodAutoscaler |
| `api.pdb.enabled` / `web.pdb.enabled` | `false` | PodDisruptionBudget |
| `networkPolicy.enabled` | `false` | Restrict ingress to the ingress controller namespace (and web to api) |

## Development

```bash
helm lint deploy/helm/omed --strict -f deploy/helm/omed/ci/full-values.yaml
helm template omed deploy/helm/omed -f deploy/helm/omed/ci/full-values.yaml \
  | kubeconform -strict -summary -kubernetes-version 1.30.0
```

`Chart.yaml` `version` and `appVersion` are bumped by release-please. Do not edit them by hand.
