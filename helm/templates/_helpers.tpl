{{/*
Chart name, truncated to the DNS label limit.
*/}}
{{- define "omed.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified release name used as the resource name prefix.
*/}}
{{- define "omed.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "omed.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels. Call with (dict "ctx" $ "component" "api").
*/}}
{{- define "omed.labels" -}}
helm.sh/chart: {{ include "omed.chart" .ctx }}
{{ include "omed.selectorLabels" . }}
app.kubernetes.io/version: {{ .ctx.Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .ctx.Release.Service }}
app.kubernetes.io/part-of: omed
{{- end }}

{{/*
Selector labels. Call with (dict "ctx" $ "component" "api").
*/}}
{{- define "omed.selectorLabels" -}}
app.kubernetes.io/name: {{ include "omed.name" .ctx }}
app.kubernetes.io/instance: {{ .ctx.Release.Name }}
{{- with .component }}
app.kubernetes.io/component: {{ . }}
{{- end }}
{{- end }}

{{- define "omed.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "omed.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Container image reference. Call with (dict "ctx" $ "image" .Values.api.image).
*/}}
{{- define "omed.image" -}}
{{- $tag := default .ctx.Chart.AppVersion .image.tag }}
{{- if .ctx.Values.global.imageRegistry }}
{{- printf "%s/%s:%s" .ctx.Values.global.imageRegistry .image.repository $tag }}
{{- else }}
{{- printf "%s:%s" .image.repository $tag }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding OMED_* secret keys.
*/}}
{{- define "omed.secretName" -}}
{{- default (include "omed.fullname" .) .Values.secrets.existingSecret }}
{{- end }}

{{/*
Build a public URL for an ingress host: https:// if the host is listed in
ingress.tls, http:// otherwise. Call with (dict "ctx" $ "host" "...").
*/}}
{{- define "omed.hostUrl" -}}
{{- $scheme := "http" }}
{{- range .ctx.Values.ingress.tls }}
{{- if has $.host .hosts }}
{{- $scheme = "https" }}
{{- end }}
{{- end }}
{{- printf "%s://%s" $scheme .host }}
{{- end }}

{{/*
Public API URL: explicit api.apiUrl, else derived from the ingress api host.
*/}}
{{- define "omed.apiUrl" -}}
{{- if .Values.api.apiUrl }}
{{- .Values.api.apiUrl }}
{{- else }}
{{- include "omed.hostUrl" (dict "ctx" . "host" .Values.ingress.api.host) }}
{{- end }}
{{- end }}

{{/*
Public web URL: explicit api.webUrl, else derived from the ingress web host.
*/}}
{{- define "omed.webUrl" -}}
{{- if .Values.api.webUrl }}
{{- .Values.api.webUrl }}
{{- else }}
{{- include "omed.hostUrl" (dict "ctx" . "host" .Values.ingress.web.host) }}
{{- end }}
{{- end }}

{{- define "omed.trustedOrigins" -}}
{{- if .Values.api.trustedOrigins }}
{{- join "," .Values.api.trustedOrigins }}
{{- else }}
{{- printf "%s,%s" (include "omed.webUrl" .) (include "omed.apiUrl" .) }}
{{- end }}
{{- end }}

{{/*
Fail early on invalid secret values when the chart manages the Secret.
*/}}
{{- define "omed.validateSecrets" -}}
{{- if not .Values.secrets.existingSecret }}
{{- if not .Values.secrets.dbUrl }}
{{- fail "secrets.dbUrl is required when secrets.existingSecret is empty" }}
{{- end }}
{{- if lt (len .Values.secrets.authSecret) 32 }}
{{- fail "secrets.authSecret must be at least 32 characters when secrets.existingSecret is empty" }}
{{- end }}
{{- if not .Values.secrets.secretKey }}
{{- fail "secrets.secretKey is required when secrets.existingSecret is empty" }}
{{- end }}
{{- if lt (len .Values.secrets.secretKey) 32 }}
{{- fail "secrets.secretKey must be at least 32 characters when secrets.existingSecret is empty" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Non-secret env vars shared by the api Deployment and the migration Job.
*/}}
{{- define "omed.api.env" -}}
{{- $c := .Values.api.config -}}
- name: OMED_ENV
  value: {{ $c.env | quote }}
- name: OMED_API_PORT
  value: {{ .Values.api.port | quote }}
- name: OMED_API_URL
  value: {{ include "omed.apiUrl" . | quote }}
- name: OMED_WEB_URL
  value: {{ include "omed.webUrl" . | quote }}
- name: OMED_TRUSTED_ORIGINS
  value: {{ include "omed.trustedOrigins" . | quote }}
- name: OMED_SESSION_COOKIE_NAME
  value: {{ $c.sessionCookieName | quote }}
- name: OMED_DB_MIGRATE_ON_START
  value: "false"
- name: OMED_DB_AUTO_MIGRATE
  value: "false"
- name: OMED_MAILER_DRIVER
  value: {{ $c.mailer.driver | quote }}
- name: OMED_MAILER_HOST
  value: {{ $c.mailer.host | quote }}
- name: OMED_MAILER_PORT
  value: {{ $c.mailer.port | quote }}
- name: OMED_MAILER_FROM_ADDRESS
  value: {{ $c.mailer.fromAddress | quote }}
- name: OMED_MAILER_FROM_NAME
  value: {{ $c.mailer.fromName | quote }}
- name: OMED_MAILER_INSECURE_TLS
  value: {{ $c.mailer.insecureTLS | quote }}
- name: OMED_S3_DRIVER
  value: {{ $c.s3.driver | quote }}
- name: OMED_S3_ENDPOINT
  value: {{ $c.s3.endpoint | quote }}
- name: OMED_S3_REGION
  value: {{ $c.s3.region | quote }}
- name: OMED_S3_FORCE_PATH_STYLE
  value: {{ $c.s3.forcePathStyle | quote }}
- name: OMED_S3_BUCKET_PREFIX
  value: {{ $c.s3.bucketPrefix | quote }}
- name: OMED_ATTACHMENT_MAX_SIZE_MB
  value: {{ $c.attachment.maxSizeMB | quote }}
- name: OMED_ATTACHMENT_ALLOWED_TYPES
  value: {{ $c.attachment.allowedTypes | quote }}
- name: OMED_CMS_MEDIA_MAX_SIZE_MB
  value: {{ $c.cmsMedia.maxSizeMB | quote }}
- name: OMED_CMS_MEDIA_ALLOWED_TYPES
  value: {{ $c.cmsMedia.allowedTypes | quote }}
- name: OMED_RATE_LIMIT_ENABLED
  value: {{ $c.rateLimit.enabled | quote }}
- name: OMED_LLM_ALLOW_PRIVATE_URLS
  value: {{ $c.llmAllowPrivateURLs | quote }}
{{- with .Values.api.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}
