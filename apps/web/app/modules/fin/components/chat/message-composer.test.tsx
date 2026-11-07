import type * as antd from "antd";
import type React from "react";
import { renderToString } from "react-dom/server";
import type * as reactI18next from "react-i18next";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { env } from "~/lib/env";
import {
  detectMimeType,
  MessageComposer,
  type MessageComposerHandle,
} from "./message-composer";

interface MockButtonProps {
  children?: React.ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  "aria-label"?: string;
  loading?: boolean;
  type?: string;
  icon?: React.ReactNode;
  "data-testid"?: string;
}

const mockMessage = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
};

let buttonPropsList: MockButtonProps[] = [];

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof reactI18next>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, defaultValue?: unknown) => {
        if (typeof defaultValue === "string") return defaultValue;
        if (
          typeof defaultValue === "object" &&
          defaultValue !== null &&
          "defaultValue" in defaultValue
        ) {
          const opts = defaultValue as { defaultValue?: unknown };
          if (typeof opts.defaultValue === "string") {
            return opts.defaultValue;
          }
        }
        return key;
      },
      i18n: { language: "en" },
    }),
  };
});

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
    Tooltip: ({
      children,
      title,
    }: {
      children: React.ReactNode;
      title: React.ReactNode;
    }) => <div data-tooltip={String(title)}>{children}</div>,
    App: {
      ...actual.App,
      useApp: () => ({
        message: mockMessage,
      }),
    },
  };
});

function createMockFile(name: string, size: number, type = ""): File {
  return new File([new Uint8Array(size)], name, { type });
}

describe("MessageComposer", () => {
  beforeEach(() => {
    buttonPropsList = [];
    vi.clearAllMocks();
  });

  describe("detectMimeType", () => {
    it("preserves explicit non-generic MIME types", () => {
      expect(detectMimeType({ name: "file.png", type: "image/png" })).toBe(
        "image/png",
      );
      expect(detectMimeType({ name: "doc.pdf", type: "application/pdf" })).toBe(
        "application/pdf",
      );
      expect(detectMimeType({ name: "image.custom", type: "image/jpeg" })).toBe(
        "image/jpeg",
      );
    });

    it("falls back to extension for missing type", () => {
      expect(detectMimeType({ name: "photo.png", type: "" })).toBe("image/png");
      expect(detectMimeType({ name: "receipt.PNG" })).toBe("image/png");
      expect(detectMimeType({ name: "invoice.jpg", type: "" })).toBe(
        "image/jpeg",
      );
      expect(detectMimeType({ name: "document.jpeg", type: "" })).toBe(
        "image/jpeg",
      );
      expect(detectMimeType({ name: "banner.webp", type: "" })).toBe(
        "image/webp",
      );
      expect(detectMimeType({ name: "report.pdf", type: "" })).toBe(
        "application/pdf",
      );
      expect(detectMimeType({ name: "statement.PDF" })).toBe("application/pdf");
    });

    it("falls back to extension for generic application/octet-stream type", () => {
      expect(
        detectMimeType({
          name: "scanned.pdf",
          type: "application/octet-stream",
        }),
      ).toBe("application/pdf");
      expect(
        detectMimeType({
          name: "screenshot.png",
          type: "application/octet-stream",
        }),
      ).toBe("image/png");
    });

    it("returns empty string or original type for unsupported extensions", () => {
      expect(detectMimeType({ name: "notes.txt", type: "" })).toBe("");
      expect(
        detectMimeType({
          name: "archive.zip",
          type: "application/octet-stream",
        }),
      ).toBe("application/octet-stream");
      expect(detectMimeType({ name: "noextension", type: "" })).toBe("");
    });
  });

  describe("file staging and preview chip", () => {
    it("stages an image file without triggering onSend", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          onStagedFileChange={onStagedFileChange}
          composerRef={composerRef}
        />,
      );

      const file = createMockFile("receipt.png", 2048, "image/png");
      composerRef.current?.stageFile(file);

      expect(onSend).not.toHaveBeenCalled();
      expect(onStagedFileChange).toHaveBeenCalledWith(
        expect.objectContaining({ name: "receipt.png" }),
      );
    });

    it("stages a file with blank type using extension fallback", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          onStagedFileChange={onStagedFileChange}
          composerRef={composerRef}
        />,
      );

      const file = createMockFile("invoice.pdf", 4096, "");
      composerRef.current?.stageFile(file);

      expect(onSend).not.toHaveBeenCalled();
      expect(onStagedFileChange).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "invoice.pdf",
          type: "application/pdf",
        }),
      );
    });

    it("rejects unsupported file types with error toast and does not stage", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          onStagedFileChange={onStagedFileChange}
          composerRef={composerRef}
        />,
      );

      const file = createMockFile("data.csv", 1024, "text/csv");
      composerRef.current?.stageFile(file);

      expect(mockMessage.error).toHaveBeenCalledWith(
        "Only images and PDF files are allowed",
      );
      expect(onStagedFileChange).not.toHaveBeenCalled();
      expect(onSend).not.toHaveBeenCalled();
    });

    it("rejects files larger than max size with error toast and does not stage", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          onStagedFileChange={onStagedFileChange}
          composerRef={composerRef}
        />,
      );

      const oversizedBytes = (env.attachmentMaxSizeMB + 1) * 1024 * 1024;
      const file = createMockFile(
        "huge.pdf",
        oversizedBytes,
        "application/pdf",
      );
      composerRef.current?.stageFile(file);

      expect(mockMessage.error).toHaveBeenCalledWith(
        `File exceeds maximum size of ${env.attachmentMaxSizeMB}MB`,
      );
      expect(onStagedFileChange).not.toHaveBeenCalled();
      expect(onSend).not.toHaveBeenCalled();
    });

    it("renders preview chip with name, formatted size, and remove button when staged", () => {
      const onSend = vi.fn();
      const file = createMockFile("receipt.png", 1572864, "image/png");

      const html = renderToString(
        <MessageComposer onSend={onSend} initialStagedFile={file} />,
      );

      expect(html).toContain('data-testid="staged-file-preview"');
      expect(html).toContain("receipt.png");
      expect(html).toContain("1.5 MB");

      const removeBtn = buttonPropsList.find(
        (b) => b["aria-label"] === "Remove attachment",
      );
      expect(removeBtn).toBeDefined();
    });

    it("clicking remove button unstages the file", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const file = createMockFile("receipt.png", 2048, "image/png");

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialStagedFile={file}
          onStagedFileChange={onStagedFileChange}
        />,
      );

      const removeBtn = buttonPropsList.find(
        (b) => b["aria-label"] === "Remove attachment",
      );
      expect(removeBtn).toBeDefined();

      removeBtn?.onClick?.();
      expect(onStagedFileChange).toHaveBeenCalledWith(null);
    });

    it("does not render preview chip when no file is staged", () => {
      const onSend = vi.fn();
      const html = renderToString(<MessageComposer onSend={onSend} />);
      expect(html).not.toContain('data-testid="staged-file-preview"');
    });
  });

  describe("Send button disabled condition and tooltip", () => {
    it("disables send button when content is empty and no file staged", () => {
      const onSend = vi.fn();
      renderToString(<MessageComposer onSend={onSend} />);

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      expect(sendBtn).toBeDefined();
      expect(sendBtn?.disabled).toBe(true);
    });

    it("enables send button when content has text even without staged file", () => {
      const onSend = vi.fn();
      renderToString(
        <MessageComposer onSend={onSend} initialContent="Hello agent" />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      expect(sendBtn).toBeDefined();
      expect(sendBtn?.disabled).toBe(false);
    });

    it("enables send button when file is staged even without text", () => {
      const onSend = vi.fn();
      const file = createMockFile("receipt.png", 1024, "image/png");
      renderToString(
        <MessageComposer onSend={onSend} initialStagedFile={file} />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      expect(sendBtn).toBeDefined();
      expect(sendBtn?.disabled).toBe(false);
    });

    it("disables send button when disabled prop is true", () => {
      const onSend = vi.fn();
      const file = createMockFile("receipt.png", 1024, "image/png");
      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="Hello"
          initialStagedFile={file}
          disabled
        />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      expect(sendBtn?.disabled).toBe(true);
    });

    it("disables send button and shows loading when isSending is true", () => {
      const onSend = vi.fn();
      renderToString(
        <MessageComposer onSend={onSend} initialContent="Hello" isSending />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      expect(sendBtn?.disabled).toBe(true);
      expect(sendBtn?.loading).toBe(true);
    });

    it("renders tooltip without auto-sends wording", () => {
      const onSend = vi.fn();
      const html = renderToString(<MessageComposer onSend={onSend} />);
      expect(html).toContain("Attach image or PDF");
      expect(html).not.toContain("auto-sends");
    });
  });

  describe("submission with click and Enter key", () => {
    it("submits text and staged file together on Send button click", () => {
      const onSend = vi.fn();
      const onStagedFileChange = vi.fn();
      const file = createMockFile("receipt.png", 2048, "image/png");

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="Please process this"
          initialStagedFile={file}
          onStagedFileChange={onStagedFileChange}
        />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      sendBtn?.onClick?.();

      expect(onSend).toHaveBeenCalledWith({
        content: "Please process this",
        files: [file],
      });
      expect(onStagedFileChange).toHaveBeenCalledWith(null);
    });

    it("submits only staged file with undefined content when text is blank", () => {
      const onSend = vi.fn();
      const file = createMockFile("invoice.pdf", 4096, "application/pdf");

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="   "
          initialStagedFile={file}
        />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      sendBtn?.onClick?.();

      expect(onSend).toHaveBeenCalledWith({
        content: undefined,
        files: [file],
      });
    });

    it("submits only text with undefined files when no file is staged", () => {
      const onSend = vi.fn();

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="Check cashflow status"
        />,
      );

      const sendBtn = buttonPropsList.find((b) => b["aria-label"] === "Send");
      sendBtn?.onClick?.();

      expect(onSend).toHaveBeenCalledWith({
        content: "Check cashflow status",
        files: undefined,
      });
    });

    it("submits on Enter key without Shift", () => {
      const onSend = vi.fn();
      const file = createMockFile("slip.png", 1024, "image/png");
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="ATM slip"
          initialStagedFile={file}
          composerRef={composerRef}
        />,
      );

      const preventDefault = vi.fn();
      composerRef.current?.submitKeyDown({
        key: "Enter",
        shiftKey: false,
        preventDefault,
      });

      expect(preventDefault).toHaveBeenCalled();
      expect(onSend).toHaveBeenCalledWith({
        content: "ATM slip",
        files: [file],
      });
    });

    it("does not submit on Shift+Enter (allows newline)", () => {
      const onSend = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="Line 1"
          composerRef={composerRef}
        />,
      );

      const preventDefault = vi.fn();
      composerRef.current?.submitKeyDown({
        key: "Enter",
        shiftKey: true,
        preventDefault,
      });

      expect(preventDefault).not.toHaveBeenCalled();
      expect(onSend).not.toHaveBeenCalled();
    });

    it("does not submit on Enter when disabled or sending", () => {
      const onSend = vi.fn();
      const composerRef: { current: MessageComposerHandle | null } = {
        current: null,
      };

      renderToString(
        <MessageComposer
          onSend={onSend}
          initialContent="Hello"
          disabled
          composerRef={composerRef}
        />,
      );

      composerRef.current?.submitKeyDown({
        key: "Enter",
        shiftKey: false,
      });

      expect(onSend).not.toHaveBeenCalled();
    });
  });
});
