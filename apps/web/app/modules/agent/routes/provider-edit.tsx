import { Button, Card, Spin } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { requireWorkspaceId } from "~/lib/workspace";
import { ProviderForm } from "../components/provider-form";
import { useProvider } from "../queries";
import type { Route } from "./+types/provider-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  return { wid, id };
}

export default function ProviderEditRoute({
  loaderData,
  params,
}: Route.ComponentProps): React.ReactElement {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const routeParams = useParams();

  const wid = loaderData?.wid ?? 0;
  const id =
    loaderData?.id ??
    (params?.id
      ? Number(params.id)
      : routeParams.id
        ? Number(routeParams.id)
        : 0);

  const { data: provider, isLoading } = useProvider(id, wid);

  if (isLoading) {
    return (
      <Page
        title={t("agent.providers.routes.editTitle")}
        trail={[
          {
            title: t("agent.providers.title"),
            onClick: () => navigate("/agent/providers"),
          },
        ]}
      >
        <Card style={{ textAlign: "center", padding: 48 }}>
          <Spin />
        </Card>
      </Page>
    );
  }

  if (!provider) {
    return (
      <Page
        title={t("agent.providers.routes.editTitle")}
        trail={[
          {
            title: t("agent.providers.title"),
            onClick: () => navigate("/agent/providers"),
          },
        ]}
      >
        <Panel strong>
          <div className="py-4 text-center">
            <p className="mb-3" style={{ color: "var(--omed-text-secondary)" }}>
              {t("agent.providers.errors.notFound")}
            </p>
            <Button onClick={() => navigate("/agent/providers")}>
              {t("common.cancel")}
            </Button>
          </div>
        </Panel>
      </Page>
    );
  }

  return (
    <Page
      title={t("agent.providers.routes.editTitle")}
      trail={[
        {
          title: t("agent.providers.title"),
          onClick: () => navigate("/agent/providers"),
        },
      ]}
    >
      <ProviderForm
        provider={provider}
        wid={wid}
        onSuccess={() => navigate("/agent/providers")}
        onCancel={() => navigate("/agent/providers")}
      />
    </Page>
  );
}
