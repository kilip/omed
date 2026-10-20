import {
  AppstoreOutlined,
  BankOutlined,
  GithubOutlined,
  GoogleOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { App, Button } from "antd";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { redirect } from "react-router";
import { Logo } from "~/components/Logo";
import { LanguageMenu, ThemeMenu } from "~/components/PreferenceMenus";
import { authClient } from "~/lib/auth-client";
import { env } from "~/lib/env";

export async function clientLoader() {
  const { data } = await authClient.getSession();
  if (data) throw redirect("/");
  return null;
}

type Provider = "google" | "github";

function Feature({
  icon,
  title,
  text,
}: {
  icon: ReactNode;
  title: string;
  text: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <span
        className="flex h-8 w-8 shrink-0 items-center justify-center"
        style={{
          borderRadius: "var(--omed-radius)",
          background: "var(--omed-tint)",
          color: "var(--omed-tint-text)",
        }}
      >
        {icon}
      </span>
      <div>
        <div className="font-medium">{title}</div>
        <div style={{ color: "var(--omed-text-secondary)" }}>{text}</div>
      </div>
    </div>
  );
}

export default function Login() {
  const { i18n, t } = useTranslation();
  const { message } = App.useApp();
  const [pending, setPending] = useState<Provider | null>(null);

  const signIn = async (provider: Provider) => {
    setPending(provider);
    try {
      const { error } = await authClient.signIn.social({
        provider,
        callbackURL: env.publicUrl,
        locale: i18n.language,
      });
      if (error) {
        setPending(null);
        message.error(error.message || t("login.failed"));
      }
    } catch {
      setPending(null);
      message.error(t("login.failed"));
    }
  };

  return (
    <div className="flex min-h-screen flex-col">
      <div className="flex justify-end gap-1 px-3 py-2">
        <LanguageMenu />
        <ThemeMenu />
      </div>
      <main className="flex flex-1 flex-wrap items-center justify-center gap-x-20 gap-y-12 px-8 pb-12">
        <section className="flex max-w-[500px] flex-[1_1_340px] flex-col gap-5">
          <span className="flex items-center gap-2 text-base font-medium">
            <Logo size={28} />
            Omed
          </span>
          <h1 className="m-0 text-2xl font-medium leading-snug">
            {t("login.headline")}
          </h1>
          <Feature
            icon={<AppstoreOutlined />}
            title={t("login.featureWorkspaces")}
            text={t("login.featureWorkspacesText")}
          />
          <Feature
            icon={<TeamOutlined />}
            title={t("login.featureMembers")}
            text={t("login.featureMembersText")}
          />
          <Feature
            icon={<BankOutlined />}
            title={t("login.featureFinance")}
            text={t("login.featureFinanceText")}
          />
        </section>

        <section
          className="surface flex w-full max-w-[400px] flex-[0_1_400px] flex-col gap-3 p-5"
          style={{ borderRadius: "var(--omed-radius-lg)" }}
          aria-labelledby="login-title"
        >
          <div>
            <h2 id="login-title" className="m-0 text-xl font-medium">
              {t("login.title")}
            </h2>
            <p
              className="m-0 mt-1"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {t("login.subtitle")}
            </p>
          </div>
          <Button
            block
            size="large"
            icon={<GoogleOutlined />}
            loading={pending === "google"}
            disabled={pending !== null && pending !== "google"}
            onClick={async () => await signIn("google")}
          >
            {t("login.google")}
          </Button>
          <Button
            block
            size="large"
            icon={<GithubOutlined />}
            loading={pending === "github"}
            disabled={pending !== null && pending !== "github"}
            onClick={async () => await signIn("github")}
          >
            {t("login.github")}
          </Button>
          <p
            className="m-0 text-xs"
            style={{ color: "var(--omed-text-secondary)" }}
          >
            {t("login.invitationHint")}
          </p>
        </section>
      </main>
    </div>
  );
}
