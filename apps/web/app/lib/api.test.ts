import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, unwrap, unwrapPage, unwrapVoid } from "./api";
import { captureApiError } from "./sentry";

vi.mock("./sentry", () => ({
  captureApiError: vi.fn(),
}));
describe("unwrapPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("unwraps items and cursor from successful response", async () => {
    const res = Promise.resolve({
      data: {
        data: [{ id: 1 }, { id: 2 }],
        meta: { cursor: "123" },
      },
      response: new Response(null, { status: 200 }),
    });

    const result = await unwrapPage(res);
    expect(result).toEqual({
      items: [{ id: 1 }, { id: 2 }],
      cursor: "123",
    });
  });

  it("handles empty data and missing cursor", async () => {
    const res = Promise.resolve({
      data: {
        meta: {},
      },
      response: new Response(null, { status: 200 }),
    });

    const result = await unwrapPage(res);
    expect(result).toEqual({
      items: [],
      cursor: undefined,
    });
  });

  it("throws ApiError when response is not ok", async () => {
    const res = Promise.resolve({
      error: { error: { code: "BAD_REQUEST", message: "Invalid request" } },
      response: new Response(null, { status: 400 }),
    });

    await expect(unwrapPage(res)).rejects.toThrow(ApiError);
    await expect(unwrapPage(res)).rejects.toMatchObject({
      status: 400,
      code: "BAD_REQUEST",
      message: "Invalid request",
    });
  });

  it("attaches requestId from meta.request_id and ignores 4xx errors in Sentry", async () => {
    const res = Promise.resolve({
      error: {
        error: { code: "BAD_REQUEST", message: "Invalid request" },
        meta: { request_id: "req-400" },
      },
      response: new Response(null, { status: 400 }),
    });

    await expect(unwrap(res)).rejects.toMatchObject({
      status: 400,
      code: "BAD_REQUEST",
      requestId: "req-400",
    });
    expect(captureApiError).not.toHaveBeenCalled();
  });

  it("reports 5xx ApiError responses to Sentry with request_id", async () => {
    const res = Promise.resolve({
      error: {
        error: { code: "ERR_UNKNOWN", message: "Internal server error" },
        meta: { request_id: "req-500" },
      },
      response: new Response(null, { status: 500 }),
    });

    await expect(unwrapVoid(res)).rejects.toMatchObject({
      status: 500,
      code: "ERR_UNKNOWN",
      message: "Internal server error",
      requestId: "req-500",
    });
    expect(captureApiError).toHaveBeenCalledTimes(1);
    expect(captureApiError).toHaveBeenCalledWith(
      expect.objectContaining({
        status: 500,
        code: "ERR_UNKNOWN",
        requestId: "req-500",
      }),
      expect.objectContaining({
        requestId: "req-500",
      }),
    );
  });
});
