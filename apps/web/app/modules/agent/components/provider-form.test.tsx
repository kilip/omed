import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  buildCreateProviderPayload,
  buildUpdateProviderPayload,
  clearFormKeyAfterSuccess,
  mapProviderError,
} from "./provider-form";

describe("Provider form logic", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("admin create submits apiKey and clears it from form state after success", () => {
    const values = {
      name: "Anthropic Claude",
      kind: "anthropic" as const,
      baseUrl: "https://api.anthropic.com",
      model: "claude-3-7-sonnet",
      apiKey: "sk-ant-test-123456",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
      isDefault: true,
    };

    const payload = buildCreateProviderPayload(values);
    expect(payload).toEqual({
      name: "Anthropic Claude",
      kind: "anthropic",
      baseUrl: "https://api.anthropic.com",
      model: "claude-3-7-sonnet",
      apiKey: "sk-ant-test-123456",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
      isDefault: true,
    });

    const cleared = clearFormKeyAfterSuccess(values);
    expect(cleared.apiKey).toBe("");
  });

  it("edit leaves apiKey empty and omits it from the PATCH body", () => {
    const existing = {
      id: 1,
      name: "Existing Provider",
      kind: "openai_compat",
      baseUrl: "https://api.openai.com/v1",
      model: "gpt-4o",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
      hasKey: true,
      keyHint: "5678",
      isDefault: false,
    };

    const values = {
      name: "Existing Provider Updated",
      kind: "openai_compat" as const,
      baseUrl: "https://api.openai.com/v1",
      model: "gpt-4o-mini",
      apiKey: "",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
    };

    const { payload } = buildUpdateProviderPayload(values, existing);
    expect(payload.apiKey).toBeUndefined();
    expect(payload.model).toBe("gpt-4o-mini");
  });

  it("changing base URL without a new API key shows the PROVIDER_KEY_REQUIRED error inline", () => {
    const existing = {
      id: 1,
      name: "Custom LLM",
      kind: "openai_compat",
      baseUrl: "https://old.endpoint.local",
      model: "custom-model",
      capabilities: {
        toolCalling: false,
        vision: false,
        structuredOutput: true,
        streaming: true,
      },
      hasKey: true,
      keyHint: "9999",
      isDefault: false,
    };

    const values = {
      name: "Custom LLM",
      kind: "openai_compat" as const,
      baseUrl: "https://new.endpoint.local",
      model: "custom-model",
      apiKey: "",
      capabilities: {
        toolCalling: false,
        vision: false,
        structuredOutput: true,
        streaming: true,
      },
    };

    const result = buildUpdateProviderPayload(values, existing);
    expect(result.inlineErrorCode).toBe("PROVIDER_KEY_REQUIRED");

    const mapped = mapProviderError(
      { code: "PROVIDER_KEY_REQUIRED" },
      (k) => k,
    );
    expect(mapped.field).toBe("apiKey");
    expect(mapped.code).toBe("PROVIDER_KEY_REQUIRED");
  });

  it("503 shows the secret-key-missing message", () => {
    const error = {
      status: 503,
      code: "SECRET_KEY_MISSING",
      message: "Secret key missing",
    };

    const mapped = mapProviderError(error, (k) => k);
    expect(mapped.code).toBe("SECRET_KEY_MISSING");
    expect(mapped.message).toBe("agent.providers.errors.secretKeyMissing");
  });
});
