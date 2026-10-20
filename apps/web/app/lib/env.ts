const read = (value: unknown, fallback: string): string =>
  typeof value === "string" && value.length > 0 ? value : fallback;

const readNumber = (value: unknown, fallback: number): number => {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return value;
  }
  if (typeof value === "string" && value.trim().length > 0) {
    const parsed = Number(value);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }
  return fallback;
};

const readList = (value: unknown, fallback: string[]): string[] => {
  if (typeof value === "string" && value.trim().length > 0) {
    const items = value
      .split(",")
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean)
      .map((s) => (s.startsWith(".") ? s : `.${s}`));
    if (items.length > 0) {
      return items;
    }
  }
  return fallback;
};

export const DEFAULT_ATTACHMENT_ALLOWED_TYPES = [
  ".pdf",
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".csv",
  ".xls",
  ".xlsx",
];

const apiUrl = read(import.meta.env.VITE_API_URL, "http://localhost:8001");

export const env = {
  apiUrl,
  publicUrl: read(import.meta.env.VITE_PUBLIC_URL, "http://localhost:3000"),
  attachmentMaxSizeMB: readNumber(
    import.meta.env.VITE_ATTACHMENT_MAX_SIZE_MB,
    10,
  ),
  attachmentAllowedTypes: readList(
    import.meta.env.VITE_ATTACHMENT_ALLOWED_TYPES,
    DEFAULT_ATTACHMENT_ALLOWED_TYPES,
  ),
};
