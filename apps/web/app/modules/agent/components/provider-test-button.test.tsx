import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { formatTestResult, ProviderTestButton } from "./provider-test-button";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts?.latency) return `${key}:${opts.latency}`;
      return key;
    },
  }),
}));

vi.mock("../queries", () => ({
  useTestProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

describe("ProviderTestButton and test result formatting", () => {
  it("test connection shows success and shows the mapped failure code", () => {
    const successResult = {
      ok: true,
      latencyMs: 42,
    };
    const t = vi.fn((key: string, opts?: Record<string, unknown>) => {
      if (opts?.latency) return `${key}:${opts.latency}`;
      return key;
    });

    const successDisplay = formatTestResult(successResult, t);
    expect(successDisplay.status).toBe("success");
    expect(successDisplay.text).toContain("42");

    const failureResult = {
      ok: false,
      code: "AUTH_FAILED",
      message: "Invalid credentials",
    };
    const failureDisplay = formatTestResult(failureResult, t);
    expect(failureDisplay.status).toBe("error");
    expect(failureDisplay.text).toContain("AUTH_FAILED");

    const secretKeyMissingResult = {
      ok: false,
      code: "SECRET_KEY_MISSING",
    };
    const secretKeyDisplay = formatTestResult(secretKeyMissingResult, t);
    expect(secretKeyDisplay.status).toBe("error");
    expect(secretKeyDisplay.text).toContain("SECRET_KEY_MISSING");

    const html = renderToString(
      <ProviderTestButton providerId={1} wid={10} lastResult={successResult} />,
    );
    expect(html).toContain("agent.providers.actions.test");
  });
});
