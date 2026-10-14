import { GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { Alert, App, Button } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Logo } from "~/components/Logo";
import { LanguageMenu, ThemeMenu } from "~/components/PreferenceMenus";
import { api, unwrapOne } from "~/lib/api";
import { authClient } from "~/lib/auth-client";
import { setActiveWorkspaceId } from "~/lib/workspace";
import type { Route } from "./+types/invite";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const [preview, session] = await Promise.all([
    unwrapOne(
      api.GET("/invitations/{token}", {
        params: { path: { token: params.token } },
      }),
    ),
    authClient.getSession(),
  ]);
  return { preview, signedIn: Boolean(session.data) };
}

export default function Invite({ loaderData, params }: Route.ComponentProps) {
  const { preview, signedIn } = loaderData;
  const { t } = useTranslation();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [pending, setPending] = useState(false);

  const accept = async () => {
    setPending(true);
    try {
      await unwrapOne(
        api.POST("/invitations/{token}/accept", {
          params: { path: { token: params.token } },
        }),
      );
      if (preview.workspaceId !== undefined) {
        setActiveWorkspaceId(preview.workspaceId);
      }
      message.success(t("invite.joined"));
      navigate("/");
    } catch (error) {
      message.error(error instanceof Error ? error.message : t("common.error"));
    } finally {
      setPending(false);
    }
  };

  const signIn = (provider: "google" | "github") =>
    authClient.signIn.social({ provider, callbackURL: window.location.href });

  return (
    <div className="flex min-h-screen flex-col">
      <div className="flex justify-end gap-1 px-3 py-2">
        <LanguageMenu />
        <ThemeMenu />
      </div>
      <main className="flex flex-1 items-center justify-center px-4 pb-12">
        <section
          className="glass-strong flex w-full max-w-[420px] flex-col gap-4 p-6"
          style={{ borderRadius: "var(--omed-radius-lg)" }}
        >
          <span className="flex items-center gap-2 font-medium">
            <Logo size={24} />
            Omed
          </span>
          <div>
            <h1 className="m-0 text-xl font-medium">
              {t("invite.title", { workspace: preview.workspaceName })}
            </h1>
            {preview.inviterName ? (
              <p
                className="m-0 mt-1"
                style={{ color: "var(--omed-text-secondary)" }}
              >
                {t("invite.invitedBy", { name: preview.inviterName })}
              </p>
            ) : null}
          </div>
          {preview.workspaceDescription ? (
            <p className="m-0">{preview.workspaceDescription}</p>
          ) : null}
          {preview.roles?.length ? (
            <p className="m-0" style={{ color: "var(--omed-text-secondary)" }}>
              {t("invite.roles", { roles: preview.roles.join(", ") })}
            </p>
          ) : null}

          {!preview.isValid ? (
            <Alert type="warning" showIcon message={t("invite.invalid")} />
          ) : signedIn ? (
            <Button
              type="primary"
              size="large"
              block
              loading={pending}
              onClick={accept}
            >
              {t("invite.accept")}
            </Button>
          ) : (
            <>
              <p
                className="m-0"
                style={{ color: "var(--omed-text-secondary)" }}
              >
                {t("invite.signInFirst")}
              </p>
              <Button
                block
                size="large"
                icon={<GoogleOutlined />}
                onClick={() => signIn("google")}
              >
                {t("login.google")}
              </Button>
              <Button
                block
                size="large"
                icon={<GithubOutlined />}
                onClick={() => signIn("github")}
              >
                {t("login.github")}
              </Button>
            </>
          )}
        </section>
      </main>
    </div>
  );
}
