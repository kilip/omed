import * as Sentry from "@sentry/react";
import { isRouteErrorResponse } from "react-router";
import type { ApiError } from "./api";
import { env } from "./env";

let initialized = false;

export function initSentry(): void {
  if (initialized || !env.sentryDsn) {
    return;
  }
  initialized = true;
  Sentry.init({
    dsn: env.sentryDsn,
    environment: env.sentryEnvironment,
    release: env.sentryRelease || undefined,
    tracesSampleRate: env.sentryTracesSampleRate,
    tracePropagationTargets: [env.apiUrl],
    integrations: [Sentry.browserTracingIntegration()],
  });
}

export function captureErrorBoundary(error: unknown): void {
  if (!env.sentryDsn) {
    return;
  }
  if (isRouteErrorResponse(error) && error.status === 404) {
    return;
  }
  Sentry.captureException(error);
}

export function captureApiError(
  error: ApiError,
  meta?: { requestId?: string; method?: string; url?: string },
): void {
  if (!env.sentryDsn || error.status < 500) {
    return;
  }
  const requestId = meta?.requestId ?? error.requestId;
  const tags: Record<string, string> = {};
  if (requestId) {
    tags.request_id = requestId;
  }
  if (error.code) {
    tags["error.code"] = error.code;
  }
  if (meta?.method) {
    tags["http.method"] = meta.method;
  }
  if (meta?.url) {
    tags["http.url"] = meta.url;
  }
  Sentry.captureException(error, {
    tags: Object.keys(tags).length > 0 ? tags : undefined,
  });
}

export function setSentryWorkspaceTag(workspaceId?: number): void {
  if (!env.sentryDsn) {
    return;
  }
  Sentry.setTag(
    "workspace_id",
    workspaceId !== undefined ? String(workspaceId) : undefined,
  );
}

export function setSentryUserContext(
  user: { id?: number | string } | null,
  workspaceId?: number,
): void {
  if (!env.sentryDsn) {
    return;
  }
  if (!user || user.id === undefined) {
    Sentry.setUser(null);
    Sentry.setTag("workspace_id", undefined);
    return;
  }
  Sentry.setUser({ id: String(user.id) });
  Sentry.setTag(
    "workspace_id",
    workspaceId !== undefined ? String(workspaceId) : undefined,
  );
}
