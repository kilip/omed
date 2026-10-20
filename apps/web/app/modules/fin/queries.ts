import type { paths } from "@omed/openapi";
import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { api, unwrapList, unwrapOne, unwrapPage } from "~/lib/api";

type SearchEntryFilters = NonNullable<
  paths["/workspaces/{wid}/fin/entries/search"]["post"]["requestBody"]
>["content"]["application/json"];

type SearchTransactionFilters = NonNullable<
  paths["/workspaces/{wid}/fin/transactions/search"]["post"]["requestBody"]
>["content"]["application/json"];

type SearchPeriodFilters = NonNullable<
  paths["/workspaces/{wid}/fin/periods/search"]["post"]["requestBody"]
>["content"]["application/json"];

export const accountKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "accounts"] as const,
};

export const periodKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "periods"] as const,
  list: (wid: number, filters?: SearchPeriodFilters) =>
    [...periodKeys.all(wid), "list", filters] as const,
};

export const currencyKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "currencies"] as const,
};

export const entryKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "entries"] as const,
  infinite: (wid: number, filters?: SearchEntryFilters) =>
    [...entryKeys.all(wid), "infinite", filters] as const,
  detail: (wid: number, id: number) =>
    [...entryKeys.all(wid), "detail", id] as const,
};

export const transactionKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "transactions"] as const,
  infinite: (wid: number, filters?: SearchTransactionFilters) =>
    [...transactionKeys.all(wid), "infinite", filters] as const,
  detail: (wid: number, id: number) =>
    [...transactionKeys.all(wid), "detail", id] as const,
};

export const ledgerKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "ledger"] as const,
  account: (wid: number, accountId: number, page: number, limit: number) =>
    [...ledgerKeys.all(wid), "account", accountId, { page, limit }] as const,
};

export const reportKeys = {
  all: (wid: number) => ["fin", "workspaces", wid, "reports"] as const,
  trialBalance: (wid: number, asOfDate: string) =>
    [...reportKeys.all(wid), "trial-balance", asOfDate] as const,
};

/** Full chart of accounts for a workspace (filtering and tree-building happen client-side). */
export const accountsQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: accountKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/{wid}/fin/accounts/search", {
          params: { path: { wid } },
          body: {},
        }),
      ),
  });

/** Template preview for chart of accounts. */
export const templatePreviewQueryOptions = (
  wid: number,
  template: string,
  lang: string,
) =>
  queryOptions({
    queryKey: ["fin", "workspaces", wid, "templates", template, lang] as const,
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/fin/accounts/templates/{template}", {
          params: {
            path: { wid, template },
            query: { lang },
          },
        }),
      ),
  });

/** Periods in a workspace. */
export const periodsQueryOptions = (
  wid: number,
  filters?: SearchPeriodFilters,
) =>
  queryOptions({
    queryKey: periodKeys.list(wid, filters),
    queryFn: () =>
      unwrapList(
        api.POST("/workspaces/{wid}/fin/periods/search", {
          params: { path: { wid } },
          body: filters ?? {},
        }),
      ),
  });

/** Supported ISO 4217 currencies. */
export const currenciesQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: currencyKeys.all(wid),
    queryFn: () =>
      unwrapList(
        api.GET("/workspaces/{wid}/fin/currencies", {
          params: { path: { wid } },
        }),
      ),
  });

/** Single transaction by ID. */
export const transactionQueryOptions = (wid: number, id: number) =>
  queryOptions({
    queryKey: transactionKeys.detail(wid, id),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/fin/transactions/{id}", {
          params: { path: { wid, id } },
        }),
      ),
  });

/** Single entry by ID with postings. */
export const entryQueryOptions = (wid: number, id: number) =>
  queryOptions({
    queryKey: entryKeys.detail(wid, id),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/fin/entries/{id}", {
          params: { path: { wid, id } },
        }),
      ),
  });

/** Account ledger with running balance. */
export const accountLedgerQueryOptions = (
  wid: number,
  accountId: number,
  page = 1,
  limit = 20,
) =>
  queryOptions({
    queryKey: ledgerKeys.account(wid, accountId, page, limit),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/fin/accounts/{id}/ledger", {
          params: {
            path: { wid, id: accountId },
            query: { page, limit },
          },
        }),
      ),
  });

/** Trial balance report as of date. */
export const trialBalanceQueryOptions = (wid: number, asOfDate: string) =>
  queryOptions({
    queryKey: reportKeys.trialBalance(wid, asOfDate),
    queryFn: () =>
      unwrapOne(
        api.POST("/workspaces/{wid}/fin/reports/trial-balance", {
          params: { path: { wid } },
          body: { asOfDate },
        }),
      ),
  });

/** Infinite query options for journal entries. */
export const entriesInfiniteQueryOptions = (
  wid: number,
  filters?: SearchEntryFilters,
  limit = 20,
) =>
  infiniteQueryOptions({
    queryKey: entryKeys.infinite(wid, filters),
    queryFn: ({ pageParam }) =>
      unwrapPage(
        api.POST("/workspaces/{wid}/fin/entries/search", {
          params: { path: { wid } },
          body: {
            ...filters,
            cursor: pageParam || undefined,
            limit,
          },
        }),
      ),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.cursor ?? undefined,
  });

/** Infinite query options for transactions. */
export const transactionsInfiniteQueryOptions = (
  wid: number,
  filters?: SearchTransactionFilters,
  limit = 20,
) =>
  infiniteQueryOptions({
    queryKey: transactionKeys.infinite(wid, filters),
    queryFn: ({ pageParam }) =>
      unwrapPage(
        api.POST("/workspaces/{wid}/fin/transactions/search", {
          params: { path: { wid } },
          body: {
            ...filters,
            cursor: pageParam || undefined,
            limit,
          },
        }),
      ),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.cursor ?? undefined,
  });
