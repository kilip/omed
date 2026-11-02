import type { QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, api } from "~/lib/api";
import {
  agentKeys,
  analyzeFinancialDocument,
  financialStatusQueryOptions,
  settingsQueryOptions,
} from "./queries";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      PATCH: vi.fn(),
    },
  };
});

describe("agent queries", () => {
  const wid = 42;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("query keys", () => {
    it("generates correct settings and financialStatus keys", () => {
      expect(agentKeys.settings(wid)).toEqual([
        "agent",
        "workspaces",
        wid,
        "settings",
      ]);
      expect(agentKeys.financialStatus(wid)).toEqual([
        "agent",
        "workspaces",
        wid,
        "financial-status",
      ]);
    });
  });

  describe("settingsQueryOptions", () => {
    it("configures queryKey, staleTime, and fetches settings", async () => {
      const options = settingsQueryOptions(wid);
      expect(options.queryKey).toEqual(agentKeys.settings(wid));
      expect(options.staleTime).toBe(60_000);

      const mockSetting = {
        financialEnabled: true,
        financialProviderId: 1,
      };
      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: mockSetting },
        response: new Response(null, { status: 200 }),
      } as never);

      const queryFn = options.queryFn as (
        context: QueryFunctionContext,
      ) => Promise<unknown>;
      const result = await queryFn({} as QueryFunctionContext);
      expect(result).toEqual(mockSetting);
      expect(api.GET).toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/settings",
        expect.objectContaining({ params: { path: { wid } } }),
      );
    });
  });

  describe("financialStatusQueryOptions", () => {
    it("configures queryKey, staleTime, and fetches financial status", async () => {
      const options = financialStatusQueryOptions(wid);
      expect(options.queryKey).toEqual(agentKeys.financialStatus(wid));
      expect(options.staleTime).toBe(60_000);

      const mockStatus = {
        available: true,
        enabled: true,
        hasProvider: true,
        hasVision: true,
      };
      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { data: mockStatus },
        response: new Response(null, { status: 200 }),
      } as never);

      const queryFn = options.queryFn as (
        context: QueryFunctionContext,
      ) => Promise<unknown>;
      const result = await queryFn({} as QueryFunctionContext);
      expect(result).toEqual(mockStatus);
      expect(api.GET).toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/financial/status",
        expect.objectContaining({ params: { path: { wid } } }),
      );
    });
  });

  describe("analyzeFinancialDocument", () => {
    it("posts FormData with file and flow and returns the unwrapped suggestion", async () => {
      const file = new File(["dummy content"], "invoice.pdf", {
        type: "application/pdf",
      });
      const mockSuggestion = {
        documentType: "invoice",
        documentTotal: 150000,
        currency: "IDR",
        items: [],
      };

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: mockSuggestion },
        response: new Response(null, { status: 200 }),
      } as never);

      const result = await analyzeFinancialDocument(wid, "EXPENSE", file);

      expect(result).toEqual(mockSuggestion);
      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/agent/financial/analyze",
        expect.objectContaining({
          params: { path: { wid } },
          body: expect.any(FormData),
          bodySerializer: expect.any(Function),
        }),
      );

      const callArgs = vi.mocked(api.POST).mock.calls[0][1] as {
        body: FormData;
        bodySerializer: (b: unknown) => unknown;
      };
      const formData = callArgs.body;
      expect(formData.get("file")).toBe(file);
      expect(formData.get("flow")).toBe("EXPENSE");
      expect(callArgs.bodySerializer(formData)).toBe(formData);
    });

    it("rejects with an abort error when signal is aborted", async () => {
      const file = new File(["dummy"], "receipt.jpg", { type: "image/jpeg" });
      const controller = new AbortController();
      controller.abort();

      await expect(
        analyzeFinancialDocument(wid, "EXPENSE", file, controller.signal),
      ).rejects.toThrow();
    });

    it("aborts when signal is aborted while in flight", async () => {
      const file = new File(["dummy"], "receipt.jpg", { type: "image/jpeg" });
      const controller = new AbortController();

      vi.mocked(api.POST).mockImplementationOnce((_path, init) => {
        return new Promise((_resolve, reject) => {
          init?.signal?.addEventListener("abort", () => {
            reject(
              new DOMException("The operation was aborted.", "AbortError"),
            );
          });
        });
      });

      const promise = analyzeFinancialDocument(
        wid,
        "EXPENSE",
        file,
        controller.signal,
      );
      controller.abort();

      await expect(promise).rejects.toThrow();
    });

    it("rejects with an ApiError whose code is RATE_LIMITED on 429 response", async () => {
      const file = new File(["dummy"], "receipt.jpg", { type: "image/jpeg" });

      vi.mocked(api.POST).mockResolvedValueOnce({
        error: {
          error: {
            code: "RATE_LIMITED",
            message: "Too many requests",
          },
        },
        response: new Response(
          JSON.stringify({
            error: { code: "RATE_LIMITED", message: "Too many requests" },
          }),
          { status: 429, statusText: "Too Many Requests" },
        ),
      } as never);

      const promise = analyzeFinancialDocument(wid, "EXPENSE", file);

      await expect(promise).rejects.toThrow(ApiError);
      await expect(promise).rejects.toMatchObject({
        status: 429,
        code: "RATE_LIMITED",
      });
    });
  });
});
