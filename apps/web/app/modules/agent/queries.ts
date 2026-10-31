import type { components } from "@omed/openapi";
import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrapList, unwrapOne, unwrapVoid } from "~/lib/api";
import { getStoredWorkspaceId } from "~/lib/workspace";

export type ProviderResponse = components["schemas"]["model.ProviderResponse"];
export type CreateProviderRequest =
  components["schemas"]["model.CreateProviderRequest"];
export type UpdateProviderRequest =
  components["schemas"]["model.UpdateProviderRequest"];
export type TestProviderRequest =
  components["schemas"]["model.TestProviderRequest"];
export type TestResult = components["schemas"]["model.TestResult"];
export type Capabilities = components["schemas"]["llmx.Capabilities"];

export const agentKeys = {
  all: (wid: number) => ["agent", "workspaces", wid] as const,
  providers: (wid: number) =>
    ["agent", "workspaces", wid, "providers"] as const,
  provider: (wid: number, id: number) =>
    ["agent", "workspaces", wid, "providers", id] as const,
};

export const providersQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: agentKeys.providers(wid),
    queryFn: () =>
      unwrapList(
        api.GET("/workspaces/{wid}/agent/providers", {
          params: { path: { wid } },
        }),
      ),
  });

export const providerQueryOptions = (wid: number, id: number) =>
  queryOptions({
    queryKey: agentKeys.provider(wid, id),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/agent/providers/{id}", {
          params: { path: { wid, id } },
        }),
      ),
  });

function useResolvedWid(wid?: number): number {
  return wid ?? getStoredWorkspaceId() ?? 0;
}

export function useProviders(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  return useQuery({
    ...providersQueryOptions(resolvedWid),
    enabled: Boolean(resolvedWid),
  });
}

export function useCreateProvider(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: CreateProviderRequest) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/providers", {
          params: { path: { wid: resolvedWid } },
          body,
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentKeys.providers(resolvedWid),
      });
    },
  });
}

export function useUpdateProvider(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: UpdateProviderRequest }) =>
      unwrapOne(
        api.PATCH("/workspaces/{wid}/agent/providers/{id}", {
          params: { path: { wid: resolvedWid, id } },
          body,
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentKeys.providers(resolvedWid),
      });
    },
  });
}

export function useDeleteProvider(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) =>
      unwrapVoid(
        api.DELETE("/workspaces/{wid}/agent/providers/{id}", {
          params: { path: { wid: resolvedWid, id } },
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentKeys.providers(resolvedWid),
      });
    },
  });
}

export function useSetDefaultProvider(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/providers/{id}/default", {
          params: { path: { wid: resolvedWid, id } },
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentKeys.providers(resolvedWid),
      });
    },
  });
}

export function useTestProvider(wid?: number) {
  const resolvedWid = useResolvedWid(wid);

  return useMutation({
    mutationFn: (id: number) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/providers/{id}/test", {
          params: { path: { wid: resolvedWid, id } },
        }),
      ),
  });
}

export function useTestProviderConfig(wid?: number) {
  const resolvedWid = useResolvedWid(wid);

  return useMutation({
    mutationFn: (body: TestProviderRequest) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/providers/test", {
          params: { path: { wid: resolvedWid } },
          body,
        }),
      ),
  });
}
