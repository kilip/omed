import {
  DownloadOutlined,
  FilePdfOutlined,
  FileUnknownOutlined,
  PictureOutlined,
  RobotOutlined,
  UserOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { Avatar, Button, Spin } from "antd";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { env } from "~/lib/env";
import type {
  DraftProposal,
  MessageResponse,
} from "../../hooks/use-agent-chat";
import { formatFileSize } from "../transaction-attachments";
import { DraftPreviewCard } from "./draft-preview-card";

type Account = components["schemas"]["model.AccountResponse"];
type Attachment = components["schemas"]["model.AttachmentResponse"];

export function extractDraftProposal(parts: unknown): DraftProposal | null {
  if (!parts) return null;
  let target = parts;
  if (typeof target === "string") {
    try {
      target = JSON.parse(target);
    } catch {
      return null;
    }
  }
  if (typeof target === "object" && target !== null) {
    const obj = target as Record<string, unknown>;
    if ("proposal" in obj && obj.proposal && typeof obj.proposal === "object") {
      return obj.proposal as DraftProposal;
    }
    if ("kind" in obj || "items" in obj || "entries" in obj) {
      return target as DraftProposal;
    }
  }
  return null;
}

export interface MessageListProps {
  messages: MessageResponse[];
  wid: number;
  isAnalyzing?: boolean;
  accounts?: Account[];
  onConfirmDraft?: (
    messageId: number,
    proposal?: DraftProposal,
  ) => void | Promise<void>;
  onDiscardDraft?: (messageId: number) => void | Promise<void>;
  confirmingMessageId?: number | null;
  discardingMessageId?: number | null;
}

export function MessageList({
  messages,
  wid,
  isAnalyzing = false,
  accounts = [],
  onConfirmDraft,
  onDiscardDraft,
  confirmingMessageId = null,
  discardingMessageId = null,
}: MessageListProps) {
  const { t } = useTranslation();
  const bottomRef = useRef<HTMLDivElement>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: scroll triggered by message count or analyzing state
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length, isAnalyzing]);

  const handleDownloadAttachment = async (att: Attachment) => {
    if (!att.id) return;
    try {
      const res = await fetch(
        `${env.apiUrl}/workspaces/${wid}/agent/chat/attachments/${att.id}/download`,
        {
          method: "GET",
          credentials: "include",
        },
      );
      if (!res.ok) throw new Error("Download failed");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = att.fileName || `attachment-${att.id}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      // Fallback direct link
      window.open(
        `${env.apiUrl}/workspaces/${wid}/agent/chat/attachments/${att.id}/download`,
        "_blank",
      );
    }
  };

  if (messages.length === 0 && !isAnalyzing) {
    return (
      <div
        data-testid="message-list-empty"
        className="flex-1 flex flex-col items-center justify-center p-8 text-center"
      >
        <div className="w-12 h-12 rounded-full bg-[var(--omed-tint)] flex items-center justify-center mb-3">
          <RobotOutlined
            className="text-2xl"
            style={{ color: "var(--omed-primary-text)" }}
          />
        </div>
        <h3 className="text-base font-semibold text-[var(--omed-text)] mb-1">
          {t("fin.chat.emptyMessagesTitle", "How can I help you today?")}
        </h3>
        <p className="text-xs text-[var(--omed-text-secondary)] max-w-sm">
          {t(
            "fin.chat.emptyMessagesDesc",
            "Send a message or attach a receipt, invoice, or ATM slip to automatically create a draft transaction.",
          )}
        </p>
      </div>
    );
  }

  return (
    <div
      data-testid="message-list"
      className="flex-1 overflow-y-auto p-4 space-y-4"
    >
      {messages.map((msg) => {
        const isUser = msg.role === "user";
        const draftProposal = !isUser ? extractDraftProposal(msg.parts) : null;

        return (
          <div
            key={msg.id}
            data-testid={`message-${msg.id}`}
            className={`flex gap-3 ${isUser ? "justify-end" : "justify-start"}`}
          >
            {!isUser && (
              <Avatar
                size="small"
                icon={<RobotOutlined />}
                className="shrink-0 mt-0.5"
                style={{
                  backgroundColor: "var(--omed-tint)",
                  color: "var(--omed-primary-text)",
                }}
              />
            )}

            <div
              className={`max-w-[85%] sm:max-w-[75%] space-y-2 ${
                isUser ? "items-end" : "items-start"
              }`}
            >
              {/* Attachments if any */}
              {msg.attachments && msg.attachments.length > 0 && (
                <div
                  className={`flex flex-wrap gap-2 ${
                    isUser ? "justify-end" : "justify-start"
                  }`}
                >
                  {msg.attachments.map((att) => {
                    const isImg = att.mime?.startsWith("image/");
                    const isPdf = att.mime === "application/pdf";

                    return (
                      <div
                        key={att.id}
                        data-testid={`attachment-${att.id}`}
                        className="flex items-center gap-2 p-2 rounded-lg border border-[var(--omed-border)] bg-[var(--omed-surface)] shadow-xs max-w-xs"
                      >
                        <div className="text-lg shrink-0">
                          {isImg ? (
                            <PictureOutlined
                              style={{ color: "var(--omed-primary)" }}
                            />
                          ) : isPdf ? (
                            <FilePdfOutlined
                              style={{ color: "var(--omed-danger)" }}
                            />
                          ) : (
                            <FileUnknownOutlined
                              style={{ color: "var(--omed-text-secondary)" }}
                            />
                          )}
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="text-xs font-medium text-[var(--omed-text)] truncate">
                            {att.fileName}
                          </div>
                          <div className="text-[10px] text-[var(--omed-text-secondary)]">
                            {formatFileSize(att.size || 0)}
                          </div>
                        </div>
                        <Button
                          type="text"
                          size="small"
                          icon={<DownloadOutlined />}
                          onClick={() => handleDownloadAttachment(att)}
                          aria-label={t("fin.chat.download", "Download")}
                        />
                      </div>
                    );
                  })}
                </div>
              )}

              {/* Message text bubble */}
              {msg.content && (
                <div
                  className={`rounded-xl px-3.5 py-2.5 text-sm whitespace-pre-wrap leading-relaxed shadow-xs ${
                    isUser
                      ? "bg-[var(--omed-primary)] text-white rounded-tr-xs"
                      : "bg-[var(--omed-surface)] text-[var(--omed-text)] border border-[var(--omed-border-soft)] rounded-tl-xs"
                  }`}
                  style={
                    isUser
                      ? {
                          backgroundColor: "var(--omed-primary)",
                          color: "var(--omed-on-primary)",
                        }
                      : undefined
                  }
                >
                  {msg.content}
                </div>
              )}

              {/* Draft Proposal Card if assistant proposed one */}
              {draftProposal && (
                <DraftPreviewCard
                  proposal={draftProposal}
                  status={msg.draftStatus || "proposed"}
                  transactionId={msg.confirmedTransactionId}
                  onConfirm={(editedProposal) => {
                    if (msg.id !== undefined) {
                      onConfirmDraft?.(msg.id, editedProposal);
                    }
                  }}
                  onDiscard={() => {
                    if (msg.id !== undefined) {
                      onDiscardDraft?.(msg.id);
                    }
                  }}
                  isConfirming={confirmingMessageId === msg.id}
                  isDiscarding={discardingMessageId === msg.id}
                  accounts={accounts}
                  wid={wid}
                />
              )}
            </div>

            {isUser && (
              <Avatar
                size="small"
                icon={<UserOutlined />}
                className="shrink-0 mt-0.5"
                style={{
                  backgroundColor: "var(--omed-surface-muted)",
                  color: "var(--omed-text-secondary)",
                }}
              />
            )}
          </div>
        );
      })}

      {/* Analyzing spinner */}
      {isAnalyzing && (
        <div
          data-testid="analyzing-indicator"
          className="flex items-center gap-3"
        >
          <Avatar
            size="small"
            icon={<RobotOutlined />}
            className="shrink-0"
            style={{
              backgroundColor: "var(--omed-tint)",
              color: "var(--omed-primary-text)",
            }}
          />
          <div className="flex items-center gap-2 rounded-xl bg-[var(--omed-surface)] border border-[var(--omed-border-soft)] px-3.5 py-2 text-xs text-[var(--omed-text-secondary)] shadow-xs">
            <Spin size="small" />
            <span>{t("fin.chat.analyzing", "Agent is analyzing...")}</span>
          </div>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}
