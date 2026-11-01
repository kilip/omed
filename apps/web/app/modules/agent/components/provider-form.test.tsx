import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useProviderModels } from "../queries";
import {
  buildCreateProviderPayload,
  buildUpdateProviderPayload,
  clearFormKeyAfterSuccess,
  DEFAULT_BASE_URLS_BY_KIND,
  mapProviderError,
  ProviderForm,
  resolveBaseUrlOnKindChange,
} from "./provider-form";

vi.mock("../queries", () => ({
  useCreateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useProviderModels: vi.fn(() => ({
    data: ["gpt-4o", "gpt-4o-mini"],
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  })),
  useProbeProviderModels: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (
      key: string,
      options?: { defaultValue?: string; error?: string } | string,
    ) => {
      if (typeof options === "object" && options?.error) {
        return `${key}:${options.error}`;
      }
      if (typeof options === "object" && options?.defaultValue) {
        return options.defaultValue;
      }
      if (typeof options === "string") return options;
      return key;
    },
    i18n: { language: "en" },
  }),
}));

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

  it("clearing base URL transmits empty string when existing had base URL", () => {
    const existing = {
      id: 1,
      name: "Anthropic Claude",
      kind: "anthropic",
      baseUrl: "https://custom.endpoint.local",
      model: "claude-3-7-sonnet",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
      hasKey: true,
      keyHint: "1234",
      isDefault: false,
    };

    const values = {
      name: "Anthropic Claude",
      kind: "anthropic" as const,
      baseUrl: "",
      model: "claude-3-7-sonnet",
      apiKey: "sk-ant-new-key-123456",
      capabilities: {
        toolCalling: true,
        vision: true,
        structuredOutput: true,
        streaming: true,
      },
    };

    const { payload, inlineErrorCode } = buildUpdateProviderPayload(
      values,
      existing,
    );
    expect(inlineErrorCode).toBeUndefined();
    expect(payload.baseUrl).toBe("");
    expect(payload.apiKey).toBe("sk-ant-new-key-123456");
  });

  it("503 shows the secret-key-missing message", () => {
    const error = {
      status: 503,
      code: "SECRET_KEY_MISSING",
      message: "Secret key missing",
    };

    const mapped = mapProviderError(error, (k) => k);
    expect(mapped.field).toBeUndefined();
    expect(mapped.code).toBe("SECRET_KEY_MISSING");
    expect(mapped.message).toBe("agent.providers.errors.secretKeyMissing");
  });

  it("non-field errors return no field so drawer can display an alert", () => {
    const notFound = {
      code: "PROVIDER_NOT_FOUND",
      message: "Not found",
    };
    const mapped = mapProviderError(notFound, (k) => k);
    expect(mapped.field).toBeUndefined();
    expect(mapped.code).toBe("PROVIDER_NOT_FOUND");
    expect(mapped.message).toBe("agent.providers.errors.notFound");
  });

  describe("Default Base URLs and Kind Change", () => {
    it("defines correct DEFAULT_BASE_URLS_BY_KIND", () => {
      expect(DEFAULT_BASE_URLS_BY_KIND.openai_compat).toBe(
        "https://api.openai.com/v1",
      );
      expect(DEFAULT_BASE_URLS_BY_KIND.anthropic).toBe(
        "https://api.anthropic.com",
      );
      expect(DEFAULT_BASE_URLS_BY_KIND.gemini).toBe(
        "https://generativelanguage.googleapis.com",
      );
    });

    it("sets default baseUrl when kind changes if user has not entered a custom URL", () => {
      // When kind changed to anthropic, baseUrl defaults to https://api.anthropic.com
      const toAnthropic = resolveBaseUrlOnKindChange(
        "https://api.openai.com/v1",
        "openai_compat",
        "anthropic",
        false,
      );
      expect(toAnthropic.baseUrl).toBe("https://api.anthropic.com");
      expect(toAnthropic.isCustomized).toBe(false);

      // When kind changed to gemini, baseUrl defaults to https://generativelanguage.googleapis.com
      const toGemini = resolveBaseUrlOnKindChange(
        "https://api.anthropic.com",
        "anthropic",
        "gemini",
        false,
      );
      expect(toGemini.baseUrl).toBe(
        "https://generativelanguage.googleapis.com",
      );
      expect(toGemini.isCustomized).toBe(false);

      // When baseUrl was cleared (empty string), changing kind sets new default
      const fromEmpty = resolveBaseUrlOnKindChange(
        "",
        "openai_compat",
        "anthropic",
        true,
      );
      expect(fromEmpty.baseUrl).toBe("https://api.anthropic.com");
      expect(fromEmpty.isCustomized).toBe(false);
    });

    it("preserves custom baseUrl when kind changes", () => {
      // User types https://custom.proxy.com, changes kind, baseUrl remains https://custom.proxy.com
      const customUrl = "https://custom.proxy.com";
      const result = resolveBaseUrlOnKindChange(
        customUrl,
        "openai_compat",
        "anthropic",
        true,
      );
      expect(result.baseUrl).toBe("https://custom.proxy.com");
      expect(result.isCustomized).toBe(true);

      // User has custom internal gateway
      const internalResult = resolveBaseUrlOnKindChange(
        "https://llm-gateway.internal/v1",
        "anthropic",
        "gemini",
        true,
      );
      expect(internalResult.baseUrl).toBe("https://llm-gateway.internal/v1");
      expect(internalResult.isCustomized).toBe(true);
    });
  });

  describe("Model selection and AutoComplete", () => {
    it("allows selecting a model from options or typing a custom model", () => {
      const customPayload = buildCreateProviderPayload({
        name: "Custom Model Provider",
        kind: "openai_compat",
        baseUrl: "https://api.openai.com/v1",
        model: "my-custom-fine-tuned-model",
        apiKey: "sk-test",
        capabilities: {
          streaming: true,
          structuredOutput: true,
          toolCalling: true,
          vision: true,
        },
      });
      expect(customPayload.model).toBe("my-custom-fine-tuned-model");

      const html = renderToString(<ProviderForm />);
      // AutoComplete renders and accepts model name
      expect(html).toContain("agent.providers.fields.model");
      expect(html).toContain("agent.providers.models.refresh");
    });

    it("pre-populates baseUrl with DEFAULT_BASE_URLS_BY_KIND.openai_compat on creation", () => {
      const html = renderToString(<ProviderForm />);
      expect(html).toContain("https://api.openai.com/v1");
    });

    it("displays non-blocking error alert when model loading fails", () => {
      vi.mocked(useProviderModels).mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        isError: true,
        error: new Error("Provider models unreachable"),
        refetch: vi.fn(),
      } as unknown as ReturnType<typeof useProviderModels>);

      const html = renderToString(
        <ProviderForm
          provider={{
            id: 42,
            name: "Broken Provider",
            kind: "openai_compat",
            baseUrl: "https://api.openai.com/v1",
            model: "gpt-4o",
            capabilities: {
              streaming: true,
              structuredOutput: true,
              toolCalling: true,
              vision: true,
            },
            hasKey: true,
            keyHint: "1234",
            isDefault: false,
          }}
        />,
      );

      expect(html).toContain(
        "agent.providers.models.loadFailed:Provider models unreachable",
      );
      // AutoComplete input is still present for manual model entry
      expect(html).toContain("gpt-4o");
    });
  });
});
