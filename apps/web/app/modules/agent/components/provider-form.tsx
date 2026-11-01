import { ReloadOutlined } from "@ant-design/icons";
import {
  Alert,
  App,
  AutoComplete,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Select,
  Space,
  Switch,
  Tooltip,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type Capabilities,
  type CreateProviderRequest,
  type ProviderResponse,
  type UpdateProviderRequest,
  useCreateProvider,
  useProbeProviderModels,
  useProviderModels,
  useUpdateProvider,
} from "../queries";

export type ProviderKind = "openai_compat" | "anthropic" | "gemini";

export const DEFAULT_BASE_URLS_BY_KIND: Record<ProviderKind, string> = {
  openai_compat: "https://api.openai.com/v1",
  anthropic: "https://api.anthropic.com",
  gemini: "https://generativelanguage.googleapis.com",
};

export type ProviderFormValues = {
  name: string;
  kind: ProviderKind;
  baseUrl?: string;
  model: string;
  apiKey?: string;
  capabilities: Capabilities;
  isDefault?: boolean;
};

export const DEFAULT_CAPABILITIES_BY_KIND: Record<string, Capabilities> = {
  openai_compat: {
    streaming: true,
    structuredOutput: true,
    toolCalling: true,
    vision: true,
  },
  anthropic: {
    streaming: true,
    structuredOutput: true,
    toolCalling: true,
    vision: true,
  },
  gemini: {
    streaming: true,
    structuredOutput: true,
    toolCalling: true,
    vision: true,
  },
};

export function resolveBaseUrlOnKindChange(
  currentUrl: string | undefined,
  prevKind: ProviderKind | undefined,
  nextKind: ProviderKind,
  isCustomized: boolean,
): { baseUrl: string; isCustomized: boolean } {
  const trimmed = currentUrl?.trim() ?? "";
  const isUntouchedOrDefault =
    !isCustomized ||
    trimmed === "" ||
    (prevKind !== undefined && trimmed === DEFAULT_BASE_URLS_BY_KIND[prevKind]);

  if (isUntouchedOrDefault) {
    return {
      baseUrl: DEFAULT_BASE_URLS_BY_KIND[nextKind],
      isCustomized: false,
    };
  }

  return {
    baseUrl: trimmed,
    isCustomized: true,
  };
}

export function buildCreateProviderPayload(
  values: ProviderFormValues,
): CreateProviderRequest {
  const payload: CreateProviderRequest = {
    name: values.name.trim(),
    kind: values.kind,
    model: values.model.trim(),
    apiKey: values.apiKey?.trim() || "",
    capabilities: values.capabilities,
    isDefault: values.isDefault,
  };
  if (values.baseUrl && values.baseUrl.trim() !== "") {
    payload.baseUrl = values.baseUrl.trim();
  }
  return payload;
}

export function clearFormKeyAfterSuccess(
  values: ProviderFormValues,
): ProviderFormValues {
  return {
    ...values,
    apiKey: "",
  };
}

export function buildUpdateProviderPayload(
  values: ProviderFormValues,
  existing?: ProviderResponse,
): { payload: UpdateProviderRequest; inlineErrorCode?: string } {
  const trimmedBaseUrl = values.baseUrl?.trim() || "";
  const existingBaseUrl = existing?.baseUrl?.trim() || "";
  const trimmedKey = values.apiKey?.trim() || "";

  let inlineErrorCode: string | undefined;
  if (existing && trimmedBaseUrl !== existingBaseUrl && !trimmedKey) {
    inlineErrorCode = "PROVIDER_KEY_REQUIRED";
  }

  const payload: UpdateProviderRequest = {
    name: values.name.trim(),
    model: values.model.trim(),
    capabilities: values.capabilities,
  };

  if (existing) {
    if (trimmedBaseUrl !== (existing.baseUrl ?? "")) {
      payload.baseUrl = trimmedBaseUrl;
    }
  } else if (trimmedBaseUrl) {
    payload.baseUrl = trimmedBaseUrl;
  }

  if (trimmedKey) {
    payload.apiKey = trimmedKey;
  }

  return { payload, inlineErrorCode };
}

export function mapProviderError(
  err: unknown,
  t: (k: string) => string,
): { field?: keyof ProviderFormValues; code?: string; message: string } {
  const errorObj = (typeof err === "object" && err !== null ? err : {}) as {
    status?: number;
    code?: string;
    message?: string;
  };

  const status = errorObj.status;
  const code =
    errorObj.code || (status === 503 ? "SECRET_KEY_MISSING" : undefined);

  if (code === "SECRET_KEY_MISSING" || status === 503) {
    return {
      code: "SECRET_KEY_MISSING",
      message: t("agent.providers.errors.secretKeyMissing"),
    };
  }

  if (code === "PROVIDER_KEY_REQUIRED") {
    return {
      field: "apiKey",
      code: "PROVIDER_KEY_REQUIRED",
      message: t("agent.providers.errors.apiKeyRequired"),
    };
  }

  if (code === "PROVIDER_NAME_TAKEN") {
    return {
      field: "name",
      code: "PROVIDER_NAME_TAKEN",
      message: t("agent.providers.errors.nameTaken"),
    };
  }

  if (code === "INVALID_BASE_URL") {
    return {
      field: "baseUrl",
      code: "INVALID_BASE_URL",
      message: t("agent.providers.errors.invalidBaseUrl"),
    };
  }

  if (code === "PROVIDER_NOT_FOUND") {
    return {
      code: "PROVIDER_NOT_FOUND",
      message: t("agent.providers.errors.notFound"),
    };
  }

  return {
    code,
    message: errorObj.message || t("common.error"),
  };
}

export type ProviderFormProps = {
  provider?: ProviderResponse | null;
  wid?: number;
  onSuccess?: () => void;
  onCancel?: () => void;
};

export function ProviderForm({
  provider,
  wid,
  onSuccess,
  onCancel,
}: ProviderFormProps): React.ReactElement {
  const { t } = useTranslation();
  const [form] = Form.useForm<ProviderFormValues>();
  const [generalError, setGeneralError] = useState<string | null>(null);

  const initialKind = (provider?.kind as ProviderKind) ?? "openai_compat";
  const [currentKind, setCurrentKind] = useState<ProviderKind>(initialKind);
  const [isBaseUrlCustomized, setIsBaseUrlCustomized] = useState(() => {
    if (!provider?.baseUrl) return false;
    return provider.baseUrl !== DEFAULT_BASE_URLS_BY_KIND[initialKind];
  });

  const [probedModels, setProbedModels] = useState<string[] | null>(null);
  const [probeError, setProbeError] = useState<string | null>(null);

  const app = App.useApp?.();
  const message = app?.message;

  const createMutation = useCreateProvider(wid);
  const updateMutation = useUpdateProvider(wid);
  const isPending = createMutation.isPending || updateMutation.isPending;

  const queryModels = useProviderModels(wid, provider?.id);
  const probeMutation = useProbeProviderModels(wid);

  const watchedApiKey = Form.useWatch("apiKey", form);
  const watchedBaseUrl = Form.useWatch("baseUrl", form);
  const watchedKind = Form.useWatch("kind", form);

  useEffect(() => {
    setGeneralError(null);
    setProbeError(null);
    setProbedModels(null);
    if (provider) {
      const k = (provider.kind as ProviderKind) ?? "openai_compat";
      setCurrentKind(k);
      const existingUrl = provider.baseUrl ?? "";
      const isCustom = Boolean(
        existingUrl && existingUrl !== DEFAULT_BASE_URLS_BY_KIND[k],
      );
      setIsBaseUrlCustomized(isCustom);
      form.setFieldsValue({
        name: provider.name ?? "",
        kind: k,
        baseUrl: existingUrl,
        model: provider.model ?? "",
        apiKey: "",
        capabilities: {
          streaming: provider.capabilities?.streaming ?? true,
          structuredOutput: provider.capabilities?.structuredOutput ?? true,
          toolCalling: provider.capabilities?.toolCalling ?? true,
          vision: provider.capabilities?.vision ?? true,
        },
        isDefault: provider.isDefault ?? false,
      });
    } else {
      setCurrentKind("openai_compat");
      setIsBaseUrlCustomized(false);
      form.setFieldsValue({
        name: "",
        kind: "openai_compat",
        baseUrl: DEFAULT_BASE_URLS_BY_KIND.openai_compat,
        model: "",
        apiKey: "",
        capabilities: DEFAULT_CAPABILITIES_BY_KIND.openai_compat,
        isDefault: false,
      });
    }
  }, [provider, form]);

  useEffect(() => {
    const trimmedKey = watchedApiKey?.trim();
    const trimmedUrl = watchedBaseUrl?.trim();
    const kind = watchedKind || currentKind;

    if (!trimmedKey || !trimmedUrl || !kind) {
      return;
    }

    const timer = setTimeout(async () => {
      try {
        setProbeError(null);
        const res = await probeMutation.mutateAsync({
          apiKey: trimmedKey,
          baseUrl: trimmedUrl,
          kind,
        });
        setProbedModels(res ?? []);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        setProbeError(msg);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [
    watchedApiKey,
    watchedBaseUrl,
    watchedKind,
    currentKind,
    probeMutation.mutateAsync,
  ]);

  const handleKindChange = (newKind: ProviderKind) => {
    form.setFieldValue("kind", newKind);
    const currentUrl = form.getFieldValue("baseUrl");
    const next = resolveBaseUrlOnKindChange(
      currentUrl,
      currentKind,
      newKind,
      isBaseUrlCustomized,
    );
    form.setFieldValue("baseUrl", next.baseUrl);
    setIsBaseUrlCustomized(next.isCustomized);
    setCurrentKind(newKind);
    setProbedModels(null);
    setProbeError(null);
  };

  const handleBaseUrlChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    const kind = (form.getFieldValue("kind") || currentKind) as ProviderKind;
    if (val && val !== DEFAULT_BASE_URLS_BY_KIND[kind]) {
      setIsBaseUrlCustomized(true);
    } else {
      setIsBaseUrlCustomized(false);
    }
  };

  const handleRefresh = async () => {
    const currentKey = form.getFieldValue("apiKey")?.trim();
    const currentUrl = form.getFieldValue("baseUrl")?.trim();
    const kind = (form.getFieldValue("kind") || currentKind) as ProviderKind;

    if (currentKey && currentUrl && kind) {
      try {
        setProbeError(null);
        const res = await probeMutation.mutateAsync({
          apiKey: currentKey,
          baseUrl: currentUrl,
          kind,
        });
        setProbedModels(res ?? []);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        setProbeError(msg);
      }
    } else if (provider?.id) {
      try {
        setProbeError(null);
        setProbedModels(null);
        await queryModels.refetch();
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        setProbeError(msg);
      }
    }
  };

  const rawModels = probedModels ?? queryModels.data ?? [];
  const modelOptions = useMemo(
    () => rawModels.map((m) => ({ value: m, label: m })),
    [rawModels],
  );

  const isLoadingModels = queryModels.isFetching || probeMutation.isPending;
  const modelError =
    probeError ||
    (queryModels.isError
      ? (queryModels.error instanceof Error
          ? queryModels.error.message
          : String(queryModels.error)) || t("common.error")
      : null);

  const handleSubmit = async () => {
    try {
      setGeneralError(null);
      const values = await form.validateFields();
      if (provider) {
        const { payload, inlineErrorCode } = buildUpdateProviderPayload(
          values,
          provider,
        );
        if (inlineErrorCode === "PROVIDER_KEY_REQUIRED") {
          form.setFields([
            {
              name: "apiKey",
              errors: [t("agent.providers.errors.apiKeyRequired")],
            },
          ]);
          return;
        }
        await updateMutation.mutateAsync({
          id: provider.id ?? 0,
          body: payload,
        });
      } else {
        const payload = buildCreateProviderPayload(values);
        await createMutation.mutateAsync(payload);
      }
      message?.success(t("common.saved"));
      form.resetFields();
      onSuccess?.();
    } catch (err: unknown) {
      if (err && typeof err === "object" && "errorFields" in err) {
        return;
      }
      const mapped = mapProviderError(err, t);
      if (mapped.field) {
        form.setFields([{ name: mapped.field, errors: [mapped.message] }]);
      } else {
        setGeneralError(mapped.message);
        message?.error(mapped.message);
      }
    }
  };

  return (
    <Card>
      {generalError && (
        <Alert
          type="error"
          message={generalError}
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          name: provider?.name ?? "",
          kind: initialKind,
          baseUrl: provider
            ? (provider.baseUrl ?? "")
            : DEFAULT_BASE_URLS_BY_KIND.openai_compat,
          model: provider?.model ?? "",
          apiKey: "",
          capabilities:
            provider?.capabilities ??
            DEFAULT_CAPABILITIES_BY_KIND.openai_compat,
          isDefault: provider?.isDefault ?? false,
        }}
      >
        <Form.Item
          name="name"
          label={t("agent.providers.fields.name")}
          rules={[
            {
              required: true,
              message: t("agent.providers.validation.nameRequired"),
            },
          ]}
        >
          <Input />
        </Form.Item>

        <Form.Item
          name="kind"
          label={t("agent.providers.fields.kind")}
          rules={[
            {
              required: true,
              message: t("agent.providers.validation.kindRequired"),
            },
          ]}
        >
          <Select
            disabled={Boolean(provider)}
            onChange={handleKindChange}
            options={[
              {
                value: "openai_compat",
                label: t("agent.providers.kinds.openai_compat"),
              },
              {
                value: "anthropic",
                label: t("agent.providers.kinds.anthropic"),
              },
              {
                value: "gemini",
                label: t("agent.providers.kinds.gemini"),
              },
            ]}
          />
        </Form.Item>

        <Form.Item
          name="model"
          label={t("agent.providers.fields.model")}
          rules={[
            {
              required: true,
              message: t("agent.providers.validation.modelRequired"),
            },
          ]}
          extra={
            isLoadingModels
              ? t("agent.providers.models.loading")
              : t("agent.providers.models.customHint")
          }
        >
          <Space.Compact style={{ width: "100%" }}>
            <Form.Item name="model" noStyle>
              <AutoComplete
                options={modelOptions}
                filterOption={(inputValue, option) =>
                  (option?.value ?? "")
                    .toLowerCase()
                    .includes(inputValue.toLowerCase())
                }
                placeholder="e.g. gpt-4o, claude-3-7-sonnet"
                style={{ width: "100%" }}
              />
            </Form.Item>
            <Tooltip title={t("agent.providers.models.refresh")}>
              <Button
                icon={<ReloadOutlined spin={isLoadingModels} />}
                loading={isLoadingModels}
                onClick={handleRefresh}
                aria-label={t("agent.providers.models.refresh")}
              />
            </Tooltip>
          </Space.Compact>
          {modelError && (
            <Alert
              type="warning"
              showIcon
              message={t("agent.providers.models.loadFailed", {
                error: modelError,
              })}
              style={{ marginTop: 8 }}
            />
          )}
        </Form.Item>

        <Form.Item name="baseUrl" label={t("agent.providers.fields.baseUrl")}>
          <Input
            placeholder={DEFAULT_BASE_URLS_BY_KIND[currentKind]}
            onChange={handleBaseUrlChange}
          />
        </Form.Item>

        <Form.Item
          name="apiKey"
          label={t("agent.providers.fields.apiKey")}
          rules={[
            {
              required: !provider,
              message: t("agent.providers.validation.apiKeyRequired"),
            },
          ]}
          extra={provider ? t("agent.providers.apiKeyEditHelp") : undefined}
        >
          <Input.Password placeholder={provider ? "••••••••" : undefined} />
        </Form.Item>

        <Form.Item
          name={["capabilities", "streaming"]}
          label={t("agent.providers.capabilities.streaming")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name={["capabilities", "structuredOutput"]}
          label={t("agent.providers.capabilities.structuredOutput")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name={["capabilities", "toolCalling"]}
          label={t("agent.providers.capabilities.toolCalling")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name={["capabilities", "vision"]}
          label={t("agent.providers.capabilities.vision")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        {!provider && (
          <Form.Item
            name="isDefault"
            label={t("agent.providers.fields.isDefault")}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        )}

        <Form.Item style={{ marginBottom: 0, marginTop: 16 }}>
          <Space>
            {onCancel && (
              <Button onClick={onCancel}>{t("common.cancel")}</Button>
            )}
            <Button type="primary" loading={isPending} onClick={handleSubmit}>
              {t("common.save")}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
}

export type ProviderFormDrawerProps = {
  open: boolean;
  onClose: () => void;
  provider?: ProviderResponse | null;
  wid?: number;
  onSuccess?: () => void;
};

export function ProviderFormDrawer({
  open,
  onClose,
  provider,
  wid,
  onSuccess,
}: ProviderFormDrawerProps): React.ReactElement | null {
  const { t } = useTranslation();

  return (
    <Drawer
      title={
        provider
          ? t("agent.providers.drawer.editTitle")
          : t("agent.providers.drawer.createTitle")
      }
      open={open}
      onClose={onClose}
      destroyOnHidden
      footer={null}
    >
      {open && (
        <ProviderForm
          provider={provider}
          wid={wid}
          onCancel={onClose}
          onSuccess={() => {
            onClose();
            onSuccess?.();
          }}
        />
      )}
    </Drawer>
  );
}
