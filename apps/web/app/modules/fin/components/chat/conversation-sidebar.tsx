import {
  CheckOutlined,
  CloseOutlined,
  CommentOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { Button, Input, Popconfirm, Spin } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";

type Conversation = components["schemas"]["model.ConversationResponse"];

export interface ConversationSidebarProps {
  conversations: Conversation[];
  activeConversationId?: number | null;
  onSelectConversation: (id: number) => void;
  onNewConversation: () => void;
  onDeleteConversation: (id: number) => void;
  onRenameConversation?: (id: number, title: string) => Promise<void> | void;
  isLoading?: boolean;
  isCreating?: boolean;
  isDeleting?: boolean;
  isRenaming?: boolean;
}

export function ConversationSidebar({
  conversations,
  activeConversationId,
  onSelectConversation,
  onNewConversation,
  onDeleteConversation,
  onRenameConversation,
  isLoading = false,
  isCreating = false,
  isDeleting = false,
  isRenaming = false,
}: ConversationSidebarProps) {
  const { t } = useTranslation();
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editTitle, setEditTitle] = useState("");

  const handleStartRename = (conv: Conversation) => {
    if (conv.id === undefined) return;
    setEditingId(conv.id);
    setEditTitle(conv.title || "");
  };

  const handleCancelRename = () => {
    setEditingId(null);
    setEditTitle("");
  };

  const handleSaveRename = async (id: number) => {
    const trimmed = editTitle.trim();
    if (!trimmed) {
      handleCancelRename();
      return;
    }
    try {
      await onRenameConversation?.(id, trimmed);
      handleCancelRename();
    } catch {
      // Keep editing mode open on failure if needed or let caller handle
    }
  };
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
            const isEditing = conv.id !== undefined && editingId === conv.id;

            if (isEditing && convId !== undefined) {
              return (
                <div
                  key={conv.id}
                  data-testid={`conversation-item-${conv.id}`}
                  className="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs bg-[var(--omed-surface)] border border-[var(--omed-border)]"
                >
                  <Input
                    size="small"
                    autoFocus
                    value={editTitle}
                    disabled={isRenaming}
                    onChange={(e) => setEditTitle(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        void handleSaveRename(convId);
                      } else if (e.key === "Escape") {
                        e.preventDefault();
                        handleCancelRename();
                      }
                    }}
                    className="flex-1 text-xs"
                    data-testid={`conversation-rename-input-${conv.id}`}
                    aria-label={t(
                      "fin.chat.conversationTitle",
                      "Conversation title",
                    )}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<CheckOutlined className="text-xs" />}
                    loading={isRenaming}
                    onClick={() => void handleSaveRename(convId)}
                    aria-label={t("fin.chat.saveTitle", "Save title")}
                    className="p-1 h-6 w-6 text-[var(--omed-success)]"
                    data-testid={`conversation-save-button-${conv.id}`}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<CloseOutlined className="text-xs" />}
                    disabled={isRenaming}
                    onClick={handleCancelRename}
                    aria-label={t("fin.chat.cancel", "Cancel")}
                    className="p-1 h-6 w-6 text-[var(--omed-text-secondary)]"
                    data-testid={`conversation-cancel-button-${conv.id}`}
                  />
                </div>
              );
            }

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

                <div className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-0.5">
                  {onRenameConversation && (
                    <Button
                      type="text"
                      size="small"
                      icon={<EditOutlined className="text-xs" />}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleStartRename(conv);
                      }}
                      aria-label={t(
                        "fin.chat.renameConversation",
                        "Rename conversation",
                      )}
                      className="p-1 h-6 w-6 text-[var(--omed-text-secondary)] hover:text-[var(--omed-text)]"
                      data-testid={`conversation-rename-button-${conv.id}`}
                    />
                  )}
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
                      onClick={(e) => e.stopPropagation()}
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
