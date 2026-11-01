import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { requireWorkspaceId } from "~/lib/workspace";
import { ProviderForm } from "../components/provider-form";
import type { Route } from "./+types/provider-new";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function ProviderNewRoute({
  loaderData,
}: Route.ComponentProps): React.ReactElement {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const wid = loaderData?.wid ?? 0;

  return (
    <Page
      title={t("agent.providers.routes.createTitle")}
      trail={[
        {
          title: t("agent.providers.title"),
          onClick: () => navigate("/agent/providers"),
        },
      ]}
    >
      <ProviderForm
        wid={wid}
        onSuccess={() => navigate("/agent/providers")}
        onCancel={() => navigate("/agent/providers")}
      />
    </Page>
  );
}
