import type { QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import { entryKeys, transactionKeys } from "../queries";
import {
  agentChatKeys,
  conversationQueryOptions,
  conversationsQueryOptions,
  financialStatusQueryOptions,
  useConfirmDraft,
  useConversation,
  useConversations,
  useCreateConversation,
  useDeleteConversation,
  useDiscardDraft,
  useFinancialStatus,
  useSendMessage,
  useUpdateConversation,
} from "./use-agent-chat";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      PATCH: vi.fn(),
      DELETE: vi.fn(),
    },
  };
});

const mockUseQueryClient = vi.fn();
const mockUseQuery = vi.fn();
const mockUseMutation = vi.fn();

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => mockUseQueryClient(),
    useQuery: (options: unknown) => mockUseQuery(options),
    useMutation: (options: unknown) => mockUseMutation(options),
  };
});

describe("use-agent-chat query keys", () => {
  const wid = 10;

  it("builds consistent keys for chat and status", () => {
    expect(agentChatKeys.all(wid)).toEqual([
      "agent",
      "workspaces",
      wid,
      "chat",
    ]);
    expect(agentChatKeys.conversations(wid, { page: 1, limit: 10 })).toEqual([
      "agent",
      "workspaces",
      wid,
      "chat",
      "conversations",
      { page: 1, limit: 10 },
    ]);
    expect(agentChatKeys.conversation(wid, 5)).toEqual([
      "agent",
      "workspaces",
      wid,
      "chat",
      "conversations",
      5,
    ]);
    expect(agentChatKeys.financialStatus(wid)).toEqual([
      "agent",
      "workspaces",
      wid,
      "financial-status",
    ]);
  });
});

describe("agent chat query options", () => {
  const wid = 10;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("conversationsQueryOptions configures queryKey and calls GET endpoint", async () => {
    const options = conversationsQueryOptions(wid, { page: 1, limit: 20 });
    expect(options.queryKey).toEqual(
      agentChatKeys.conversations(wid, { page: 1, limit: 20 }),
    );

    const mockResponse = {
      conversations: [{ id: 1, title: "Test" }],
      total: 1,
    };
    vi.mocked(api.GET).mockResolvedValueOnce({
      data: { data: mockResponse },
      response: new Response(null, { status: 200 }),
    } as never);

    const queryFn = options.queryFn as (
      ctx: QueryFunctionContext,
    ) => Promise<unknown>;
    const res = await queryFn({} as QueryFunctionContext);

    expect(res).toEqual(mockResponse);
    expect(api.GET).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations",
      expect.objectContaining({
        params: {
          path: { wid },
          query: { page: 1, limit: 20 },
        },
      }),
    );
  });

  it("conversationQueryOptions configures queryKey and calls GET conversation by id", async () => {
    const options = conversationQueryOptions(wid, 42);
    expect(options.queryKey).toEqual(agentChatKeys.conversation(wid, 42));

    const mockDetail = { id: 42, title: "Receipt", messages: [] };
    vi.mocked(api.GET).mockResolvedValueOnce({
      data: { data: mockDetail },
      response: new Response(null, { status: 200 }),
    } as never);

    const queryFn = options.queryFn as (
      ctx: QueryFunctionContext,
    ) => Promise<unknown>;
    const res = await queryFn({} as QueryFunctionContext);

    expect(res).toEqual(mockDetail);
    expect(api.GET).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations/{id}",
      expect.objectContaining({
        params: {
          path: { wid, id: 42 },
        },
      }),
    );
  });

  it("financialStatusQueryOptions configures queryKey and fetches financial status", async () => {
    const options = financialStatusQueryOptions(wid);
    expect(options.queryKey).toEqual(agentChatKeys.financialStatus(wid));
    expect(options.staleTime).toBe(60_000);

    const mockStatus = { enabled: true, available: true };
    vi.mocked(api.GET).mockResolvedValueOnce({
      data: { data: mockStatus },
      response: new Response(null, { status: 200 }),
    } as never);

    const queryFn = options.queryFn as (
      ctx: QueryFunctionContext,
    ) => Promise<unknown>;
    const res = await queryFn({} as QueryFunctionContext);

    expect(res).toEqual(mockStatus);
    expect(api.GET).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/financial/status",
      expect.objectContaining({
        params: {
          path: { wid },
        },
      }),
    );
  });
});

describe("agent chat hooks", () => {
  const wid = 7;
  const mockQueryClient = {
    invalidateQueries: vi.fn(),
    removeQueries: vi.fn(),
    setQueryData: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseQueryClient.mockReturnValue(mockQueryClient);
  });

  it("useConversations delegates to useQuery with enabled guard", () => {
    useConversations(wid, { page: 2 });
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: agentChatKeys.conversations(wid, { page: 2 }),
        enabled: true,
      }),
    );
  });

  it("useConversation delegates to useQuery with enabled guard", () => {
    useConversation(wid, 3);
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: agentChatKeys.conversation(wid, 3),
        enabled: true,
      }),
    );

    useConversation(0, undefined);
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: false,
      }),
    );
  });

  it("useFinancialStatus delegates to useQuery with optional enabled flag", () => {
    useFinancialStatus(wid, { enabled: false });
    expect(mockUseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: agentChatKeys.financialStatus(wid),
        enabled: false,
      }),
    );
  });

  it("useCreateConversation creates conversation and invalidates chat queries", async () => {
    let capturedOptions: {
      mutationFn: (body?: { title?: string }) => Promise<unknown>;
      onSuccess?: () => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn(), mutateAsync: vi.fn() };
    });

    useCreateConversation(wid);

    const createdConversation = { id: 101, title: "New Receipt" };
    vi.mocked(api.POST).mockResolvedValueOnce({
      data: { data: createdConversation },
      response: new Response(null, { status: 201 }),
    } as never);

    const result = await capturedOptions.mutationFn({ title: "New Receipt" });
    expect(result).toEqual(createdConversation);
    expect(api.POST).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations",
      expect.objectContaining({
        params: { path: { wid } },
        body: { title: "New Receipt" },
      }),
    );

    capturedOptions.onSuccess?.();
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.all(wid),
    });
  });

  it("useDeleteConversation deletes conversation, removes cached detail and invalidates list", async () => {
    let capturedOptions: {
      mutationFn: (id: number) => Promise<unknown>;
      onSuccess?: (data: unknown, id: number) => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useDeleteConversation(wid);

    vi.mocked(api.DELETE).mockResolvedValueOnce({
      response: new Response(null, { status: 200 }),
    } as never);

    await capturedOptions.mutationFn(99);
    expect(api.DELETE).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations/{id}",
      expect.objectContaining({
        params: { path: { wid, id: 99 } },
      }),
    );

    capturedOptions.onSuccess?.(undefined, 99);
    expect(mockQueryClient.removeQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.conversation(wid, 99),
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.all(wid),
    });
  });
  it("useUpdateConversation updates conversation title and updates cache", async () => {
    let capturedOptions: {
      mutationFn: (variables: {
        conversationId: number;
        title: string;
      }) => Promise<unknown>;
      onSuccess?: (
        data: unknown,
        variables: { conversationId: number; title: string },
      ) => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useUpdateConversation(wid);

    const updatedConv = { id: 77, title: "Renamed Title" };
    vi.mocked(api.PATCH).mockResolvedValueOnce({
      data: { data: updatedConv },
      response: new Response(null, { status: 200 }),
    } as never);

    const result = await capturedOptions.mutationFn({
      conversationId: 77,
      title: "Renamed Title",
    });
    expect(result).toEqual(updatedConv);
    expect(api.PATCH).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations/{id}",
      expect.objectContaining({
        params: { path: { wid, id: 77 } },
        body: { title: "Renamed Title" },
      }),
    );

    capturedOptions.onSuccess?.(updatedConv, {
      conversationId: 77,
      title: "Renamed Title",
    });
    expect(mockQueryClient.setQueryData).toHaveBeenCalledWith(
      agentChatKeys.conversation(wid, 77),
      expect.any(Function),
    );
    // Test updater fn passed to setQueryData
    const updater = vi.mocked(mockQueryClient.setQueryData).mock
      .calls[0][1] as (old: unknown) => unknown;
    expect(updater({ id: 77, title: "Old Title", messages: [] })).toEqual({
      id: 77,
      title: "Renamed Title",
      messages: [],
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.conversations(wid),
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.conversation(wid, 77),
    });
  });

  it("useSendMessage packs FormData for content and multiple files", async () => {
    let capturedOptions: {
      mutationFn: (params: {
        content?: string;
        files?: File[];
      }) => Promise<unknown>;
      onSuccess?: () => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useSendMessage(wid, 42);

    const file1 = new File(["dummy1"], "receipt1.jpg", { type: "image/jpeg" });
    const file2 = new File(["dummy2"], "receipt2.pdf", {
      type: "application/pdf",
    });

    const sendResponse = {
      userMessage: { id: 1, role: "user", content: "Analyze these" },
      assistantMessage: {
        id: 2,
        role: "assistant",
        content: "Here is the proposal",
      },
    };
    vi.mocked(api.POST).mockResolvedValueOnce({
      data: { data: sendResponse },
      response: new Response(null, { status: 200 }),
    } as never);

    const result = await capturedOptions.mutationFn({
      content: "Analyze these",
      files: [file1, file2],
    });

    expect(result).toEqual(sendResponse);
    expect(api.POST).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/conversations/{id}/messages",
      expect.objectContaining({
        params: { path: { wid, id: 42 } },
        body: expect.any(FormData),
        bodySerializer: expect.any(Function),
      }),
    );

    const callArgs = vi.mocked(api.POST).mock.calls[0][1] as unknown as {
      body: FormData;
      bodySerializer: (b: unknown) => unknown;
    };
    expect(callArgs.body.get("content")).toBe("Analyze these");
    expect(callArgs.body.get("file")).toBe(file1);
    expect(callArgs.body.get("files")).toBe(file2);
    expect(callArgs.bodySerializer(callArgs.body)).toBe(callArgs.body);

    capturedOptions.onSuccess?.();
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.conversation(wid, 42),
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.all(wid),
    });
  });

  it("useSendMessage updates conversation title in cache if conversationTitle is returned", async () => {
    let capturedOptions: {
      mutationFn: (params: {
        content?: string;
        conversationId?: number;
      }) => Promise<unknown>;
      onSuccess?: (
        data: unknown,
        variables: { conversationId?: number; content?: string },
      ) => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useSendMessage(wid, 42);

    const sendResponse = {
      conversationTitle: "Auto Generated Title",
      userMessage: { id: 1, role: "user", content: "Analyze receipt" },
      assistantMessage: {
        id: 2,
        role: "assistant",
        content: "Draft ready",
      },
    };

    capturedOptions.onSuccess?.(sendResponse, { conversationId: 42 });

    expect(mockQueryClient.setQueryData).toHaveBeenCalledWith(
      agentChatKeys.conversation(wid, 42),
      expect.any(Function),
    );
    const updater = vi.mocked(mockQueryClient.setQueryData).mock
      .calls[0][1] as (old: unknown) => unknown;
    const updated = updater({
      id: 42,
      title: "New conversation",
      messages: [],
    });
    expect(updated).toMatchObject({
      id: 42,
      title: "Auto Generated Title",
    });

    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.conversations(wid),
    });
  });

  it("useConfirmDraft sends proposal and invalidates chat, transactions and entries", async () => {
    let capturedOptions: {
      mutationFn: (args: {
        messageId: number;
        proposal?: unknown;
      }) => Promise<unknown>;
      onSuccess?: () => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useConfirmDraft(wid);

    const confirmResponse = { transactionId: 888 };
    vi.mocked(api.POST).mockResolvedValueOnce({
      data: { data: confirmResponse },
      response: new Response(null, { status: 200 }),
    } as never);

    const sampleProposal = {
      kind: "expense" as const,
      totalAmount: "70000.00",
    };
    const result = await capturedOptions.mutationFn({
      messageId: 123,
      proposal: sampleProposal,
    });

    expect(result).toEqual(confirmResponse);
    expect(api.POST).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/messages/{id}/confirm",
      expect.objectContaining({
        params: { path: { wid, id: 123 } },
        body: { proposal: sampleProposal },
      }),
    );

    capturedOptions.onSuccess?.();
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.all(wid),
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: transactionKeys.all(wid),
    });
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: entryKeys.all(wid),
    });
  });

  it("useDiscardDraft posts discard and invalidates chat queries", async () => {
    let capturedOptions: {
      mutationFn: (messageId: number) => Promise<unknown>;
      onSuccess?: () => void;
    } = {} as never;

    mockUseMutation.mockImplementation((opts) => {
      capturedOptions = opts;
      return { mutate: vi.fn() };
    });

    useDiscardDraft(wid);

    vi.mocked(api.POST).mockResolvedValueOnce({
      response: new Response(null, { status: 200 }),
    } as never);

    await capturedOptions.mutationFn(456);
    expect(api.POST).toHaveBeenCalledWith(
      "/workspaces/{wid}/agent/chat/messages/{id}/discard",
      expect.objectContaining({
        params: { path: { wid, id: 456 } },
      }),
    );

    capturedOptions.onSuccess?.();
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: agentChatKeys.all(wid),
    });
  });
});
