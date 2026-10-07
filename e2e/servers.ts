import { type ChildProcess, spawn } from "node:child_process";
import path from "node:path";
import { e2eEnv } from "./env";

let authProcess: ChildProcess | null = null;
let dashProcess: ChildProcess | null = null;

async function waitForUrl(url: string, timeoutMs = 60000): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (
        res.ok ||
        res.status === 200 ||
        res.status === 404 ||
        res.status === 302
      ) {
        return;
      }
    } catch {
      // server not ready yet
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`Timeout waiting for server at ${url} after ${timeoutMs}ms`);
}

export async function startServers(): Promise<void> {
  const rootDir = path.resolve(import.meta.dirname, "..");
  const authDir = path.resolve(rootDir, "apps/auth");
  const dashDir = path.resolve(rootDir, "apps/dash");

  const cleanEnv = { ...process.env };
  delete cleanEnv.NODE_OPTIONS;

  console.log("[servers] Starting @omed/auth on :8001...");
  authProcess = spawn("bun", ["run", "src/main.ts"], {
    cwd: authDir,
    env: {
      ...cleanEnv,
      AUTH_PORT: "8001",
      AUTH_DB_URL: e2eEnv.DB_URL,
      AUTH_SECRET: e2eEnv.AUTH_SECRET,
      AUTH_URL: e2eEnv.AUTH_URL,
    },
    stdio: "pipe",
  });

  authProcess.stderr?.on("data", (data) => {
    const s = data.toString();
    if (!s.includes("deprecated")) console.error(`[auth:err] ${s.trim()}`);
  });

  console.log("[servers] Starting @omed/dash on :3001...");
  dashProcess = spawn("bun", ["run", "e2e:dev"], {
    cwd: dashDir,
    env: {
      ...cleanEnv,
      VITE_AUTH_URL: e2eEnv.AUTH_URL,
    },
    stdio: "pipe",
  });

  dashProcess.stderr?.on("data", (data) => {
    const s = data.toString();
    if (!s.includes("deprecated")) console.error(`[dash:err] ${s.trim()}`);
  });

  // Wait for both to be ready
  await Promise.all([
    waitForUrl(`${e2eEnv.AUTH_URL}/hello`, 30000),
    waitForUrl(e2eEnv.DASH_URL, 60000),
  ]);
  try {
    await fetch(`${e2eEnv.DASH_URL}/login`);
  } catch {}
  console.log("[servers] Both auth & dash servers are ready!");
}

export async function stopServers(): Promise<void> {
  console.log("[servers] Stopping servers...");
  if (authProcess) {
    authProcess.kill("SIGTERM");
    authProcess = null;
  }
  if (dashProcess) {
    dashProcess.kill("SIGTERM");
    dashProcess = null;
  }
}
