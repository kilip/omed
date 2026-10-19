import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { requireWorkspaceId } from "~/lib/workspace";
import type { Route } from "./+types/entry";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  return { wid, id };
}

export default function EntryRoute() {
  const { t } = useTranslation();

  return (
    <Page title={t("fin.journal.number", "Journal Entry")}>
      <Panel>
        <div>Entry detail stub</div>
      </Panel>
    </Page>
  );
}
