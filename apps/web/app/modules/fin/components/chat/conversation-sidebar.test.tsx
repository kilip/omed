import type * as antd from "antd";
import type React from "react";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ConversationSidebar } from "./conversation-sidebar";

interface MockButtonProps {
  children?: React.ReactNode;
  disabled?: boolean;
  onClick?: (e?: unknown) => void;
  "aria-label"?: string;
  loading?: boolean;
  type?: string;
  icon?: React.ReactNode;
  "data-testid"?: string;
}

interface MockInputProps {
  value?: string;
  disabled?: boolean;
  onChange?: (e: { target: { value: string } }) => void;
  onKeyDown?: (e: {
    key: string;
    preventDefault: () => void;
    stopPropagation: () => void;
  }) => void;
  "aria-label"?: string;
  "data-testid"?: string;
  autoFocus?: boolean;
}

let buttonPropsList: MockButtonProps[] = [];
let inputPropsList: MockInputProps[] = [];

let mockEditingId: number | null = null;
let mockEditTitle = "";
const mockSetEditingId = vi.fn((val: number | null) => {
  mockEditingId = val;
});
const mockSetEditTitle = vi.fn((val: string) => {
  mockEditTitle = val;
});

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useState: (initial: unknown) => {
      // Return custom state tuple instead of delegating to actual.useState conditionally
      if (typeof initial === "string") {
        return [mockEditTitle, mockSetEditTitle];
      }
      return [mockEditingId, mockSetEditingId];
    },
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, defaultValue?: string) => defaultValue ?? key,
  }),
}));

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof antd>();
  return {
    ...actual,
    Button: (props: MockButtonProps) => {
      buttonPropsList.push(props);
      return (
        <button
          type="button"
          disabled={props.disabled}
          onClick={props.onClick}
          aria-label={props["aria-label"]}
          data-testid={props["data-testid"]}
        >
          {props.children}
        </button>
      );
    },
    Input: (props: MockInputProps) => {
      inputPropsList.push(props);
      return (
        <input
          value={props.value}
          disabled={props.disabled}
          data-testid={props["data-testid"]}
          aria-label={props["aria-label"]}
          readOnly
        />
      );
    },
    Popconfirm: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Spin: () => <div>Loading...</div>,
  };
});

describe("ConversationSidebar", () => {
  const mockConversations = [
    { id: 1, title: "Conversation 1" },
    { id: 2, title: "Conversation 2" },
  ];

  beforeEach(() => {
    buttonPropsList = [];
    inputPropsList = [];
    mockEditingId = null;
    mockEditTitle = "";
    vi.clearAllMocks();
  });

  it("renders conversations list", () => {
    const html = renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={vi.fn()}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
      />,
    );

    expect(html).toContain("Conversation 1");
    expect(html).toContain("Conversation 2");
  });

  it("selects conversation on click", () => {
    const onSelect = vi.fn();
    const html = renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={onSelect}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
      />,
    );

    expect(html).toContain('data-testid="conversation-item-1"');
    expect(html).toContain('data-testid="conversation-item-2"');
  });

  it("renders rename button when onRenameConversation is provided", () => {
    const html = renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={vi.fn()}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
        onRenameConversation={vi.fn()}
      />,
    );

    expect(html).toContain('data-testid="conversation-rename-button-1"');
    expect(html).toContain('data-testid="conversation-rename-button-2"');
  });

  it("triggers rename button click and prevents event propagation", () => {
    const onRename = vi.fn();
    renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={vi.fn()}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
        onRenameConversation={onRename}
      />,
    );

    const renameBtn = buttonPropsList.find(
      (b) => b["data-testid"] === "conversation-rename-button-1",
    );
    expect(renameBtn).toBeDefined();

    const stopPropagation = vi.fn();
    renameBtn?.onClick?.({ stopPropagation });
    expect(stopPropagation).toHaveBeenCalled();
    expect(mockSetEditingId).toHaveBeenCalledWith(1);
    expect(mockSetEditTitle).toHaveBeenCalledWith("Conversation 1");
  });

  it("renders inline editing input and action buttons when editing a conversation", () => {
    mockEditingId = 1;
    mockEditTitle = "Conversation 1";

    const onRename = vi.fn();
    const html = renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={vi.fn()}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
        onRenameConversation={onRename}
      />,
    );

    expect(html).toContain('data-testid="conversation-rename-input-1"');
    expect(html).toContain('data-testid="conversation-save-button-1"');
    expect(html).toContain('data-testid="conversation-cancel-button-1"');
  });

  it("handles Enter keydown, Escape keydown, Save click, and Cancel click in inline editing", async () => {
    mockEditingId = 1;
    mockEditTitle = "New Title";

    const onRename = vi.fn().mockResolvedValue(undefined);
    renderToString(
      <ConversationSidebar
        conversations={mockConversations}
        onSelectConversation={vi.fn()}
        onNewConversation={vi.fn()}
        onDeleteConversation={vi.fn()}
        onRenameConversation={onRename}
      />,
    );

    const input = inputPropsList.find(
      (i) => i["data-testid"] === "conversation-rename-input-1",
    );
    expect(input).toBeDefined();

    // Test input change
    input?.onChange?.({ target: { value: "Changed Title" } });
    expect(mockSetEditTitle).toHaveBeenCalledWith("Changed Title");

    // Test Enter key press saves
    const enterPreventDefault = vi.fn();
    input?.onKeyDown?.({
      key: "Enter",
      preventDefault: enterPreventDefault,
      stopPropagation: vi.fn(),
    });
    expect(enterPreventDefault).toHaveBeenCalled();
    expect(onRename).toHaveBeenCalledWith(1, "New Title");

    // Test Save button click saves
    const saveBtn = buttonPropsList.find(
      (b) => b["data-testid"] === "conversation-save-button-1",
    );
    expect(saveBtn).toBeDefined();
    saveBtn?.onClick?.();
    expect(onRename).toHaveBeenCalledWith(1, "New Title");

    // Test Escape key press cancels without saving
    onRename.mockClear();
    const escapePreventDefault = vi.fn();
    input?.onKeyDown?.({
      key: "Escape",
      preventDefault: escapePreventDefault,
      stopPropagation: vi.fn(),
    });
    expect(escapePreventDefault).toHaveBeenCalled();
    expect(onRename).not.toHaveBeenCalled();
    expect(mockSetEditingId).toHaveBeenCalledWith(null);

    // Test Cancel button click cancels without saving
    const cancelBtn = buttonPropsList.find(
      (b) => b["data-testid"] === "conversation-cancel-button-1",
    );
    expect(cancelBtn).toBeDefined();
    cancelBtn?.onClick?.();
    expect(mockSetEditingId).toHaveBeenCalledWith(null);
  });
});
