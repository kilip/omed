import type { paths } from "@omed/openapi";
import createClient from "openapi-fetch";
import { env } from "./env";
import { captureApiError } from "./sentry";

export const api = createClient<paths>({
  baseUrl: env.apiUrl,
  credentials: "include",
});

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly requestId?: string;

  constructor(
    message: string,
    status: number,
    code?: string,
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

/** Catch handler: a 403 becomes `fallback`, anything else is rethrown. */
export const orForbidden =
  <T>(fallback: T) =>
  (error: unknown): T => {
    if (error instanceof ApiError && error.status === 403) return fallback;
    throw error;
  };

type ErrorBody = {
  error?: { code?: string; message?: string };
  meta?: { request_id?: string };
};

function toApiError(error: unknown, response: Response): ApiError {
  const body = (error ?? {}) as ErrorBody;
  const requestId =
    typeof body.meta?.request_id === "string" && body.meta.request_id.length > 0
      ? body.meta.request_id
      : undefined;
  const apiError = new ApiError(
    body.error?.message || response.statusText || "Request failed",
    response.status,
    body.error?.code,
    requestId,
  );
  if (response.status >= 500) {
    captureApiError(apiError, {
      requestId,
      url: response.url || undefined,
    });
  }
  return apiError;
}

type Result<T> = {
  data?: { data?: T };
  error?: unknown;
  response: Response;
};

/** Returns the `data` field of the API envelope, or throws an `ApiError`. */
export async function unwrap<T>(
  request: Promise<Result<T>>,
): Promise<T | undefined> {
  const { data, error, response } = await request;
  if (error !== undefined || !response.ok) throw toApiError(error, response);
  return data?.data;
}

export async function unwrapList<T>(
  request: Promise<Result<T[]>>,
): Promise<T[]> {
  return (await unwrap(request)) ?? [];
}

export async function unwrapOne<T>(request: Promise<Result<T>>): Promise<T> {
  const value = await unwrap(request);
  if (value === undefined) throw new ApiError("Empty response", 502);
  return value;
}

export async function unwrapVoid(
  request: Promise<{ error?: unknown; response: Response }>,
): Promise<void> {
  const { error, response } = await request;
  if (error !== undefined || !response.ok) throw toApiError(error, response);
}

type PageResult<T> = {
  data?: {
    data?: T[];
    meta?: {
      cursor?: string;
    };
  };
  error?: unknown;
  response: Response;
};

export async function unwrapPage<T>(
  request: Promise<PageResult<T>>,
): Promise<{ items: T[]; cursor?: string }> {
  const { data, error, response } = await request;
  if (error !== undefined || !response.ok) throw toApiError(error, response);
  return {
    items: data?.data ?? [],
    cursor: data?.meta?.cursor || undefined,
  };
}
