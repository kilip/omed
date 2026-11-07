import { MenuOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { Alert, App, Button, Drawer, Result, Spin } from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { useAgentPermissions } from "~/modules/agent/hooks/use-agent-permissions";
import {
  ConversationSidebar,
  MessageComposer,
  MessageList,
} from "../components/chat";
import {
  type DraftProposal,
  type SendMessageParams,
  useConfirmDraft,
  useConversation,
  useConversations,
  useCreateConversation,
  useDeleteConversation,
  useDiscardDraft,
  useFinancialStatus,
  useSendMessage,
} from "../hooks/use-agent-chat";
import { useFinPermissions } from "../hooks/use-fin-permissions";
import { accountsQueryOptions } from "../queries";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  await getQueryClient().ensureQueryData(accountsQueryOptions(wid));
  return { wid };
}

export default function FinChatPage({
  loaderData,
}: {
  loaderData?: { wid: number };
}) {
  const { t } = useTranslation();
  const { message } = App.useApp();
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
  const conversations = conversationsData?.conversations ?? [];

  const { data: accounts = [] } = useQuery(accountsQueryOptions(wid));

  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [confirmingMessageId, setConfirmingMessageId] = useState<number | null>(
    null,
  );
  const [discardingMessageId, setDiscardingMessageId] = useState<number | null>(
    null,
  );

  // Automatically select the first conversation if none selected
  useEffect(() => {
    if (selectedId === null && conversations.length > 0) {
      setSelectedId(conversations[0]?.id ?? null);
    }
  }, [conversations, selectedId]);

  const activeConversationId = selectedId ?? conversations[0]?.id ?? null;

  const { data: activeConversation, isLoading: isDetailLoading } =
    useConversation(wid, activeConversationId ?? undefined);

  const createConversation = useCreateConversation(wid);
  const deleteConversation = useDeleteConversation(wid);
  const sendMessage = useSendMessage(wid, activeConversationId ?? undefined);
  const confirmDraft = useConfirmDraft(wid);
  const discardDraft = useDiscardDraft(wid);

  const handleNewConversation = async () => {
    try {
      const conv = await createConversation.mutateAsync({
        title: t("fin.chat.newConversation", "New conversation"),
      });
      if (conv?.id) {
        setSelectedId(conv.id);
        setMobileDrawerOpen(false);
      }
    } catch {
      // Error handled by query/mutation
    }
  };

  const handleDeleteConversation = async (id: number) => {
    try {
      await deleteConversation.mutateAsync(id);
      if (activeConversationId === id) {
        const remaining = conversations.filter((c) => c.id !== id);
        setSelectedId(remaining[0]?.id ?? null);
      }
    } catch {
      // Error handled by query/mutation
    }
  };

  const handleSendMessage = async (params: SendMessageParams) => {
    try {
      let targetConvId = activeConversationId;
      if (!targetConvId) {
        // Auto-create new conversation if none selected
        const title =
          params.files?.[0]?.name?.replace(/\.[^/.]+$/, "") ||
          params.content?.slice(0, 30) ||
          t("fin.chat.newConversation", "New conversation");

        const conv = await createConversation.mutateAsync({ title });
        targetConvId = conv.id ?? null;
        if (targetConvId) {
          setSelectedId(targetConvId);
        }
      }

      await sendMessage.mutateAsync({
        ...params,
        conversationId: targetConvId ?? undefined,
      });
    } catch (err) {
      const errorMsg =
        err instanceof Error
          ? err.message
          : t("fin.chat.sendFailed", "Failed to send message");
      message.error(errorMsg);
    }
  };

  const handleConfirmDraft = async (
    messageId: number,
    proposal?: DraftProposal,
  ) => {
    setConfirmingMessageId(messageId);
    try {
      await confirmDraft.mutateAsync({ messageId, proposal });
    } finally {
      setConfirmingMessageId(null);
    }
  };

  const handleDiscardDraft = async (messageId: number) => {
    setDiscardingMessageId(messageId);
    try {
      await discardDraft.mutateAsync(messageId);
    } finally {
      setDiscardingMessageId(null);
    }
  };

  if (!hasPermissions) {
    return (
      <Page title={t("fin.chat.title", "Talk to Agent")}>
        <div className="mx-auto max-w-xl py-12">
          <Result
            status="403"
            title={t("common.forbidden", "Access Denied")}
            subTitle={t(
              "fin.chat.forbidden",
              "You need permission to use the financial agent and create transactions to access Talk to Agent.",
            )}
          />
        </div>
      </Page>
    );
  }

  if (isStatusLoading) {
    return (
      <Page title={t("fin.chat.title", "Talk to Agent")}>
        <div className="flex h-64 items-center justify-center">
          <Spin size="large" />
        </div>
      </Page>
    );
  }

  const isAgentDisabled = !status?.enabled || !status?.available;

  return (
    <Page title={t("fin.chat.title", "Talk to Agent")}>
      <div className="flex flex-col gap-3 h-[calc(100vh-140px)] min-h-[500px]">
        {isAgentDisabled && (
          <Alert
            type="warning"
            showIcon
            message={t("fin.chat.disabledTitle", "Financial Agent Unavailable")}
            description={
              status?.reason
                ? t(`agent.status.${status.reason}`, status.reason)
                : t(
                    "fin.chat.disabledDescription",
                    "The financial agent is currently disabled or has no active provider configured.",
                  )
            }
          />
        )}

        <div className="flex-1 rounded-lg border border-[var(--omed-border)] bg-[var(--omed-surface)] overflow-hidden shadow-xs flex flex-row min-h-0">
          {/* Desktop Sidebar */}
          <div className="hidden sm:flex shrink-0">
            <ConversationSidebar
              conversations={conversations}
              activeConversationId={activeConversationId}
              onSelectConversation={(id) => setSelectedId(id)}
              onNewConversation={handleNewConversation}
              onDeleteConversation={handleDeleteConversation}
              isLoading={isConversationsLoading}
              isCreating={createConversation.isPending}
              isDeleting={deleteConversation.isPending}
            />
          </div>

          {/* Mobile Sidebar Drawer */}
          <Drawer
            title={t("fin.chat.title", "Talk to Agent")}
            placement="left"
            open={mobileDrawerOpen}
            onClose={() => setMobileDrawerOpen(false)}
            styles={{ body: { padding: 0 } }}
          >
            <ConversationSidebar
              conversations={conversations}
              activeConversationId={activeConversationId}
              onSelectConversation={(id) => {
                setSelectedId(id);
                setMobileDrawerOpen(false);
              }}
              onNewConversation={() => {
                void handleNewConversation();
              }}
              onDeleteConversation={handleDeleteConversation}
              isLoading={isConversationsLoading}
              isCreating={createConversation.isPending}
              isDeleting={deleteConversation.isPending}
            />
          </Drawer>

          {/* Chat main pane */}
          <div className="flex-1 flex flex-col min-w-0 bg-[var(--omed-bg)] h-full">
            {/* Chat header */}
            <div className="px-4 py-2.5 bg-[var(--omed-surface)] border-b border-[var(--omed-border-soft)] flex items-center justify-between shrink-0">
              <div className="flex items-center gap-2 min-w-0">
                <Button
                  type="text"
                  icon={<MenuOutlined />}
                  onClick={() => setMobileDrawerOpen(true)}
                  className="sm:hidden"
                  aria-label={t("fin.chat.title", "Talk to Agent")}
                />
                <div className="min-w-0">
                  <h2 className="text-sm font-semibold text-[var(--omed-text)] truncate">
                    {activeConversation?.title ||
                      t("fin.chat.title", "Talk to Agent")}
                  </h2>
                  <p className="text-[11px] text-[var(--omed-text-secondary)] truncate hidden sm:block">
                    {t(
                      "fin.chat.placeholderDesc",
                      "Chat with the financial agent to analyze receipts, slips, and invoices into draft transactions.",
                    )}
                  </p>
                </div>
              </div>
            </div>

            {/* Message list area */}
            <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
              {isDetailLoading && !activeConversation ? (
                <div className="flex-1 flex items-center justify-center p-8">
                  <Spin size="small" />
                </div>
              ) : (
                <MessageList
                  messages={activeConversation?.messages || []}
                  wid={wid}
                  isAnalyzing={sendMessage.isPending}
                  accounts={accounts}
                  onConfirmDraft={handleConfirmDraft}
                  onDiscardDraft={handleDiscardDraft}
                  confirmingMessageId={confirmingMessageId}
                  discardingMessageId={discardingMessageId}
                />
              )}
            </div>

            {/* Composer docked at bottom */}
            <MessageComposer
              onSend={handleSendMessage}
              disabled={isAgentDisabled}
              isSending={sendMessage.isPending}
            />
          </div>
        </div>
      </div>
    </Page>
  );
}
