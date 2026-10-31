import { Button, Space, Tag } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { type TestResult, useTestProvider } from "../queries";
import { mapProviderError } from "./provider-form";

export type TestResultDisplay = {
  status: "success" | "error";
  text: string;
};

export function formatTestResult(
  result: TestResult,
  t: (k: string, opts?: Record<string, unknown>) => string,
): TestResultDisplay {
  if (result.ok) {
    return {
      status: "success",
      text: t("agent.providers.test.success", {
        latency: result.latencyMs ?? 0,
      }),
    };
  }

  const code = result.code || "UNKNOWN_ERROR";
  return {
    status: "error",
    text: t(`agent.providers.test.codes.${code}`),
  };
}

export type ProviderTestButtonProps = {
  providerId: number;
  wid?: number;
  disabled?: boolean;
  lastResult?: TestResult;
  onResult?: (result: TestResult) => void;
  size?: "small" | "middle" | "large";
};

export function ProviderTestButton({
  providerId,
  wid,
  disabled,
  lastResult,
  onResult,
  size = "small",
}: ProviderTestButtonProps): React.ReactElement {
  const { t } = useTranslation();
  const testMutation = useTestProvider(wid);
  const [result, setResult] = useState<TestResult | undefined>(lastResult);

  const activeResult = lastResult ?? result;

  const handleTest = async () => {
    try {
      const res = await testMutation.mutateAsync(providerId);
      setResult(res);
      onResult?.(res);
    } catch (err) {
      const mapped = mapProviderError(err, t);
      const errResult: TestResult = {
        ok: false,
        code: mapped.code || "UNKNOWN_ERROR",
      };
      setResult(errResult);
      onResult?.(errResult);
    }
  };

  const display = activeResult ? formatTestResult(activeResult, t) : null;

  return (
    <Space size="small">
      <Button
        size={size}
        loading={testMutation.isPending}
        disabled={disabled || !providerId}
        onClick={handleTest}
      >
        {t("agent.providers.actions.test")}
      </Button>
      {display && (
        <Tag color={display.status === "success" ? "success" : "error"}>
          {display.text}
        </Tag>
      )}
    </Space>
  );
}
