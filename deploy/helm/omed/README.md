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

From the OCI registry on GHCR:

```bash
kubectl create secret generic omed-secrets \
  --from-literal=OMED_DB_URL='postgres://user:pass@db:5432/omed?sslmode=require' \
  --from-literal=OMED_AUTH_SECRET="$(openssl rand -hex 32)" \
  --from-literal=OMED_S3_ACCESS_KEY='s3-access-key' \
  --from-literal=OMED_S3_SECRET_KEY='s3-secret-key' \
  --from-literal=OMED_MAILER_USER='smtp-user' \
  --from-literal=OMED_MAILER_PASSWORD='smtp-password' \
  --from-literal=OMED_GOOGLE_ID='google-client-id' \
  --from-literal=OMED_GOOGLE_SECRET='google-client-secret' \
  --from-literal=OMED_GITHUB_ID='github-client-id' \
  --from-literal=OMED_GITHUB_SECRET='github-client-secret'

helm install omed oci://ghcr.io/kilip/charts/omed \
  --set secrets.existingSecret=omed-secrets \
  --set ingress.className=nginx \
  --set ingress.web.host=app.example.com \
  --set ingress.api.host=api.example.com
```

Or from local source checkout:

```bash
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
  s3AccessKey: s3-access-key
  s3SecretKey: s3-secret-key
  mailerUser: smtp-user
  mailerPassword: smtp-password
  googleClientId: google-client-id
  googleClientSecret: google-client-secret
  githubClientId: github-client-id
  githubClientSecret: github-client-secret
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

| Env Var Key | Values Key (`secrets.*`) | Required | Purpose |
| --- | --- | --- | --- |
| `OMED_DB_URL` | `dbUrl` | yes | PostgreSQL connection string |
| `OMED_AUTH_SECRET` | `authSecret` | yes | Session signing secret, at least 32 bytes in production |
| `OMED_S3_ACCESS_KEY` | `s3AccessKey` | no | S3 access key ID |
| `OMED_S3_SECRET_KEY` | `s3SecretKey` | no | S3 secret access key |
| `OMED_MAILER_USER` | `mailerUser` | no | SMTP username |
| `OMED_MAILER_PASSWORD` | `mailerPassword` | no | SMTP password |
| `OMED_GOOGLE_ID` | `googleClientId` | no | Google OAuth client ID |
| `OMED_GOOGLE_SECRET` | `googleClientSecret` | no | Google OAuth client secret |
| `OMED_GITHUB_ID` | `githubClientId` | no | GitHub OAuth client ID |
| `OMED_GITHUB_SECRET` | `githubClientSecret` | no | GitHub OAuth client secret |

### Example: Gmail SMTP Setup

To configure email delivery via Gmail SMTP:

> [!NOTE]
> Gmail requires a 16-character **App Password** (not your standard Google account password). Enable 2-Step Verification on your Google Account and generate an App Password under **Security > 2-Step Verification > App passwords**.

1. Configure SMTP in `my-values.yaml`:

   ```yaml
   api:
     config:
       mailer:
         driver: smtp
         host: smtp.gmail.com
         port: 587
         fromAddress: your-email@gmail.com
         fromName: "Omed"
         insecureTLS: false
   ```

2. Add credentials to your secret:

   - **External secret (`secrets.existingSecret`):**
     ```bash
     kubectl create secret generic omed-secrets \
       --from-literal=OMED_MAILER_USER='your-email@gmail.com' \
       --from-literal=OMED_MAILER_PASSWORD='xxxx xxxx xxxx xxxx' \
       # ... other secrets (OMED_DB_URL, OMED_AUTH_SECRET, etc.)
     ```
   - **Chart-managed secret:**
     ```yaml
     secrets:
       mailerUser: "your-email@gmail.com"
       mailerPassword: "xxxx xxxx xxxx xxxx"
     ```

### Secret Management Options

1. **External Secret (`secrets.existingSecret`):**
   Recommended for GitOps / production. Create the Kubernetes Secret beforehand using `kubectl` or an operator (e.g. External Secrets Operator, Sealed Secrets):

   ```bash
   kubectl create secret generic omed-secrets \
     --from-literal=OMED_DB_URL='postgres://omed:pass@paradedb-rw.storage.svc:5432/omed?sslmode=disable' \
     --from-literal=OMED_AUTH_SECRET="$(openssl rand -hex 32)" \
     --from-literal=OMED_S3_ACCESS_KEY='minioadmin' \
     --from-literal=OMED_S3_SECRET_KEY='minioadmin'
   ```

   Then set `secrets.existingSecret: omed-secrets` in your values. When `existingSecret` is configured, chart-managed secret values are ignored.

2. **Chart-managed Secret:**
   Define the values under `secrets.*` in your values file. The chart creates a hook Secret (`pre-install,pre-upgrade`, weight `-10`) so the migration Job can read it before the application pods start.

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
