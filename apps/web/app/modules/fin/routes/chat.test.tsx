import type * as reactQuery from "@tanstack/react-query";
import type * as antd from "antd";
import type React from "react";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SendMessageParams } from "../hooks/use-agent-chat";
import FinChatPage from "./chat";

const mockMessage = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
};

let capturedComposerProps: {
  onSend?: (params: SendMessageParams) => Promise<void>;
  disabled?: boolean;
  isSending?: boolean;
} = {};

const mockMutateAsync = vi.fn();
const mockCreateConversationMutateAsync = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, defaultValue?: string) => defaultValue ?? key,
    i18n: { language: "en" },
  }),
}));

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof antd>();
  return {
    ...actual,
    App: {
      ...actual.App,
      useApp: () => ({
        message: mockMessage,
      }),
    },
  };
});

vi.mock("~/components/Page", () => ({
  Page: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn().mockResolvedValue(1),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: () => ({
    ensureQueryData: vi.fn(),
  }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof reactQuery>();
  return {
    ...actual,
    useQuery: () => ({ data: [] }),
  };
});

vi.mock("~/modules/agent/hooks/use-agent-permissions", () => ({
  useAgentPermissions: () => ({
    canUseFinancial: true,
  }),
}));

vi.mock("../hooks/use-fin-permissions", () => ({
  useFinPermissions: () => ({
    canCreateTransaction: true,
  }),
}));

vi.mock("../hooks/use-agent-chat", () => ({
  useFinancialStatus: () => ({
    data: { ready: true },
    isLoading: false,
  }),
  useConversations: () => ({
    data: { conversations: [{ id: 101, title: "Test conversation" }] },
    isLoading: false,
  }),
  useConversation: () => ({
    data: { id: 101, title: "Test conversation", messages: [] },
    isLoading: false,
  }),
  useCreateConversation: () => ({
    mutateAsync: mockCreateConversationMutateAsync,
  }),
  useDeleteConversation: () => ({
    mutateAsync: vi.fn(),
  }),
  useSendMessage: () => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useConfirmDraft: () => ({
    mutateAsync: vi.fn(),
  }),
  useDiscardDraft: () => ({
    mutateAsync: vi.fn(),
  }),
}));

vi.mock("../components/chat", () => ({
  ConversationSidebar: () => <div>Sidebar</div>,
  MessageList: () => <div>MessageList</div>,
  MessageComposer: (props: typeof capturedComposerProps) => {
    capturedComposerProps = props;
    return <div>MessageComposer</div>;
  },
}));

describe("FinChatPage error toast handling", () => {
  beforeEach(() => {
    capturedComposerProps = {};
    vi.clearAllMocks();
  });

  it("calls message.error with error message when sendMessage.mutateAsync rejects with Error", async () => {
    mockMutateAsync.mockRejectedValueOnce(
      new Error("Gemini upstream 502 error"),
    );

    renderToString(<FinChatPage loaderData={{ wid: 1 }} />);

    expect(capturedComposerProps.onSend).toBeDefined();

    await capturedComposerProps.onSend?.({
      content: "Analyze this document",
    });

    expect(mockMessage.error).toHaveBeenCalledTimes(1);
    expect(mockMessage.error).toHaveBeenCalledWith("Gemini upstream 502 error");
  });

  it("calls message.error with localized fallback string when rejection is not an Error instance", async () => {
    mockMutateAsync.mockRejectedValueOnce("Unknown server failure");

    renderToString(<FinChatPage loaderData={{ wid: 1 }} />);

    expect(capturedComposerProps.onSend).toBeDefined();

    await capturedComposerProps.onSend?.({
      content: "Analyze this document",
    });

    expect(mockMessage.error).toHaveBeenCalledTimes(1);
    expect(mockMessage.error).toHaveBeenCalledWith("Failed to send message");
  });

  it("creates a conversation first if none exists and reports error when sendMessage fails", async () => {
    mockCreateConversationMutateAsync.mockResolvedValueOnce({ id: 202 });
    mockMutateAsync.mockRejectedValueOnce(new Error("Network timeout"));

    renderToString(<FinChatPage loaderData={{ wid: 1 }} />);

    await capturedComposerProps.onSend?.({
      content: "Hello",
    });

    expect(mockMessage.error).toHaveBeenCalledWith("Network timeout");
  });
});
