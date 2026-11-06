import type { components, paths } from "@omed/openapi";
import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { getStoredWorkspaceId } from "~/lib/workspace";
import { entryKeys, transactionKeys } from "../queries";

export type ConversationResponse =
  components["schemas"]["model.ConversationResponse"];
export type ConversationDetailResponse =
  components["schemas"]["model.ConversationDetailResponse"];
export type ListConversationsResponse =
  components["schemas"]["model.ListConversationsResponse"];
export type MessageResponse = components["schemas"]["model.MessageResponse"];
export type SendMessageResponse =
  components["schemas"]["model.SendMessageResponse"];
export type DraftProposal = components["schemas"]["model.DraftProposal"];
export type ConfirmDraftRequest =
  components["schemas"]["model.ConfirmDraftRequest"];
export type ConfirmDraftResponse =
  components["schemas"]["model.ConfirmDraftResponse"];
export type CreateConversationRequest =
  components["schemas"]["model.CreateConversationRequest"];
export type FinancialStatus = components["schemas"]["model.FinancialStatus"];

export type SendMessageParams = {
  conversationId?: number;
  content?: string;
  files?: File[];
};

export const agentChatKeys = {
  all: (wid: number) => ["agent", "workspaces", wid, "chat"] as const,
  conversations: (wid: number, params?: { page?: number; limit?: number }) =>
    [...agentChatKeys.all(wid), "conversations", params] as const,
  conversation: (wid: number, id: number) =>
    [...agentChatKeys.all(wid), "conversations", id] as const,
  financialStatus: (wid: number) =>
    ["agent", "workspaces", wid, "financial-status"] as const,
};

export const conversationsQueryOptions = (
  wid: number,
  params?: paths["/workspaces/{wid}/agent/chat/conversations"]["get"]["parameters"]["query"],
) =>
  queryOptions({
    queryKey: agentChatKeys.conversations(wid, params),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/agent/chat/conversations", {
          params: {
            path: { wid },
            query: params,
          },
        }),
      ),
  });

export const conversationQueryOptions = (wid: number, conversationId: number) =>
  queryOptions({
    queryKey: agentChatKeys.conversation(wid, conversationId),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/agent/chat/conversations/{id}", {
          params: {
            path: { wid, id: conversationId },
          },
        }),
      ),
  });

export const financialStatusQueryOptions = (wid: number) =>
  queryOptions({
    queryKey: agentChatKeys.financialStatus(wid),
    queryFn: () =>
      unwrapOne(
        api.GET("/workspaces/{wid}/agent/financial/status", {
          params: {
            path: { wid },
          },
        }),
      ),
    staleTime: 60_000,
  });

function useResolvedWid(wid?: number): number {
  return wid ?? getStoredWorkspaceId() ?? 0;
}

export function useConversations(
  wid?: number,
  params?: paths["/workspaces/{wid}/agent/chat/conversations"]["get"]["parameters"]["query"],
) {
  const resolvedWid = useResolvedWid(wid);
  return useQuery({
    ...conversationsQueryOptions(resolvedWid, params),
    enabled: Boolean(resolvedWid),
  });
}

export function useConversation(wid?: number, conversationId?: number) {
  const resolvedWid = useResolvedWid(wid);
  return useQuery({
    ...conversationQueryOptions(resolvedWid, conversationId ?? 0),
    enabled: Boolean(resolvedWid && conversationId),
  });
}

export function useFinancialStatus(
  wid?: number,
  options?: { enabled?: boolean },
) {
  const resolvedWid = useResolvedWid(wid);
  return useQuery({
    ...financialStatusQueryOptions(resolvedWid),
    enabled:
      options?.enabled !== undefined
        ? options.enabled && Boolean(resolvedWid)
        : Boolean(resolvedWid),
  });
}

export function useCreateConversation(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body?: CreateConversationRequest) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/chat/conversations", {
          params: { path: { wid: resolvedWid } },
          body: body ?? {},
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentChatKeys.all(resolvedWid),
      });
    },
  });
}

export function useDeleteConversation(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (conversationId: number) =>
      unwrapVoid(
        api.DELETE("/workspaces/{wid}/agent/chat/conversations/{id}", {
          params: { path: { wid: resolvedWid, id: conversationId } },
        }),
      ),
    onSuccess: (_data, conversationId) => {
      queryClient.removeQueries({
        queryKey: agentChatKeys.conversation(resolvedWid, conversationId),
      });
      queryClient.invalidateQueries({
        queryKey: agentChatKeys.all(resolvedWid),
      });
    },
  });
}

export function useSendMessage(wid?: number, conversationId?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      conversationId: overrideConvId,
      content,
      files,
    }: SendMessageParams) => {
      const convId = overrideConvId ?? conversationId ?? 0;
      const formData = new FormData();
      if (content !== undefined && content !== "") {
        formData.append("content", content);
      }
      if (files && files.length > 0) {
        formData.append("file", files[0]);
        for (let i = 1; i < files.length; i++) {
          formData.append("files", files[i]);
        }
      }

      return unwrapOne(
        api.POST("/workspaces/{wid}/agent/chat/conversations/{id}/messages", {
          params: { path: { wid: resolvedWid, id: convId } },
          body: formData as unknown as {
            content?: string;
            file?: string;
            files?: string;
          },
          bodySerializer: (body) => body as unknown as BodyInit,
        }),
      );
    },
    onSuccess: (data, variables) => {
      const targetId = variables?.conversationId ?? conversationId;
      if (targetId) {
        queryClient.setQueryData(
          agentChatKeys.conversation(resolvedWid, targetId),
          (old: ConversationDetailResponse | undefined) => {
            const existingMessages = old?.messages || [];
            const userMsg = data?.userMessage;
            const asstMsg = data?.assistantMessage;
            const newMessages = [...existingMessages];
            if (userMsg && !newMessages.some((m) => m.id === userMsg.id)) {
              newMessages.push(userMsg);
            }
            if (asstMsg && !newMessages.some((m) => m.id === asstMsg.id)) {
              newMessages.push(asstMsg);
            }
            return {
              ...(old || { id: targetId, title: "Chat" }),
              messages: newMessages,
            };
          },
        );
        queryClient.invalidateQueries({
          queryKey: agentChatKeys.conversation(resolvedWid, targetId),
        });
      }
      queryClient.invalidateQueries({
        queryKey: agentChatKeys.all(resolvedWid),
      });
    },
  });
}

export function useConfirmDraft(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      messageId,
      proposal,
    }: {
      messageId: number;
      proposal?: DraftProposal;
    }) =>
      unwrapOne(
        api.POST("/workspaces/{wid}/agent/chat/messages/{id}/confirm", {
          params: { path: { wid: resolvedWid, id: messageId } },
          body: proposal ? { proposal } : {},
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentChatKeys.all(resolvedWid),
      });
      queryClient.invalidateQueries({
        queryKey: transactionKeys.all(resolvedWid),
      });
      queryClient.invalidateQueries({
        queryKey: entryKeys.all(resolvedWid),
      });
    },
  });
}

export function useDiscardDraft(wid?: number) {
  const resolvedWid = useResolvedWid(wid);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: number) =>
      unwrapVoid(
        api.POST("/workspaces/{wid}/agent/chat/messages/{id}/discard", {
          params: { path: { wid: resolvedWid, id: messageId } },
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: agentChatKeys.all(resolvedWid),
      });
    },
  });
}
