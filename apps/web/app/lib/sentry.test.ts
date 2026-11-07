import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./api";

const {
  mockInit,
  mockBrowserTracingIntegration,
  mockCaptureException,
  mockSetUser,
  mockSetTag,
} = vi.hoisted(() => ({
  mockInit: vi.fn(),
  mockBrowserTracingIntegration: vi.fn(() => ({
    name: "BrowserTracing",
  })),
  mockCaptureException: vi.fn(),
  mockSetUser: vi.fn(),
  mockSetTag: vi.fn(),
}));

vi.mock("@sentry/react", () => ({
  init: mockInit,
  browserTracingIntegration: mockBrowserTracingIntegration,
  captureException: mockCaptureException,
  setUser: mockSetUser,
  setTag: mockSetTag,
}));

const loadSentry = async () => {
  vi.resetModules();
  return import("./sentry");
};

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("sentry helpers", () => {
  it("is a no-op when sentryDsn is empty", async () => {
    vi.stubGlobal("window", { __OMED_CONFIG__: { sentryDsn: "" } });
    const sentry = await loadSentry();

    sentry.initSentry();
    sentry.captureErrorBoundary(new Error("boom"));
    sentry.captureApiError(
      new ApiError("Internal error", 500, "ERR_UNKNOWN", "req-1"),
    );
    sentry.setSentryUserContext({ id: 42 }, 7);
    sentry.setSentryWorkspaceTag(9);

    expect(mockInit).not.toHaveBeenCalled();
    expect(mockCaptureException).not.toHaveBeenCalled();
    expect(mockSetUser).not.toHaveBeenCalled();
    expect(mockSetTag).not.toHaveBeenCalled();
  });

  it("initializes Sentry once with browserTracingIntegration when DSN is configured", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        apiUrl: "https://api.omed.example.com",
        sentryDsn: "https://key@sentry.example.com/1",
        sentryEnvironment: "production",
        sentryRelease: "2026.11.07",
        sentryTracesSampleRate: "0.5",
      },
    });
    const sentry = await loadSentry();

    sentry.initSentry();
    sentry.initSentry();

    expect(mockBrowserTracingIntegration).toHaveBeenCalledWith();
    expect(mockInit).toHaveBeenCalledTimes(1);
    expect(mockInit).toHaveBeenCalledWith({
      dsn: "https://key@sentry.example.com/1",
      environment: "production",
      release: "2026.11.07",
      tracesSampleRate: 0.5,
      tracePropagationTargets: ["https://api.omed.example.com"],
      integrations: [{ name: "BrowserTracing" }],
    });
  });

  it("captures non-404 ErrorBoundary errors and skips 404 route errors", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryDsn: "https://key@sentry.example.com/1",
      },
    });
    const sentry = await loadSentry();

    const notFoundRouteError = {
      status: 404,
      statusText: "Not Found",
      internal: true,
      data: "Not found",
    };
    sentry.captureErrorBoundary(notFoundRouteError);
    expect(mockCaptureException).not.toHaveBeenCalled();

    const serverRouteError = {
      status: 500,
      statusText: "Internal Server Error",
      internal: true,
      data: "Boom",
    };
    sentry.captureErrorBoundary(serverRouteError);
    expect(mockCaptureException).toHaveBeenCalledTimes(1);
    expect(mockCaptureException).toHaveBeenCalledWith(serverRouteError);

    const renderError = new Error("Render crash");
    sentry.captureErrorBoundary(renderError);
    expect(mockCaptureException).toHaveBeenCalledTimes(2);
    expect(mockCaptureException).toHaveBeenLastCalledWith(renderError);
  });

  it("captures 5xx ApiError with request_id tag and ignores 4xx ApiError", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryDsn: "https://key@sentry.example.com/1",
      },
    });
    const sentry = await loadSentry();

    const badRequest = new ApiError(
      "Bad request",
      400,
      "ERR_BAD_REQUEST",
      "req-400",
    );
    sentry.captureApiError(badRequest);
    expect(mockCaptureException).not.toHaveBeenCalled();

    const serverError = new ApiError(
      "Database down",
      503,
      "ERR_UNKNOWN",
      "req-503",
    );
    sentry.captureApiError(serverError, {
      method: "POST",
      url: "https://api.omed.example.com/v1/items",
    });

    expect(mockCaptureException).toHaveBeenCalledTimes(1);
    expect(mockCaptureException).toHaveBeenCalledWith(serverError, {
      tags: {
        request_id: "req-503",
        "error.code": "ERR_UNKNOWN",
        "http.method": "POST",
        "http.url": "https://api.omed.example.com/v1/items",
      },
    });
  });

  it("sets and clears user and workspace_id scope", async () => {
    vi.stubGlobal("window", {
      __OMED_CONFIG__: {
        sentryDsn: "https://key@sentry.example.com/1",
      },
    });
    const sentry = await loadSentry();

    sentry.setSentryUserContext({ id: 12 }, 34);
    expect(mockSetUser).toHaveBeenCalledWith({ id: "12" });
    expect(mockSetTag).toHaveBeenCalledWith("workspace_id", "34");

    sentry.setSentryWorkspaceTag(56);
    expect(mockSetTag).toHaveBeenLastCalledWith("workspace_id", "56");

    sentry.setSentryUserContext(null);
    expect(mockSetUser).toHaveBeenLastCalledWith(null);
    expect(mockSetTag).toHaveBeenLastCalledWith("workspace_id", undefined);
  });
});
