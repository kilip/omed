#!/bin/sh
# Render runtime config for the web SPA from OMED_WEB_* env vars, then exec
# the given command. The generated file is served as /config.js (symlinked
# from build/client) and read by app/lib/env.ts via window.__OMED_CONFIG__.
set -eu

config_dir="${OMED_WEB_CONFIG_DIR:-/tmp/omed}"
config_file="${config_dir}/config.js"

# Escape a value for use inside a double-quoted JS string literal.
escape() {
  printf '%s' "$1" | tr -d '\r\n' | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e 's/</\\u003c/g'
}

entries=""
add() {
  key="$1"
  value="$2"
  if [ -z "$value" ]; then
    return
  fi
  if [ -n "$entries" ]; then
    entries="${entries},"
  fi
  entries="${entries}\"${key}\":\"$(escape "$value")\""
}

add apiUrl "${OMED_WEB_API_URL:-}"
add publicUrl "${OMED_WEB_PUBLIC_URL:-}"
add attachmentMaxSizeMB "${OMED_WEB_ATTACHMENT_MAX_SIZE_MB:-}"
add attachmentAllowedTypes "${OMED_WEB_ATTACHMENT_ALLOWED_TYPES:-}"
add cmsMediaMaxSizeMB "${OMED_WEB_CMS_MEDIA_MAX_SIZE_MB:-}"
add cmsMediaAllowedTypes "${OMED_WEB_CMS_MEDIA_ALLOWED_TYPES:-}"

mkdir -p "$config_dir"
printf 'window.__OMED_CONFIG__ = {%s};\n' "$entries" >"$config_file"

exec "$@"
