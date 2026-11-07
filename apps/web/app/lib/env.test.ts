import { afterEach, describe, expect, it, vi } from "vitest";

const loadEnv = async () => {
  vi.resetModules();
  const mod = await import("./env");
  return mod.env;
};

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("env runtime config", () => {
  it("prefers runtime config over build-time env", async () => {
    vi.stubEnv("VITE_API_URL", "http://build.example.com");
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        apiUrl: "https://api.example.com",
        attachmentMaxSizeMB: "20",
        cmsMediaAllowedTypes: "png, .GIF",
      },
    });

    const env = await loadEnv();

    expect(env.apiUrl).toBe("https://api.example.com");
    expect(env.attachmentMaxSizeMB).toBe(20);
    expect(env.cmsMediaAllowedTypes).toEqual([".png", ".gif"]);
  });

  it("falls back to build-time env when runtime config is empty", async () => {
    vi.stubEnv("VITE_API_URL", "http://build.example.com");
    vi.stubGlobal("window", { __OMED_CONFIG__: { apiUrl: "" } });

    const env = await loadEnv();

    expect(env.apiUrl).toBe("http://build.example.com");
  });

  it("falls back to defaults when no window is available", async () => {
    vi.stubEnv("VITE_API_URL", "");

    const env = await loadEnv();

    expect(env.apiUrl).toBe("http://localhost:8001");
  });

  it("falls back to default for invalid runtime numbers", async () => {
    vi.stubEnv("VITE_ATTACHMENT_MAX_SIZE_MB", "");
    vi.stubGlobal("window", {
      __OMED_CONFIG__: { attachmentMaxSizeMB: "not-a-number" },
    });

    const env = await loadEnv();

    expect(env.attachmentMaxSizeMB).toBe(10);
  });

  it("ignores a non-object runtime config", async () => {
    vi.stubEnv("VITE_PUBLIC_URL", "http://web.example.com");
    vi.stubGlobal("window", { __OMED_CONFIG__: "bogus" });

    const env = await loadEnv();

    expect(env.publicUrl).toBe("http://web.example.com");
  });

  it("parses Sentry runtime configuration and clamps sample rate", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryDsn: "https://public@sentry.example.com/1",
        sentryEnvironment: "staging",
        sentryRelease: "1.2.3",
        sentryTracesSampleRate: "0.25",
      },
    });

    const env = await loadEnv();

    expect(env.sentryDsn).toBe("https://public@sentry.example.com/1");
    expect(env.sentryEnvironment).toBe("staging");
    expect(env.sentryRelease).toBe("1.2.3");
    expect(env.sentryTracesSampleRate).toBe(0.25);
  });

  it("clamps sentryTracesSampleRate to [0, 1] and defaults to 0 on invalid values", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryTracesSampleRate: "2.5",
      },
    });
    expect((await loadEnv()).sentryTracesSampleRate).toBe(1);

    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryTracesSampleRate: -0.5,
      },
    });
    expect((await loadEnv()).sentryTracesSampleRate).toBe(0);

    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryTracesSampleRate: "invalid",
      },
    });
    expect((await loadEnv()).sentryTracesSampleRate).toBe(0);
  });
});
