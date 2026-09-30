import path from "node:path";
import dotenv from "dotenv";

// Load environment variables from monorepo root and workspace env files
const monorepoRoot = path.resolve(import.meta.dirname, "../../../..");
dotenv.config({ path: path.join(monorepoRoot, ".env.local") });
dotenv.config({ path: path.join(monorepoRoot, "apps/auth/.env") });
dotenv.config({
  path: path.join(monorepoRoot, "packages/better-auth/.env.test"),
});
dotenv.config({ path: path.resolve(import.meta.dirname, "../../.env") });

export const e2eConfig = {
  baseUrl: process.env.BASE_URL || "http://localhost:3001",
  cookieDomain: process.env.COOKIE_DOMAIN || "localhost",
  browser: {
    headless: process.env.HEADLESS !== "false",
    slowMo: Number(process.env.SLOW_MO || 0),
    timeout: 30000,
  },
  viewport: {
    width: 1280,
    height: 720,
  },
};
