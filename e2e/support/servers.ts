import { type ChildProcess, spawn } from "node:child_process";
import { mkdirSync, openSync } from "node:fs";
import { createRequire } from "node:module";
import net from "node:net";
import path from "node:path";
import { config, repoRoot } from "./config";

const webRequire = createRequire(path.join(repoRoot, "apps/web/package.json"));
const reactRouterBin = path.join(
  path.dirname(webRequire.resolve("@react-router/dev/package.json")),
  "bin.cjs",
);

type Service = {
  name: string;
  cwd: string;
  command: string;
  args: string[];
  /** Env on top of `process.env`. */
  env: NodeJS.ProcessEnv;
  /** Public URL, used to decide whether the service is already running. */
  url: string;
  /** URL polled until it answers, once the process has been spawned. */
  readyUrl: string;
  /** Status codes that count as "ready". Defaults to any non-5xx response. */
  readyWhen?: (status: number) => boolean;
};

export const apiCoverDir = path.join(repoRoot, "coverage", "api-raw");
const logDir = path.join(repoRoot, "e2e", "reports", "logs");
const children: { service: Service; process: ChildProcess }[] = [];

const sharedEnv = (): NodeJS.ProcessEnv => ({
  OMED_ENV: "test",
  OMED_DB_URL: config.dbUrl,
  OMED_S3_DRIVER: config.s3Driver,
  OMED_S3_ENDPOINT: config.s3Endpoint,
  OMED_S3_REGION: config.s3Region,
  OMED_S3_ACCESS_KEY: config.s3AccessKey,
  OMED_S3_SECRET_KEY: config.s3SecretKey,
  OMED_S3_FORCE_PATH_STYLE: config.s3ForcePathStyle,
  OMED_ATTACHMENT_MAX_SIZE_MB: config.attachmentMaxSizeMB,
  OMED_ATTACHMENT_ALLOWED_TYPES: config.attachmentAllowedTypes,
  OMED_AUTH_PATH: config.authPath,
  OMED_API_URL: config.apiUrl,
  OMED_WEB_URL: config.webUrl,
  OMED_TRUSTED_ORIGINS: config.trustedOrigins,
  OMED_AUTH_SECRET: config.authSecret,
});

const port = (url: string): number => Number(new URL(url).port);

const services = (): Service[] => [
  {
    name: "api",
    cwd: path.join(repoRoot, "apps/api"),
    command: "go",
    args: [
      "run",
      "-cover",
      "-coverpkg=github.com/kilip/omed/api/...",
      "./cmd/api",
    ],
    env: {
      ...sharedEnv(),
      GOCOVERDIR: apiCoverDir,
    },
    url: config.apiUrl,
    // Every route sits behind the JWT firewall, so any non-5xx answer
    // (a 401 for an anonymous request) proves the server is up.
    readyUrl: `${config.apiUrl}/workspaces`,
  },
  {
    name: "web",
    cwd: path.join(repoRoot, "apps/web"),
    command: "bun",
    args: [
      "--conditions=development",
      reactRouterBin,
      "dev",
      "--port",
      String(port(config.webUrl)),
    ],
    env: {
      ...sharedEnv(),
      VITE_API_URL: config.apiUrl,
      VITE_PUBLIC_URL: config.webUrl,
      VITE_ATTACHMENT_MAX_SIZE_MB: config.attachmentMaxSizeMB,
      VITE_ATTACHMENT_ALLOWED_TYPES: config.attachmentAllowedTypes,
      VITE_COVERAGE: "true",
    },
    url: config.webUrl,
    readyUrl: `${config.webUrl}/login`,
    readyWhen: (status) => status === 200,
  },
];

function isListening(url: string): Promise<boolean> {
  const { hostname, port: p } = new URL(url);
  return new Promise((resolve) => {
    const socket = net.connect({ host: hostname, port: Number(p) });
    socket.once("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.once("error", () => resolve(false));
    socket.setTimeout(1_000, () => {
      socket.destroy();
      resolve(false);
    });
  });
}

async function isReady(service: Service): Promise<boolean> {
  try {
    const res = await fetch(service.readyUrl, { redirect: "manual" });
    return service.readyWhen ? service.readyWhen(res.status) : res.status < 500;
  } catch {
    return false;
  }
}

async function waitUntilReady(
  service: Service,
  child: ChildProcess,
): Promise<void> {
  const deadline = Date.now() + config.startTimeoutMs;
  let exited: number | null = null;
  child.once("exit", (code) => {
    exited = code ?? -1;
  });

  while (Date.now() < deadline) {
    if (exited !== null) {
      throw new Error(
        `[e2e] ${service.name} exited with code ${exited} before it was ready. ` +
          `See ${path.join(logDir, `${service.name}.log`)}`,
      );
    }
    if (await isReady(service)) return;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `[e2e] ${service.name} was not ready at ${service.readyUrl} within ` +
      `${config.startTimeoutMs}ms. See ${path.join(logDir, `${service.name}.log`)}`,
  );
}

async function start(service: Service): Promise<void> {
  if (await isListening(service.url)) {
    console.log(`[e2e] ${service.name} already running at ${service.url}`);
    return;
  }

  mkdirSync(logDir, { recursive: true });
  const log = openSync(path.join(logDir, `${service.name}.log`), "w");
  console.log(`[e2e] starting ${service.name} at ${service.url}`);

  const child = spawn(service.command, service.args, {
    cwd: service.cwd,
    env: { ...process.env, ...service.env },
    stdio: ["ignore", log, log],
    // Own process group, so the whole tree (go run -> binary) can be stopped.
    detached: true,
  });
  children.push({ service, process: child });

  await waitUntilReady(service, child);
}

/** Starts `api` and `web` unless they already answer on their port. */
export async function startServers(): Promise<void> {
  if (!config.autoStart) return;

  mkdirSync(apiCoverDir, { recursive: true });

  for (const service of services()) {
    await start(service);
  }
}

/** Stops only the processes this runner started. */
export async function stopServers(): Promise<void> {
  const stopped: Promise<void>[] = [];
  for (const { process: child } of children.splice(0).reverse()) {
    if (child.pid === undefined || child.exitCode !== null) continue;
    stopped.push(
      new Promise<void>((resolve) => {
        child.once("exit", () => resolve());
        setTimeout(resolve, 5_000);
      }),
    );
    try {
      if (process.platform === "win32") {
        spawn("taskkill", ["/pid", String(child.pid), "/T", "/F"]);
      } else {
        process.kill(-child.pid, "SIGTERM");
      }
    } catch {
      // Already gone.
    }
  }
  await Promise.all(stopped);
}
