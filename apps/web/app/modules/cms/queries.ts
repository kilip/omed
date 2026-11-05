import type { paths } from "@omed/openapi";
import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { api, unwrapOne, unwrapPage } from "~/lib/api";

type SearchMediaFilters = NonNullable<
  paths["/workspaces/{wid}/cms/media/search"]["post"]["requestBody"]
>["content"]["application/json"];

type SearchPostFilters = NonNullable<
  paths["/workspaces/{wid}/cms/posts/search"]["post"]["requestBody"]
>["content"]["application/json"];

type SearchPageFilters = NonNullable<
  paths["/workspaces/{wid}/cms/pages/search"]["post"]["requestBody"]
>["content"]["application/json"];

export const settingKeys = {
  all: (wid: number) => ["cms", "workspaces", wid, "settings"] as const,
};

export const mediaKeys = {
  all: (wid: number) => ["cms", "workspaces", wid, "media"] as const,
  infinite: (wid: number, filters?: SearchMediaFilters) =>
    [...mediaKeys.all(wid), "infinite", filters] as const,
  detail: (wid: number, id: number) =>
    [...mediaKeys.all(wid), "detail", id] as const,
};

export const postKeys = {
  all: (wid: number) => ["cms", "workspaces", wid, "posts"] as const,
  infinite: (wid: number, filters?: SearchPostFilters) =>
    [...postKeys.all(wid), "infinite", filters] as const,
  detail: (wid: number, id: number) =>
    [...postKeys.all(wid), "detail", id] as const,
};

export const pageKeys = {
  all: (wid: number) => ["cms", "workspaces", wid, "pages"] as const,
  infinite: (wid: number, filters?: SearchPageFilters) =>
    [...pageKeys.all(wid), "infinite", filters] as const,
  detail: (wid: number, id: number) =>
    [...pageKeys.all(wid), "detail", id] as const,
};

/** Query options for CMS workspace settings. */
export const cmsSettingQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: settingKeys.all(wid),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/cms/settings", {
          params: { path: { wid } },
        }),
      ),
  });

/** Infinite query options for media library. */
export const mediaInfiniteQueryOptions = (
  wid: number,
  filters?: SearchMediaFilters,
  limit = 20,
) =>
  infiniteQueryOptions({
    queryKey: mediaKeys.infinite(wid, filters),
    queryFn: ({ pageParam }) =>
      unwrapPage(
        api.POST("/workspaces/{wid}/cms/media/search", {
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

/** Single media item by ID. */
export const mediaQueryOptions = (wid: number, id: number) =>
  queryOptions({
    queryKey: mediaKeys.detail(wid, id),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/cms/media/{id}", {
          params: { path: { wid, id } },
        }),
      ),
  });

/** Infinite query options for posts. */
export const postsInfiniteQueryOptions = (
  wid: number,
  filters?: SearchPostFilters,
  limit = 20,
) =>
  infiniteQueryOptions({
    queryKey: postKeys.infinite(wid, filters),
    queryFn: ({ pageParam }) =>
      unwrapPage(
        api.POST("/workspaces/{wid}/cms/posts/search", {
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

/** Single post by ID, optionally scoped to a locale. */
export const postQueryOptions = (wid: number, id: number, locale?: string) =>
  queryOptions({
    queryKey: [...postKeys.detail(wid, id), locale] as const,
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/cms/posts/{id}", {
          params: {
            path: { wid, id },
            query: locale ? { locale } : undefined,
          },
        }),
      ),
  });

/** Infinite query options for pages. */
export const pagesInfiniteQueryOptions = (
  wid: number,
  filters?: SearchPageFilters,
  limit = 20,
) =>
  infiniteQueryOptions({
    queryKey: pageKeys.infinite(wid, filters),
    queryFn: ({ pageParam }) =>
      unwrapPage(
        api.POST("/workspaces/{wid}/cms/pages/search", {
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

/** Single page by ID, optionally scoped to a locale. */
export const pageQueryOptions = (wid: number, id: number, locale?: string) =>
  queryOptions({
    queryKey: [...pageKeys.detail(wid, id), locale] as const,
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/cms/pages/{id}", {
          params: {
            path: { wid, id },
            query: locale ? { locale } : undefined,
          },
        }),
      ),
  });
