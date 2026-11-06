import { Alert, Result, Spin } from "antd";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { requireWorkspaceId } from "~/lib/workspace";
import { useAgentPermissions } from "~/modules/agent/hooks/use-agent-permissions";
import { useConversations, useFinancialStatus } from "../hooks/use-agent-chat";
import { useFinPermissions } from "../hooks/use-fin-permissions";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function FinChatPage({
  loaderData,
}: {
  loaderData?: { wid: number };
}) {
  const { t } = useTranslation();
  const wid = loaderData?.wid ?? 0;

  const agentPerms = useAgentPermissions(wid);
  const finPerms = useFinPermissions(wid);

  const hasPermissions =
    agentPerms.canUseFinancial && finPerms.canCreateTransaction;

  const { data: status, isLoading: isStatusLoading } = useFinancialStatus(wid, {
    enabled: Boolean(wid && hasPermissions),
  });

  const { data: conversationsData, isLoading: isConversationsLoading } =
    useConversations(wid);

  if (!hasPermissions) {
    return (
      <Page title={t("fin.menu.talkToAgent", "Talk to Agent")}>
        <div className="mx-auto max-w-xl py-12">
          <Result
            status="403"
            title={t("common.forbidden", "Access Denied")}
            subTitle={t(
              "fin.agentChat.forbidden",
              "You need permission to use the financial agent and create transactions to access Talk to Agent.",
            )}
          />
        </div>
      </Page>
    );
  }

  if (isStatusLoading) {
    return (
      <Page title={t("fin.menu.talkToAgent", "Talk to Agent")}>
        <div className="flex h-64 items-center justify-center">
          <Spin size="large" />
        </div>
      </Page>
    );
  }

  const isAgentDisabled = !status?.enabled || !status?.available;

  return (
    <Page title={t("fin.menu.talkToAgent", "Talk to Agent")}>
      <div className="flex flex-col gap-4">
        {isAgentDisabled && (
          <Alert
            type="warning"
            showIcon
            message={t(
              "fin.agentChat.disabledTitle",
              "Financial Agent Unavailable",
            )}
            description={
              status?.reason
                ? t(`agent.status.${status.reason}`, status.reason)
                : t(
                    "fin.agentChat.disabledDescription",
                    "The financial agent is currently disabled or has no active provider configured.",
                  )
            }
          />
        )}

        <div className="rounded-lg border border-[var(--color-border-secondary,#f0f0f0)] bg-white p-6 shadow-sm dark:bg-[#1f1f1f]">
          <h2 className="text-lg font-semibold mb-2">
            {t("fin.menu.talkToAgent", "Talk to Agent")}
          </h2>
          <p className="text-sm text-neutral-500 mb-4">
            {t(
              "fin.agentChat.placeholderDesc",
              "Chat with the financial agent to analyze receipts, slips, and invoices into draft transactions.",
            )}
          </p>
          {isConversationsLoading ? (
            <Spin size="small" />
          ) : (
            <p className="text-xs text-neutral-400">
              {t(
                "fin.agentChat.conversationCount",
                "Total conversations: {{count}}",
                {
                  count:
                    conversationsData?.total ??
                    conversationsData?.conversations?.length ??
                    0,
                },
              )}
            </p>
          )}
        </div>
      </div>
    </Page>
  );
}
