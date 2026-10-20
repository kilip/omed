import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);

function loadEnvFile(filePath: string) {
  if (!existsSync(filePath)) return;
  const content = readFileSync(filePath, "utf-8");
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const eqIdx = line.indexOf("=");
    if (eqIdx === -1) continue;
    const key = line.slice(0, eqIdx).trim();
    let val = line.slice(eqIdx + 1).trim();
    if (
      (val.startsWith('"') && val.endsWith('"')) ||
      (val.startsWith("'") && val.endsWith("'"))
    ) {
      val = val.slice(1, -1);
    }
    if (process.env[key] === undefined || process.env[key] === "") {
      process.env[key] = val;
    }
  }
}

// Load envs: apps/api/.env is source of truth for db & api
loadEnvFile(path.join(repoRoot, "apps/api/.env"));
loadEnvFile(path.join(repoRoot, "e2e/.env"));

const env = (name: string, fallback: string): string => {
  const value = process.env[name];
  return value && value.length > 0 ? value : fallback;
};

const flag = (name: string, fallback: boolean): boolean => {
  const value = process.env[name];
  if (value === undefined || value === "") return fallback;
  return value === "true" || value === "1";
};

/**
 * Defaults mirror the `.env.example` files and `.devcontainer/compose.yaml`.
 * Every value can be overridden through the same `OMED_*` variables the apps use.
 */
const apiUrl = env("OMED_API_URL", "http://localhost:8001");

export const config = {
  apiUrl,
  authUrl: apiUrl,
  webUrl: env("OMED_WEB_URL", "http://localhost:3000"),
  authPath: env("OMED_AUTH_PATH", "/auth"),
  authSecret: env(
    "OMED_AUTH_SECRET",
    "6171ce314c622f9a4cdb542c810d40b91edde2f659bcd5224c6aeee7d7ea4925",
  ),
  dbUrl: env(
    "OMED_DB_URL",
    "postgres://omed:omed@localhost:5432/omed?sslmode=disable",
  ),
  s3Driver: env("OMED_S3_DRIVER", "s3"),
  s3Endpoint: env("OMED_S3_ENDPOINT", "http://localhost:9000"),
  s3Region: env("OMED_S3_REGION", "us-east-1"),
  s3AccessKey: env("OMED_S3_ACCESS_KEY", "rustfsadmin"),
  s3SecretKey: env("OMED_S3_SECRET_KEY", "rustfsadmin"),
  s3ForcePathStyle: env("OMED_S3_FORCE_PATH_STYLE", "true"),
  attachmentMaxSizeMB: env("OMED_ATTACHMENT_MAX_SIZE_MB", "10"),
  attachmentAllowedTypes: env(
    "OMED_ATTACHMENT_ALLOWED_TYPES",
    ".pdf,.png,.jpg,.jpeg,.webp,.csv,.xls,.xlsx",
  ),
  trustedOrigins: env(
    "OMED_TRUSTED_ORIGINS",
    "http://localhost:3000,http://localhost:8001",
  ),
  autoStart: flag("E2E_AUTO_START", true),
  headed: flag("HEADED", false),
  startTimeoutMs: Number(env("E2E_START_TIMEOUT_MS", "120000")),
};

export type E2EConfig = typeof config;
