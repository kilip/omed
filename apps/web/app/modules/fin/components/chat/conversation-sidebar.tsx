import {
  CommentOutlined,
  DeleteOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { Button, Popconfirm, Spin } from "antd";
import { useTranslation } from "react-i18next";

type Conversation = components["schemas"]["model.ConversationResponse"];

export interface ConversationSidebarProps {
  conversations: Conversation[];
  activeConversationId?: number | null;
  onSelectConversation: (id: number) => void;
  onNewConversation: () => void;
  onDeleteConversation: (id: number) => void;
  isLoading?: boolean;
  isCreating?: boolean;
  isDeleting?: boolean;
}

export function ConversationSidebar({
  conversations,
  activeConversationId,
  onSelectConversation,
  onNewConversation,
  onDeleteConversation,
  isLoading = false,
  isCreating = false,
  isDeleting = false,
}: ConversationSidebarProps) {
  const { t } = useTranslation();

  return (
    <div
      data-testid="conversation-sidebar"
      className="w-full sm:w-72 md:w-80 border-r border-[var(--omed-border-soft)] bg-[var(--omed-surface-muted)] flex flex-col h-full shrink-0"
    >
      {/* Sidebar header with New Chat button */}
      <div className="p-3 border-b border-[var(--omed-border-soft)] flex items-center justify-between gap-2">
        <span className="text-xs font-semibold text-[var(--omed-text-secondary)] uppercase tracking-wider">
          {t("fin.chat.title", "Talk to Agent")}
        </span>
        <Button
          type="primary"
          size="small"
          icon={<PlusOutlined />}
          onClick={onNewConversation}
          loading={isCreating}
          className="text-xs"
        >
          {t("fin.chat.newConversation", "New conversation")}
        </Button>
      </div>

      {/* Conversations list */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {isLoading ? (
          <div className="flex items-center justify-center p-8">
            <Spin size="small" />
          </div>
        ) : conversations.length === 0 ? (
          <div className="p-6 text-center text-xs text-[var(--omed-text-muted)]">
            {t("fin.chat.emptyConversations", "No conversations yet")}
          </div>
        ) : (
          conversations.map((conv) => {
            const isActive = conv.id === activeConversationId;
            const convId = conv.id;
            return (
              <div
                key={conv.id}
                data-testid={`conversation-item-${conv.id}`}
                className={`group flex items-center justify-between gap-1 px-2 py-1.5 rounded-lg text-xs transition-colors ${
                  isActive
                    ? "bg-[var(--omed-tint)] text-[var(--omed-primary-text)] font-semibold"
                    : "text-[var(--omed-text)] hover:bg-[var(--omed-surface)]"
                }`}
              >
                <button
                  type="button"
                  onClick={() => {
                    if (convId !== undefined) {
                      onSelectConversation(convId);
                    }
                  }}
                  className="flex items-center gap-2 min-w-0 flex-1 text-left bg-transparent border-0 p-1 cursor-pointer"
                >
                  <CommentOutlined
                    className="shrink-0 text-sm"
                    style={{
                      color: isActive
                        ? "var(--omed-primary-text)"
                        : "var(--omed-text-secondary)",
                    }}
                  />
                  <span className="truncate">
                    {conv.title ||
                      t("fin.chat.newConversation", "New conversation")}
                  </span>
                </button>

                <div className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Popconfirm
                    title={t(
                      "fin.chat.deleteConversationConfirm",
                      "Are you sure you want to delete this conversation?",
                    )}
                    onConfirm={() => {
                      if (convId !== undefined) {
                        onDeleteConversation(convId);
                      }
                    }}
                    okText={t("fin.chat.delete", "Delete")}
                    cancelText={t("fin.chat.cancel", "Cancel")}
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined className="text-xs" />}
                      disabled={isDeleting}
                      aria-label={t(
                        "fin.chat.deleteConversation",
                        "Delete conversation",
                      )}
                      className="p-1 h-6 w-6"
                    />
                  </Popconfirm>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
