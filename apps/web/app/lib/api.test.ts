import { describe, expect, it } from "vitest";
import { ApiError, unwrapPage } from "./api";

describe("unwrapPage", () => {
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
});
