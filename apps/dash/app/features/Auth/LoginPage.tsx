import { GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Divider, Space, Typography, theme } from "antd";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useSearchParams } from "react-router";
import { signIn } from "~/lib/auth";
import { LanguageSwitcher } from "~/shared/ui/LanguageSwitcher";

type Provider = "google" | "github";

// Hanya terima path internal supaya tidak jadi open redirect
function safeRedirect(value: string | null) {
  return value?.startsWith("/") && !value?.startsWith("//") ? value : "/";
}

export default function LoginPage() {
  const { t } = useTranslation("auth");
  const { token } = theme.useToken();
  const [params] = useSearchParams();
  const [loading, setLoading] = useState<Provider | null>(null);
  const [error, setError] = useState<string | null>(
    params.get("error") ? t("errorGeneric") : null,
  );

  const redirectTo = safeRedirect(params.get("redirect"));

  const providers: { id: Provider; label: string; icon: React.ReactNode }[] = [
    {
      id: "google",
      label: t("continueWithGoogle"),
      icon: <GoogleOutlined />,
    },
    {
      id: "github",
      label: t("continueWithGitHub"),
      icon: <GithubOutlined />,
    },
  ];

  async function doSignIn(provider: Provider) {
    setError(null);
    setLoading(provider);
    try {
      const { error } = await signIn.social({
        provider,
        callbackURL: `${window.location.origin}${redirectTo}`,
        errorCallbackURL: `${window.location.origin}/login?error=1`,
      });
      if (error) throw error;
      // sukses: browser di-redirect ke provider, jadi loading dibiarkan
    } catch {
      setError(t("errorConnection"));
      setLoading(null);
    }
  }

  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        position: "relative",
        background: token.colorBgLayout,
      }}
    >
      <div style={{ position: "absolute", top: 16, right: 16 }}>
        <LanguageSwitcher showLabel />
      </div>

      <Card
        style={{ width: "100%", maxWidth: 400 }}
        styles={{ body: { padding: 40 } }}
      >
        <Space orientation="vertical" size={24} style={{ width: "100%" }}>
          <div style={{ textAlign: "center" }}>
            <img
              src="/logo-icon.svg"
              alt="O"
              width={48}
              height={48}
              style={{ display: "block", margin: "0 auto 20px" }}
            />
            <Typography.Title level={3} style={{ margin: 0 }}>
              {t("loginTitle")}
            </Typography.Title>
            <Typography.Text type="secondary">
              {t("loginSubtitle")}
            </Typography.Text>
          </div>

          {error && <Alert type="error" showIcon title={error} />}

          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            {providers.map((p) => (
              <Button
                key={p.id}
                block
                size="large"
                icon={p.icon}
                loading={loading === p.id}
                disabled={loading !== null && loading !== p.id}
                onClick={() => doSignIn(p.id)}
                style={{ height: 48 }}
              >
                {p.label}
              </Button>
            ))}
          </Space>

          <Divider style={{ margin: 0 }} />

          <Typography.Paragraph
            type="secondary"
            style={{ margin: 0, textAlign: "center", fontSize: 12 }}
          >
            <Trans
              ns="auth"
              i18nKey="termsNotice"
              components={{
                terms: <a href="/terms">Terms of Service</a>,
                privacy: <a href="/privacy">Privacy Policy</a>,
              }}
            />
          </Typography.Paragraph>
        </Space>
      </Card>

      <Typography.Text type="secondary" style={{ marginTop: 24, fontSize: 13 }}>
        <Trans
          ns="auth"
          i18nKey="troubleSigningIn"
          components={{
            contact: <a href="mailto:support@omed.app">Contact us</a>,
          }}
        />
      </Typography.Text>
    </main>
  );
}
