import {
  DeleteOutlined,
  DownloadOutlined,
  EyeOutlined,
  FileExcelOutlined,
  FileImageOutlined,
  FilePdfOutlined,
  PaperClipOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient } from "@tanstack/react-query";
import { App, Button, Modal, Popconfirm, Spin, Upload } from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ApiError, api, unwrapOne, unwrapVoid } from "~/lib/api";
import { env } from "~/lib/env";
import { useAction } from "~/lib/use-action";
import { transactionKeys } from "../queries";

export type AttachmentFile = components["schemas"]["model.FileResponse"];

export type AttachmentValidationResult =
  | { valid: true }
  | { valid: false; reason: "empty" | "too_large" | "invalid_type" };

export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

export function getFileExtension(filename: string): string {
  const trimmed = filename.trim().toLowerCase();
  const lastDot = trimmed.lastIndexOf(".");
  if (lastDot <= 0 || lastDot === trimmed.length - 1) return "";
  return trimmed.slice(lastDot);
}

export function validateAttachmentFile(
  file: { name: string; size: number },
  allowedTypes: string[] = env.attachmentAllowedTypes,
  maxSizeMB: number = env.attachmentMaxSizeMB,
): AttachmentValidationResult {
  if (file.size <= 0) {
    return { valid: false, reason: "empty" };
  }
  const maxBytes = maxSizeMB * 1024 * 1024;
  if (file.size > maxBytes) {
    return { valid: false, reason: "too_large" };
  }
  const ext = getFileExtension(file.name);
  const normalizedAllowed = allowedTypes.map((item) => {
    const clean = item.trim().toLowerCase();
    return clean.startsWith(".") ? clean : `.${clean}`;
  });
  if (!ext || !normalizedAllowed.includes(ext)) {
    return { valid: false, reason: "invalid_type" };
  }
  return { valid: true };
}

export function isPreviewableAttachment(
  file: Pick<AttachmentFile, "name" | "mimeType">,
): boolean {
  const mime = (file.mimeType ?? "").toLowerCase();
  const ext = getFileExtension(file.name ?? "");
  return (
    mime.startsWith("image/") ||
    mime === "application/pdf" ||
    [".png", ".jpg", ".jpeg", ".webp", ".pdf"].includes(ext)
  );
}

export function isImageAttachment(
  file: Pick<AttachmentFile, "name" | "mimeType">,
): boolean {
  const mime = (file.mimeType ?? "").toLowerCase();
  const ext = getFileExtension(file.name ?? "");
  return (
    mime.startsWith("image/") ||
    [".png", ".jpg", ".jpeg", ".webp"].includes(ext)
  );
}

export async function uploadTransactionAttachment(
  wid: number,
  transactionId: number,
  file: File,
): Promise<AttachmentFile> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("name", file.name);

  return unwrapOne(
    api.POST("/workspaces/{wid}/fin/transactions/{id}/attachments", {
      params: { path: { wid, id: transactionId } },
      body: formData as unknown as { file: string; name?: string },
      bodySerializer: (body) => body as unknown as BodyInit,
    }),
  );
}

async function fetchAttachmentBlob(
  wid: number,
  fileId: number,
  fallbackMimeType?: string,
): Promise<Blob> {
  const response = await fetch(
    `${env.apiUrl}/workspaces/${wid}/files/${fileId}/download`,
    {
      method: "GET",
      credentials: "include",
    },
  );
  if (!response.ok) {
    throw new ApiError(
      response.statusText || "Download failed",
      response.status,
    );
  }
  const buffer = await response.arrayBuffer();
  const contentType =
    response.headers.get("content-type") ||
    fallbackMimeType ||
    "application/octet-stream";
  return new Blob([buffer], { type: contentType });
}

function AttachmentIcon({
  file,
}: {
  file: Pick<AttachmentFile, "name" | "mimeType">;
}) {
  const ext = getFileExtension(file.name ?? "");
  if (ext === ".pdf" || file.mimeType === "application/pdf") {
    return <FilePdfOutlined style={{ color: "var(--omed-danger)" }} />;
  }
  if (isImageAttachment(file)) {
    return <FileImageOutlined style={{ color: "var(--omed-info)" }} />;
  }
  if ([".csv", ".xls", ".xlsx"].includes(ext)) {
    return <FileExcelOutlined style={{ color: "var(--omed-success)" }} />;
  }
  return <PaperClipOutlined style={{ color: "var(--omed-text-secondary)" }} />;
}

export type TransactionAttachmentsSectionProps = {
  wid: number;
  transactionId: number;
  attachments: AttachmentFile[];
  canModify: boolean;
};

export function TransactionAttachmentsSection({
  wid,
  transactionId,
  attachments,
  canModify,
}: TransactionAttachmentsSectionProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();

  const [previewFile, setPreviewFile] = useState<AttachmentFile | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  const allowedTypesLabel = env.attachmentAllowedTypes.join(", ");

  const validateAndNotify = (file: File): boolean => {
    const check = validateAttachmentFile(file);
    if (!check.valid) {
      if (check.reason === "too_large") {
        message.error(
          t("fin.transactions.attachments.fileTooLarge", {
            maxSize: env.attachmentMaxSizeMB,
          }),
        );
      } else {
        message.error(
          t("fin.transactions.attachments.invalidFileType", {
            types: allowedTypesLabel,
          }),
        );
      }
      return false;
    }
    return true;
  };

  const handleUpload = async (file: File) => {
    if (!validateAndNotify(file)) return;

    await run(async () => {
      await uploadTransactionAttachment(wid, transactionId, file);
      await queryClient.invalidateQueries({
        queryKey: transactionKeys.detail(wid, transactionId),
      });
    }, "common.saved");
  };

  const handleDelete = async (fileId?: number) => {
    if (!fileId) return;
    await run(async () => {
      await unwrapVoid(
        api.DELETE(
          "/workspaces/{wid}/fin/transactions/{id}/attachments/{fileId}",
          {
            params: { path: { wid, id: transactionId, fileId } },
          },
        ),
      );
      await queryClient.invalidateQueries({
        queryKey: transactionKeys.detail(wid, transactionId),
      });
    }, "common.deleted");
  };

  const handleDownload = async (att: AttachmentFile) => {
    if (!att.id) return;
    try {
      const blob = await fetchAttachmentBlob(wid, att.id, att.mimeType);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = att.name || `attachment-${att.id}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch {
      message.error(t("common.error"));
    }
  };

  const handlePreview = async (att: AttachmentFile) => {
    if (!att.id) return;
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
      setPreviewUrl(null);
    }
    setPreviewFile(att);
    setPreviewLoading(true);
    try {
      const blob = await fetchAttachmentBlob(wid, att.id, att.mimeType);
      setPreviewUrl(URL.createObjectURL(blob));
    } catch {
      message.error(t("common.error"));
      setPreviewFile(null);
    } finally {
      setPreviewLoading(false);
    }
  };

  const closePreview = () => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreviewUrl(null);
    setPreviewFile(null);
  };

  return (
    <div
      className="pt-4 border-t"
      style={{ borderColor: "var(--omed-border)" }}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
        <div>
          <h3 className="text-base font-semibold m-0">
            {t("fin.transactions.attachments.title", "Attachments")}
          </h3>
          {canModify && (
            <span
              className="text-xs block"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {t("fin.transactions.attachments.hint", {
                types: allowedTypesLabel,
                maxSize: env.attachmentMaxSizeMB,
              })}
            </span>
          )}
        </div>

        {canModify && (
          <Upload
            accept={env.attachmentAllowedTypes.join(",")}
            showUploadList={false}
            beforeUpload={(file) => {
              void handleUpload(file);
              return false;
            }}
          >
            <Button icon={<UploadOutlined />} loading={pending}>
              {t("fin.transactions.attachments.upload", "Upload attachment")}
            </Button>
          </Upload>
        )}
      </div>

      {attachments.length === 0 ? (
        <p
          className="text-sm m-0"
          style={{ color: "var(--omed-text-secondary)" }}
        >
          {t("fin.transactions.attachments.empty", "No attachments yet.")}
        </p>
      ) : (
        <ul className="m-0 p-0 list-none flex flex-col gap-2">
          {attachments.map((att) => (
            <li
              key={att.id}
              data-testid="transaction-attachment-item"
              className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 rounded"
              style={{
                border: "1px solid var(--omed-border)",
                background: "var(--omed-surface-muted)",
              }}
            >
              <div className="flex items-center gap-2 min-w-0">
                <AttachmentIcon file={att} />
                <span className="font-medium text-sm truncate">{att.name}</span>
                <span
                  className="text-xs shrink-0"
                  style={{ color: "var(--omed-text-secondary)" }}
                >
                  ({formatFileSize(att.size ?? 0)})
                </span>
              </div>

              <div className="flex items-center gap-1.5">
                {isPreviewableAttachment(att) && (
                  <Button
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => void handlePreview(att)}
                  >
                    {t("fin.transactions.attachments.preview", "Preview")}
                  </Button>
                )}

                <Button
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={() => void handleDownload(att)}
                >
                  {t("fin.transactions.attachments.download", "Download")}
                </Button>

                {canModify && (
                  <Popconfirm
                    title={t(
                      "fin.transactions.attachments.deleteConfirm",
                      "Delete this attachment?",
                    )}
                    onConfirm={() => void handleDelete(att.id)}
                  >
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      loading={pending}
                    >
                      {t(
                        "fin.transactions.attachments.delete",
                        "Delete attachment",
                      )}
                    </Button>
                  </Popconfirm>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <Modal
        open={Boolean(previewFile)}
        title={previewFile?.name}
        footer={null}
        onCancel={closePreview}
        width={760}
        destroyOnHidden
      >
        {previewLoading ? (
          <div className="flex justify-center py-12">
            <Spin />
          </div>
        ) : previewFile && previewUrl ? (
          isImageAttachment(previewFile) ? (
            <div className="flex justify-center">
              <img
                src={previewUrl}
                alt={previewFile.name}
                className="max-h-[70vh] max-w-full object-contain"
              />
            </div>
          ) : (
            <iframe
              src={previewUrl}
              title={previewFile.name}
              className="w-full h-[70vh] border-0"
            />
          )
        ) : null}
      </Modal>
    </div>
  );
}

export type PendingAttachmentsPickerProps = {
  files: File[];
  onChange: (files: File[]) => void;
  disabled?: boolean;
};

export function PendingAttachmentsPicker({
  files,
  onChange,
  disabled,
}: PendingAttachmentsPickerProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const allowedTypesLabel = env.attachmentAllowedTypes.join(", ");

  const handleSelect = (file: File) => {
    const check = validateAttachmentFile(file);
    if (!check.valid) {
      if (check.reason === "too_large") {
        message.error(
          t("fin.transactions.attachments.fileTooLarge", {
            maxSize: env.attachmentMaxSizeMB,
          }),
        );
      } else {
        message.error(
          t("fin.transactions.attachments.invalidFileType", {
            types: allowedTypesLabel,
          }),
        );
      }
      return;
    }
    const exists = files.some(
      (f) => f.name === file.name && f.size === file.size,
    );
    if (exists) {
      return;
    }
    onChange([...files, file]);
  };

  const handleRemove = (target: File) => {
    onChange(files.filter((f) => f !== target));
  };

  return (
    <div
      className="pt-2 border-t"
      style={{ borderColor: "var(--omed-border)" }}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
        <div>
          <span className="text-sm font-semibold block">
            {t("fin.transactions.attachments.title", "Attachments")}
          </span>
          <span
            className="text-xs block"
            style={{ color: "var(--omed-text-secondary)" }}
          >
            {t("fin.transactions.attachments.hint", {
              types: allowedTypesLabel,
              maxSize: env.attachmentMaxSizeMB,
            })}
          </span>
        </div>

        <Upload
          accept={env.attachmentAllowedTypes.join(",")}
          multiple
          showUploadList={false}
          disabled={disabled}
          beforeUpload={(file) => {
            handleSelect(file);
            return false;
          }}
        >
          <Button icon={<UploadOutlined />} disabled={disabled}>
            {t("fin.transactions.attachments.upload", "Upload attachment")}
          </Button>
        </Upload>
      </div>

      {files.length > 0 && (
        <ul className="m-0 p-0 list-none flex flex-col gap-1.5">
          {files.map((file) => (
            <li
              key={`${file.name}-${file.size}`}
              data-testid="pending-attachment-item"
              className="flex items-center justify-between gap-2 px-3 py-1.5 rounded"
              style={{
                border: "1px solid var(--omed-border)",
                background: "var(--omed-surface-muted)",
              }}
            >
              <div className="flex items-center gap-2 min-w-0">
                <AttachmentIcon
                  file={{ name: file.name, mimeType: file.type }}
                />
                <span className="text-sm truncate">{file.name}</span>
                <span
                  className="text-xs shrink-0"
                  style={{ color: "var(--omed-text-secondary)" }}
                >
                  ({formatFileSize(file.size)})
                </span>
              </div>

              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                disabled={disabled}
                onClick={() => handleRemove(file)}
              >
                {t("common.delete", "Delete")}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
