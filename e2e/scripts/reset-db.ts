import { execSync } from "node:child_process";
import path from "node:path";
import pg from "pg";
import { e2eEnv } from "../env";

const targetUrl = new URL(e2eEnv.DB_URL);
const targetDbName = targetUrl.pathname.replace(/^\//, "");

if (!targetDbName.endsWith("_e2e")) {
  console.error(
    `[reset-db] Aborting: target database "${targetDbName}" does not end with "_e2e"!`,
  );
  process.exit(1);
}

console.log(
  `[reset-db] Resetting test database "${targetDbName}" on ${targetUrl.host}...`,
);

// 1. Connect to administrative "postgres" or "omed" database to drop and recreate targetDbName
const adminUrl = new URL(e2eEnv.DB_URL);
adminUrl.pathname = "/omed";

const adminClient = new pg.Client({
  connectionString: adminUrl.toString(),
});

try {
  await adminClient.connect();
  // Terminate active connections to target db
  await adminClient.query(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '${targetDbName}'
		  AND pid <> pg_backend_pid();
	`);
  await adminClient.query(`DROP DATABASE IF EXISTS "${targetDbName}";`);
  await adminClient.query(`CREATE DATABASE "${targetDbName}";`);
  console.log(`[reset-db] Created database "${targetDbName}"`);
} finally {
  await adminClient.end();
}

// 2. Connect to the fresh target DB and create necessary schemas
const targetClient = new pg.Client({
  connectionString: e2eEnv.DB_URL,
});

try {
  await targetClient.connect();
  await targetClient.query("CREATE SCHEMA IF NOT EXISTS auth;");
  await targetClient.query("CREATE SCHEMA IF NOT EXISTS finance;");
  await targetClient.query("CREATE SCHEMA IF NOT EXISTS blog;");
  console.log(
    `[reset-db] Created schemas (auth, finance, blog) in "${targetDbName}"`,
  );
} finally {
  await targetClient.end();
}

// 3. Push schema to target database using drizzle-kit
console.log(`[reset-db] Pushing auth schema via drizzle-kit...`);
const authDir = path.resolve(import.meta.dirname, "../../apps/auth");
execSync("bun run db:push", {
  cwd: authDir,
  stdio: "inherit",
  env: {
    ...process.env,
    AUTH_DB_URL: e2eEnv.DB_URL,
  },
});

console.log(`[reset-db] Database reset completed successfully.`);
