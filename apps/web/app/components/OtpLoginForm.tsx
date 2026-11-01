import { App, Button, Form, Input } from "antd";
import { type JSX, useEffect, useReducer, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { authClient } from "~/lib/auth-client";

export type OtpState = {
  step: "email" | "code";
  email: string;
  codeLength: number;
  resendAt: number;
};

export type OtpAction =
  | {
      type: "requested";
      email: string;
      codeLength: number;
      resendAfterSeconds: number;
      now: number;
    }
  | { type: "changeEmail" };

export function otpStepReducer(state: OtpState, action: OtpAction): OtpState {
  switch (action.type) {
    case "requested":
      return {
        step: "code",
        email: action.email,
        codeLength: action.codeLength,
        resendAt: action.now + action.resendAfterSeconds * 1000,
      };
    case "changeEmail":
      return {
        ...state,
        step: "email",
      };
    default:
      return state;
  }
}

export function remainingSeconds(resendAt: number, now: number): number {
  return Math.max(0, Math.ceil((resendAt - now) / 1000));
}

export function mapOtpErrorKey(
  code?: string,
): "login.otp.invalid" | "login.otp.unavailable" | "login.otp.failed" {
  if (code === "AUTH_OTP_INVALID") {
    return "login.otp.invalid";
  }
  if (code === "AUTH_OTP_UNAVAILABLE") {
    return "login.otp.unavailable";
  }
  return "login.otp.failed";
}

export function OtpLoginForm(): JSX.Element {
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const navigate = useNavigate();

  const [state, dispatch] = useReducer(otpStepReducer, {
    step: "email",
    email: "",
    codeLength: 6,
    resendAt: 0,
  });

  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState("");
  const [now, setNow] = useState(() => Date.now());
  const [form] = Form.useForm<{ email: string }>();

  useEffect(() => {
    if (state.step !== "code") return;
    const timer = setInterval(() => {
      setNow(Date.now());
    }, 1000);
    return () => clearInterval(timer);
  }, [state.step]);

  useEffect(() => {
    if (state.step === "email" && state.email) {
      form.setFieldsValue({ email: state.email });
    }
  }, [state.step, state.email, form]);

  const sendOtp = async (targetEmail: string) => {
    setLoading(true);
    try {
      const res = await authClient.otp.request({
        email: targetEmail,
        locale: i18n.language,
      });
      if (res.error || !res.data) {
        message.error(t(mapOtpErrorKey(res.error?.code)));
        return false;
      }
      const currentTime = Date.now();
      setNow(currentTime);
      dispatch({
        type: "requested",
        email: targetEmail,
        codeLength: res.data.codeLength ?? 6,
        resendAfterSeconds: res.data.resendAfterSeconds ?? 60,
        now: currentTime,
      });
      setCode("");
      return true;
    } catch {
      message.error(t("login.otp.failed"));
      return false;
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (codeToVerify: string) => {
    setLoading(true);
    try {
      const res = await authClient.otp.verify({
        email: state.email,
        code: codeToVerify,
      });
      if (res.error) {
        setCode("");
        message.error(t(mapOtpErrorKey(res.error.code)));
        return;
      }
      navigate("/", { replace: true });
    } catch {
      setCode("");
      message.error(t("login.otp.failed"));
    } finally {
      setLoading(false);
    }
  };

  const handleOtpChange = (val: string) => {
    setCode(val);
    if (val.length === state.codeLength && !loading) {
      handleVerify(val);
    }
  };

  const remaining = remainingSeconds(state.resendAt, now);

  if (state.step === "code") {
    return (
      <div className="flex flex-col gap-3">
        <p
          className="m-0 text-xs"
          style={{ color: "var(--omed-text-secondary)" }}
        >
          {t("login.otp.codeHint", {
            length: state.codeLength,
            email: state.email,
          })}
        </p>
        <div className="flex justify-center">
          <Input.OTP
            length={state.codeLength}
            value={code}
            onChange={handleOtpChange}
            disabled={loading}
            size="large"
          />
        </div>
        <Button
          block
          size="large"
          disabled={remaining > 0 || loading}
          loading={loading}
          onClick={() => sendOtp(state.email)}
        >
          {remaining > 0
            ? t("login.otp.resendIn", { seconds: remaining })
            : t("login.otp.resend")}
        </Button>
        <Button
          type="link"
          block
          onClick={() => {
            dispatch({ type: "changeEmail" });
            setCode("");
          }}
        >
          {t("login.otp.changeEmail")}
        </Button>
      </div>
    );
  }

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={(values) => sendOtp(values.email.trim())}
      requiredMark={false}
      initialValues={{ email: state.email }}
    >
      <Form.Item
        name="email"
        label={t("login.otp.emailLabel")}
        rules={[{ required: true }, { type: "email" }]}
      >
        <Input
          type="email"
          size="large"
          placeholder={t("login.otp.emailPlaceholder")}
          autoComplete="email"
          disabled={loading}
        />
      </Form.Item>
      <Form.Item className="mb-0">
        <Button
          block
          type="primary"
          size="large"
          htmlType="submit"
          loading={loading}
        >
          {t("login.otp.sendCode")}
        </Button>
      </Form.Item>
    </Form>
  );
}
