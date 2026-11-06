import { PaperClipOutlined, SendOutlined } from "@ant-design/icons";
import { App, Button, Tooltip } from "antd";
import { type ChangeEvent, type KeyboardEvent, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { env } from "~/lib/env";
import type { SendMessageParams } from "../../hooks/use-agent-chat";

export interface MessageComposerProps {
  onSend: (params: SendMessageParams) => void | Promise<void>;
  disabled?: boolean;
  isSending?: boolean;
}

export function MessageComposer({
  onSend,
  disabled = false,
  isSending = false,
}: MessageComposerProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [content, setContent] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendText();
    }
  };

  const handleSendText = () => {
    const trimmed = content.trim();
    if (!trimmed || disabled || isSending) return;
    setContent("");
    void onSend({ content: trimmed });
  };

  const handleAttachClick = () => {
    if (disabled || isSending) return;
    fileInputRef.current?.click();
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    // Reset file input value so re-selecting the exact same file fires onChange
    e.target.value = "";
    if (!file) return;

    // Validate file type
    const isImage = file.type.startsWith("image/");
    const isPdf = file.type === "application/pdf";
    if (!isImage && !isPdf) {
      message.error(
        t("fin.chat.invalidFileType", "Only images and PDF files are allowed"),
      );
      return;
    }

    // Validate size (max 25MB)
    const maxBytes = env.attachmentMaxSizeMB * 1024 * 1024;
    if (file.size > maxBytes) {
      message.error(
        t("fin.chat.fileTooLarge", {
          maxSize: env.attachmentMaxSizeMB,
          defaultValue: `File exceeds maximum size of ${env.attachmentMaxSizeMB}MB`,
        }),
      );
      return;
    }

    // Auto-send immediately on selection
    void onSend({ files: [file] });
  };

  return (
    <div
      data-testid="message-composer"
      className="p-3 bg-[var(--omed-surface)] border-t border-[var(--omed-border-soft)]"
    >
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*,application/pdf"
        className="hidden"
        onChange={handleFileChange}
      />

      <div className="flex items-end gap-2 rounded-lg border border-[var(--omed-border)] bg-[var(--omed-bg)] p-2 focus-within:border-[var(--omed-primary)] transition-colors">
        <Tooltip
          title={t(
            "fin.chat.attachTooltip",
            "Attach image or PDF (auto-sends)",
          )}
        >
          <Button
            type="text"
            icon={<PaperClipOutlined className="text-base" />}
            onClick={handleAttachClick}
            disabled={disabled || isSending}
            aria-label={t("fin.chat.attachFile", "Attach file")}
            className="shrink-0 text-[var(--omed-text-secondary)] hover:text-[var(--omed-primary)]"
          />
        </Tooltip>

        <textarea
          rows={1}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled || isSending}
          placeholder={t(
            "fin.chat.inputPlaceholder",
            "Type a message... (Press Enter to send, Shift+Enter for new line)",
          )}
          className="flex-1 resize-none bg-transparent py-1 px-1 text-sm text-[var(--omed-text)] placeholder-[var(--omed-text-muted)] focus:outline-none max-h-32 min-h-[24px]"
        />

        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={handleSendText}
          disabled={disabled || isSending || !content.trim()}
          loading={isSending}
          aria-label={t("fin.chat.send", "Send")}
          className="shrink-0"
        />
      </div>
    </div>
  );
}
