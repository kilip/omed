import type { paths } from "@omed/openapi";
import createClient from "openapi-fetch";
import { env } from "./env";

export const api = createClient<paths>({
  baseUrl: env.apiUrl,
  credentials: "include",
});

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/** Catch handler: a 403 becomes `fallback`, anything else is rethrown. */
export const orForbidden =
  <T>(fallback: T) =>
  (error: unknown): T => {
    if (error instanceof ApiError && error.status === 403) return fallback;
    throw error;
  };

type ErrorBody = { error?: { code?: string; message?: string } };

function toApiError(error: unknown, response: Response): ApiError {
  const body = (error ?? {}) as ErrorBody;
  return new ApiError(
    body.error?.message || response.statusText || "Request failed",
    response.status,
    body.error?.code,
  );
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
