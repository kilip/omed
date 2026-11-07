import {
  CloseOutlined,
  FilePdfOutlined,
  PaperClipOutlined,
  PictureOutlined,
  SendOutlined,
} from "@ant-design/icons";
import { App, Button, Tooltip } from "antd";
import { type ChangeEvent, type KeyboardEvent, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { SendMessageParams } from "../../hooks/use-agent-chat";
import { formatFileSize } from "../transaction-attachments";

export const MAX_CHAT_ATTACHMENT_BYTES = 500 * 1024;
export interface MessageComposerHandle {
  stageFile: (file: File) => void;
  removeStagedFile: () => void;
  setContent: (content: string) => void;
  send: () => void;
  submitKeyDown: (e: {
    key: string;
    shiftKey?: boolean;
    preventDefault?: () => void;
  }) => void;
}

export interface MessageComposerProps {
  onSend: (params: SendMessageParams) => void | Promise<void>;
  disabled?: boolean;
  isSending?: boolean;
  initialContent?: string;
  initialStagedFile?: File | null;
  onStagedFileChange?: (file: File | null) => void;
  composerRef?: { current: MessageComposerHandle | null };
}

export function detectMimeType(file: { name: string; type?: string }): string {
  const type = file.type?.trim();
  if (type && type !== "application/octet-stream") {
    return type;
  }
  const lastDot = file.name.lastIndexOf(".");
  if (lastDot === -1) {
    return type || "";
  }
  const ext = file.name.slice(lastDot).toLowerCase();
  switch (ext) {
    case ".png":
      return "image/png";
    case ".jpg":
    case ".jpeg":
      return "image/jpeg";
    case ".webp":
      return "image/webp";
    case ".pdf":
      return "application/pdf";
    default:
      return type || "";
  }
}

export function MessageComposer({
  onSend,
  disabled = false,
  isSending = false,
  initialContent = "",
  initialStagedFile = null,
  onStagedFileChange,
  composerRef,
}: MessageComposerProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [content, setContent] = useState(initialContent);
  const [stagedFile, setStagedFileState] = useState<File | null>(
    initialStagedFile,
  );
  const fileInputRef = useRef<HTMLInputElement>(null);

  const setStagedFile = (file: File | null) => {
    setStagedFileState(file);
    onStagedFileChange?.(file);
  };

  const handleSend = () => {
    const trimmed = content.trim();
    if ((!trimmed && !stagedFile) || disabled || isSending) return;
    const filesToSend = stagedFile ? [stagedFile] : undefined;
    const contentToSend = trimmed || undefined;
    setContent("");
    setStagedFile(null);
    void onSend({
      content: contentToSend,
      files: filesToSend,
    });
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault?.();
      handleSend();
    }
  };

  const handleAttachClick = () => {
    if (disabled || isSending) return;
    fileInputRef.current?.click();
  };

  const processAndStageFile = (file: File) => {
    // Validate file type
    const mimeType = detectMimeType(file);
    const isImage = mimeType.startsWith("image/");
    const isPdf = mimeType === "application/pdf";
    if (!isImage && !isPdf) {
      message.error(
        t("fin.chat.invalidFileType", "Only images and PDF files are allowed"),
      );
      return;
    }

    // Validate size (max 500KB)
    if (file.size > MAX_CHAT_ATTACHMENT_BYTES) {
      message.error(
        t("fin.chat.fileTooLarge", {
          maxSize: "500KB",
          defaultValue: "File exceeds maximum size of 500KB",
        }),
      );
      return;
    }

    let fileToStage = file;
    if ((!file.type || file.type === "application/octet-stream") && mimeType) {
      try {
        if (typeof File !== "undefined" && file instanceof File) {
          fileToStage = new File([file], file.name, { type: mimeType });
        } else if (file && typeof file === "object") {
          try {
            Object.defineProperty(file, "type", {
              value: mimeType,
              configurable: true,
              writable: true,
            });
          } catch {
            // Ignore if sealed/frozen
          }
        }
      } catch {
        fileToStage = file;
      }
    }

    setStagedFile(fileToStage);
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    // Reset file input value so re-selecting the exact same file fires onChange
    e.target.value = "";
    if (!file) return;
    processAndStageFile(file);
  };

  if (composerRef) {
    composerRef.current = {
      stageFile: processAndStageFile,
      removeStagedFile: () => setStagedFile(null),
      setContent,
      send: handleSend,
      submitKeyDown: (e) => {
        handleKeyDown(e as KeyboardEvent<HTMLTextAreaElement>);
      },
    };
  }

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

      {stagedFile && (
        <div className="mb-2 flex items-center">
          <div
            data-testid="staged-file-preview"
            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-[var(--omed-border)] bg-[var(--omed-surface-muted)] text-xs text-[var(--omed-text)] max-w-full"
          >
            {detectMimeType(stagedFile).startsWith("image/") ? (
              <PictureOutlined
                style={{ color: "var(--omed-primary)" }}
                className="shrink-0"
              />
            ) : (
              <FilePdfOutlined
                style={{ color: "var(--omed-danger)" }}
                className="shrink-0"
              />
            )}
            <span
              className="truncate max-w-[200px] font-medium"
              title={stagedFile.name}
            >
              {stagedFile.name}
            </span>
            <span className="text-[var(--omed-text-secondary)] shrink-0">
              ({formatFileSize(stagedFile.size)})
            </span>
            <Button
              type="text"
              size="small"
              icon={<CloseOutlined className="text-xs" />}
              onClick={() => setStagedFile(null)}
              aria-label={t("fin.chat.removeAttachment", "Remove attachment")}
              className="shrink-0 text-[var(--omed-text-secondary)] hover:text-[var(--omed-danger)] -mr-1 h-5 w-5 min-w-5 p-0 flex items-center justify-center"
            />
          </div>
        </div>
      )}

      <div className="flex items-end gap-2 rounded-lg border border-[var(--omed-border)] bg-[var(--omed-bg)] p-2 focus-within:border-[var(--omed-primary)] transition-colors">
        <Tooltip
          title={t("fin.chat.attachTooltip", "Attach image or PDF (max 500KB)")}
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
          onClick={handleSend}
          disabled={disabled || isSending || (!content.trim() && !stagedFile)}
          loading={isSending}
          aria-label={t("fin.chat.send", "Send")}
          className="shrink-0"
        />
      </div>
    </div>
  );
}
