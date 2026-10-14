import path from "node:path";
import { fileURLToPath } from "node:url";

const env = (name: string, fallback: string): string => {
  const value = process.env[name];
  return value && value.length > 0 ? value : fallback;
};

const flag = (name: string, fallback: boolean): boolean => {
  const value = process.env[name];
  if (value === undefined || value === "") return fallback;
  return value === "true" || value === "1";
};

export const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);

/**
 * Defaults mirror the `.env.example` files and `.devcontainer/compose.yaml`.
 * Every value can be overridden through the same `OMED_*` variables the apps use.
 */
export const config = {
  authUrl: env("OMED_AUTH_URL", "http://localhost:8001"),
  apiUrl: env("OMED_API_URL", "http://localhost:8002"),
  webUrl: env("OMED_WEB_URL", "http://localhost:3001"),
  authPath: env("OMED_AUTH_PATH", "/auth"),
  authSecret: env(
    "OMED_AUTH_SECRET",
    "6171ce314c622f9a4cdb542c810d40b91edde2f659bcd5224c6aeee7d7ea4925",
  ),
  dbUrl: env(
    "OMED_DB_URL",
    "postgres://omed:omed@localhost:5432/omed?sslmode=disable",
  ),
  trustedOrigins: env(
    "OMED_TRUSTED_ORIGINS",
    "http://localhost:3001,http://localhost:8002,http://localhost:8001",
  ),
  autoStart: flag("E2E_AUTO_START", true),
  headed: flag("HEADED", false),
  startTimeoutMs: Number(env("E2E_START_TIMEOUT_MS", "120000")),
};

export type E2EConfig = typeof config;
