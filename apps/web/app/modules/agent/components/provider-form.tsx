import {
  Alert,
  App,
  Button,
  Drawer,
  Form,
  Input,
  Select,
  Space,
  Switch,
} from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type Capabilities,
  type CreateProviderRequest,
  type ProviderResponse,
  type UpdateProviderRequest,
  useCreateProvider,
  useUpdateProvider,
} from "../queries";

export type ProviderKind = "openai_compat" | "anthropic" | "gemini";

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

  if (trimmedBaseUrl) {
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
  const [form] = Form.useForm<ProviderFormValues>();
  const [generalError, setGeneralError] = useState<string | null>(null);
  const app = App.useApp?.();
  const message = app?.message;

  const createMutation = useCreateProvider(wid);
  const updateMutation = useUpdateProvider(wid);
  const isPending = createMutation.isPending || updateMutation.isPending;

  useEffect(() => {
    if (open) {
      setGeneralError(null);
      if (provider) {
        form.setFieldsValue({
          name: provider.name ?? "",
          kind: (provider.kind as ProviderKind) ?? "openai_compat",
          baseUrl: provider.baseUrl ?? "",
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
        form.setFieldsValue({
          name: "",
          kind: "openai_compat",
          baseUrl: "",
          model: "",
          apiKey: "",
          capabilities: DEFAULT_CAPABILITIES_BY_KIND.openai_compat,
          isDefault: false,
        });
      }
    }
  }, [open, provider, form]);

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
      onClose();
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
    <Drawer
      title={
        provider
          ? t("agent.providers.drawer.editTitle")
          : t("agent.providers.drawer.createTitle")
      }
      open={open}
      onClose={onClose}
      destroyOnHidden
      extra={
        <Space>
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={isPending} onClick={handleSubmit}>
            {t("common.save")}
          </Button>
        </Space>
      }
    >
      {generalError && (
        <Alert
          type="error"
          message={generalError}
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}
      <Form form={form} layout="vertical">
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
        >
          <Input />
        </Form.Item>

        <Form.Item name="baseUrl" label={t("agent.providers.fields.baseUrl")}>
          <Input placeholder="https://api.openai.com/v1" />
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
      </Form>
    </Drawer>
  );
}
