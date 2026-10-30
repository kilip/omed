import type { QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import {
  expenseKeys,
  expensesInfiniteQueryOptions,
  finSettingsQueryOptions,
  incomeKeys,
  incomesInfiniteQueryOptions,
  settingKeys,
} from "./queries";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
    },
  };
});

describe("fin cash flow queries", () => {
  const wid = 7;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("query keys", () => {
    it("generates correct incomeKeys", () => {
      expect(incomeKeys.all(wid)).toEqual(["fin", "workspaces", wid, "income"]);
      expect(incomeKeys.infinite(wid, { query: "consulting" })).toEqual([
        "fin",
        "workspaces",
        wid,
        "income",
        "infinite",
        { query: "consulting" },
      ]);
    });

    it("generates correct expenseKeys", () => {
      expect(expenseKeys.all(wid)).toEqual([
        "fin",
        "workspaces",
        wid,
        "expense",
      ]);
      expect(expenseKeys.infinite(wid, { query: "hosting" })).toEqual([
        "fin",
        "workspaces",
        wid,
        "expense",
        "infinite",
        { query: "hosting" },
      ]);
    });
  });

  describe("incomesInfiniteQueryOptions", () => {
    it("configures queryKey, pagination, and fetches pages correctly", async () => {
      const filters = { query: "invoice" };
      const options = incomesInfiniteQueryOptions(wid, filters);

      expect(options.queryKey).toEqual([
        "fin",
        "workspaces",
        wid,
        "income",
        "infinite",
        filters,
      ]);
      expect(options.initialPageParam).toBe("");
      expect(options.getNextPageParam({ items: [], cursor: "next-cur" })).toBe(
        "next-cur",
      );
      expect(options.getNextPageParam({ items: [] })).toBeUndefined();

      const mockResponse = {
        data: {
          data: [{ id: 101, amount: "500000" }],
          meta: { cursor: "cur-2" },
        },
        response: new Response(),
      };
      vi.mocked(api.POST).mockResolvedValueOnce(mockResponse as never);

      const context = {
        pageParam: "cur-1",
      } as QueryFunctionContext<ReturnType<typeof incomeKeys.infinite>, string>;
      const result = await options.queryFn(context);

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/fin/income/search",
        {
          params: { path: { wid } },
          body: {
            ...filters,
            cursor: "cur-1",
            limit: 20,
          },
        },
      );
      expect(result).toEqual({
        items: [{ id: 101, amount: "500000" }],
        cursor: "cur-2",
      });
    });
  });

  describe("expensesInfiniteQueryOptions", () => {
    it("configures queryKey, pagination, and fetches pages correctly", async () => {
      const filters = { categoryAccountId: 12 };
      const options = expensesInfiniteQueryOptions(wid, filters, 10);

      expect(options.queryKey).toEqual([
        "fin",
        "workspaces",
        wid,
        "expense",
        "infinite",
        filters,
      ]);
      expect(options.initialPageParam).toBe("");
      expect(options.getNextPageParam({ items: [], cursor: "next-cur" })).toBe(
        "next-cur",
      );
      expect(options.getNextPageParam({ items: [] })).toBeUndefined();

      const mockResponse = {
        data: {
          data: [{ id: 202, amount: "150000" }],
          meta: { cursor: "cur-next" },
        },
        response: new Response(),
      };
      vi.mocked(api.POST).mockResolvedValueOnce(mockResponse as never);

      const context = {
        pageParam: "",
      } as QueryFunctionContext<
        ReturnType<typeof expenseKeys.infinite>,
        string
      >;
      const result = await options.queryFn(context);

      expect(api.POST).toHaveBeenCalledWith(
        "/workspaces/{wid}/fin/expense/search",
        {
          params: { path: { wid } },
          body: {
            ...filters,
            cursor: undefined,
            limit: 10,
          },
        },
      );
      expect(result).toEqual({
        items: [{ id: 202, amount: "150000" }],
        cursor: "cur-next",
      });
    });
  });

  describe("finSettingsQueryOptions", () => {
    it("generates correct settingKeys", () => {
      expect(settingKeys.all(wid)).toEqual([
        "fin",
        "workspaces",
        wid,
        "settings",
      ]);
    });

    it("configures queryKey and fetches settings via GET", async () => {
      const options = finSettingsQueryOptions(wid);
      expect(options.queryKey).toEqual(["fin", "workspaces", wid, "settings"]);

      const mockResponse = {
        data: {
          data: {
            baseCurrency: "USD",
            decimalStyle: "dot",
            onboarding: true,
          },
        },
        response: new Response(),
      };
      vi.mocked(api.GET).mockResolvedValueOnce(mockResponse as never);

      const result = await (
        options as { queryFn: () => Promise<unknown> }
      ).queryFn();
      expect(api.GET).toHaveBeenCalledWith("/workspaces/{wid}/fin/settings", {
        params: { path: { wid } },
      });
      expect(result).toEqual({
        baseCurrency: "USD",
        decimalStyle: "dot",
        onboarding: true,
      });
    });
  });
});
