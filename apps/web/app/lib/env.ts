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

export const DEFAULT_CMS_MEDIA_ALLOWED_TYPES = [
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".gif",
  ".pdf",
];

/**
 * Runtime configuration injected by `/config.js` (generated from `OMED_WEB_*`
 * env vars at container start). Values are untrusted and parsed by the
 * `read*` helpers; missing values fall back to build-time `VITE_*` env.
 */
export type RuntimeConfig = Partial<
  Record<
    | "apiUrl"
    | "publicUrl"
    | "attachmentMaxSizeMB"
    | "attachmentAllowedTypes"
    | "cmsMediaMaxSizeMB"
    | "cmsMediaAllowedTypes",
    unknown
  >
>;

declare global {
  interface Window {
    __OMED_CONFIG__?: RuntimeConfig;
  }
}

const readRuntimeConfig = (): RuntimeConfig => {
  if (typeof window === "undefined") {
    return {};
  }
  const config = window.__OMED_CONFIG__;
  return typeof config === "object" && config !== null ? config : {};
};

const runtime = readRuntimeConfig();

const pick = (runtimeValue: unknown, buildValue: unknown): unknown =>
  runtimeValue === undefined || runtimeValue === "" ? buildValue : runtimeValue;

const apiUrl = read(
  pick(runtime.apiUrl, import.meta.env.VITE_API_URL),
  "http://localhost:8001",
);

export const env = {
  apiUrl,
  publicUrl: read(
    pick(runtime.publicUrl, import.meta.env.VITE_PUBLIC_URL),
    "http://localhost:3000",
  ),
  attachmentMaxSizeMB: readNumber(
    pick(
      runtime.attachmentMaxSizeMB,
      import.meta.env.VITE_ATTACHMENT_MAX_SIZE_MB,
    ),
    10,
  ),
  attachmentAllowedTypes: readList(
    pick(
      runtime.attachmentAllowedTypes,
      import.meta.env.VITE_ATTACHMENT_ALLOWED_TYPES,
    ),
    DEFAULT_ATTACHMENT_ALLOWED_TYPES,
  ),
  cmsMediaMaxSizeMB: readNumber(
    pick(runtime.cmsMediaMaxSizeMB, import.meta.env.VITE_CMS_MEDIA_MAX_SIZE_MB),
    10,
  ),
  cmsMediaAllowedTypes: readList(
    pick(
      runtime.cmsMediaAllowedTypes,
      import.meta.env.VITE_CMS_MEDIA_ALLOWED_TYPES,
    ),
    DEFAULT_CMS_MEDIA_ALLOWED_TYPES,
  ),
};
