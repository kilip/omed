import { betterAuth } from "better-auth";
import { type TestHelpers, testUtils } from "better-auth/plugins";
import { config } from "./config";

/**
 * `@omed/better-auth` validates its env when it is imported, so the values the
 * e2e runner uses must be in `process.env` first (real values always win).
 */
const applyAuthEnv = () => {
  const defaults: Record<string, string> = {
    OMED_DB_URL: config.dbUrl,
    OMED_AUTH_URL: config.authUrl,
    OMED_AUTH_PATH: config.authPath,
    OMED_API_URL: config.apiUrl,
    OMED_WEB_URL: config.webUrl,
    OMED_TRUSTED_ORIGINS: config.trustedOrigins,
    OMED_AUTH_SECRET: config.authSecret,
  };
  for (const [key, value] of Object.entries(defaults)) {
    process.env[key] ??= value;
  }
};

let helpers: Promise<TestHelpers> | undefined;

/**
 * Better Auth `testUtils` helpers (createUser, saveUser, login, deleteUser).
 *
 * The plugin hands out sessions without a password, so it lives in a separate
 * test-only instance, never in the auth service. The instance shares the
 * database, base URL and secret with the running service, which is why a
 * cookie it signs is accepted by the service.
 */
export const getTestHelpers = (): Promise<TestHelpers> => {
  helpers ??= (async () => {
    applyAuthEnv();
    const { options } = await import("@omed/better-auth/auth");
    const testAuth = betterAuth({
      ...options,
      plugins: [...options.plugins, testUtils()],
    });
    const context = (await testAuth.$context) as unknown as {
      test: TestHelpers;
    };
    return context.test;
  })();
  return helpers;
};
