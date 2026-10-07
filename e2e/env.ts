import { execSync } from "node:child_process";

function getPostgresHost(): string {
  // If explicit E2E_DB_URL or AUTH_DB_URL is set, use its hostname
  const explicit = process.env.E2E_DB_URL ?? process.env.AUTH_DB_URL;
  if (explicit) {
    try {
      const u = new URL(explicit);
      if (u.hostname && u.hostname !== "localhost") return u.hostname;
    } catch {}
  }

  // Try docker container IP if available on host
  try {
    const ip = execSync(
      "docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' omed_devcontainer-postgres-1",
      { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] },
    ).trim();
    if (ip) return ip;
  } catch {}

  return "localhost";
}

const defaultHost = getPostgresHost();
const base =
  process.env.AUTH_DB_URL ?? `postgresql://omed:omed@${defaultHost}:5432/omed`;
const url = new URL(process.env.E2E_DB_URL ?? base);
if (!process.env.E2E_DB_URL) {
  url.pathname = "/omed_e2e";
}

if (process.env.NODE_ENV === "test") {
  throw new Error(
    "E2E must not run with NODE_ENV=test (better-auth would switch to in-memory PGlite)",
  );
}

export const e2eEnv = {
  DB_URL: url.toString(),
  AUTH_SECRET:
    process.env.AUTH_SECRET ??
    "e2e-secret-not-for-production-00000000000000000000",
  AUTH_URL: process.env.AUTH_URL ?? "http://localhost:8001",
  DASH_URL: process.env.DASH_URL ?? "http://localhost:3001",
};

// Set env vars so imported @omed/better-auth and worker processes use omed_e2e
process.env.AUTH_DB_URL = e2eEnv.DB_URL;
process.env.AUTH_DB_DRIVER = "node";
process.env.AUTH_SECRET = e2eEnv.AUTH_SECRET;
process.env.AUTH_URL = e2eEnv.AUTH_URL;
