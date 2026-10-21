import { DEFAULT_CMS_MEDIA_ALLOWED_TYPES, env } from "~/lib/env";

export type MediaValidationResult =
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

export function isImageMedia(mimeType?: string, name?: string): boolean {
  const mime = (mimeType ?? "").toLowerCase();
  const ext = getFileExtension(name ?? "");
  return (
    mime.startsWith("image/") ||
    [".png", ".jpg", ".jpeg", ".webp", ".gif"].includes(ext)
  );
}

export function validateMediaFile(
  file: { name: string; size: number },
  allowedTypes: string[] = env.cmsMediaAllowedTypes ||
    DEFAULT_CMS_MEDIA_ALLOWED_TYPES,
  maxSizeMB: number = env.cmsMediaMaxSizeMB || 10,
): MediaValidationResult {
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

export function getPublicMediaUrl(wid: number, mediaId: number): string {
  return `${env.apiUrl}/public/workspaces/${wid}/cms/media/${mediaId}`;
}
