import type React from "react";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import type { JobStatus } from "~/modules/agent/hooks/analysis-queue";
import {
  formatFileSize,
  getFileExtension,
  isImageAttachment,
  isPreviewableAttachment,
  PendingAttachmentsPicker,
  validateAttachmentFile,
} from "./transaction-attachments";

interface MockUploadProps {
  children?: React.ReactNode;
  beforeUpload?: (file: File) => boolean | Promise<unknown>;
  disabled?: boolean;
}

interface MockButtonProps {
  children?: React.ReactNode;
  disabled?: boolean;
  danger?: boolean;
  onClick?: () => void;
}

interface MockPopconfirmProps {
  children?: React.ReactNode;
  title?: React.ReactNode;
  onConfirm?: () => void;
  disabled?: boolean;
}

let lastUploadProps: MockUploadProps | null = null;
let buttonPropsList: MockButtonProps[] = [];
let popconfirmPropsList: MockPopconfirmProps[] = [];

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, options?: { defaultValue?: string } | string) => {
        if (typeof options === "object" && options?.defaultValue) {
          return options.defaultValue;
        }
        if (typeof options === "string") return options;
        return key;
      },
      i18n: { language: "en" },
    }),
  };
});

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return {
    ...actual,
    Upload: (props: MockUploadProps) => {
      lastUploadProps = props;
      return <div data-testid="upload">{props.children}</div>;
    },
    Button: (props: MockButtonProps) => {
      buttonPropsList.push(props);
      return (
        <button
          type="button"
          disabled={props.disabled}
          onClick={props.onClick}
          data-danger={props.danger}
        >
          {props.children}
        </button>
      );
    },
    Popconfirm: (props: MockPopconfirmProps) => {
      popconfirmPropsList.push(props);
      return (
        <div data-testid="popconfirm" data-title={props.title}>
          {props.children}
        </div>
      );
    },
    App: {
      ...actual.App,
      useApp: () => ({
        message: {
          success: vi.fn(),
          error: vi.fn(),
          warning: vi.fn(),
          info: vi.fn(),
        },
      }),
    },
  };
});

describe("transaction-attachments helpers", () => {
  describe("formatFileSize", () => {
    it("formats bytes, kilobytes, and megabytes", () => {
      expect(formatFileSize(0)).toBe("0 B");
      expect(formatFileSize(512)).toBe("512 B");
      expect(formatFileSize(2048)).toBe("2.0 KB");
      expect(formatFileSize(1572864)).toBe("1.5 MB");
    });
  });

  describe("getFileExtension", () => {
    it("extracts normalized lowercase extension", () => {
      expect(getFileExtension("Invoice.PDF")).toBe(".pdf");
      expect(getFileExtension("archive.tar.gz")).toBe(".gz");
      expect(getFileExtension("noext")).toBe("");
      expect(getFileExtension(".hidden")).toBe("");
    });
  });

  describe("validateAttachmentFile", () => {
    const allowed = [".pdf", ".png", ".csv"];

    it("accepts valid file within size limit and allowed extension", () => {
      expect(
        validateAttachmentFile({ name: "receipt.pdf", size: 1024 }, allowed, 2),
      ).toEqual({ valid: true });
    });

    it("rejects empty file", () => {
      expect(
        validateAttachmentFile({ name: "receipt.pdf", size: 0 }, allowed, 2),
      ).toEqual({ valid: false, reason: "empty" });
    });

    it("rejects file exceeding maxSizeMB", () => {
      expect(
        validateAttachmentFile(
          { name: "receipt.pdf", size: 3 * 1024 * 1024 },
          allowed,
          2,
        ),
      ).toEqual({ valid: false, reason: "too_large" });
    });

    it("rejects file with disallowed extension", () => {
      expect(
        validateAttachmentFile({ name: "script.exe", size: 128 }, allowed, 2),
      ).toEqual({ valid: false, reason: "invalid_type" });
    });
  });

  describe("isPreviewableAttachment & isImageAttachment", () => {
    it("identifies images and PDFs as previewable", () => {
      expect(
        isPreviewableAttachment({
          name: "receipt.png",
          mimeType: "image/png",
        }),
      ).toBe(true);
      expect(
        isPreviewableAttachment({
          name: "invoice.pdf",
          mimeType: "application/pdf",
        }),
      ).toBe(true);
      expect(
        isPreviewableAttachment({
          name: "sheet.xlsx",
          mimeType:
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      ).toBe(false);
    });

    it("distinguishes images from PDFs", () => {
      expect(
        isImageAttachment({ name: "photo.jpg", mimeType: "image/jpeg" }),
      ).toBe(true);
      expect(
        isImageAttachment({ name: "doc.pdf", mimeType: "application/pdf" }),
      ).toBe(false);
    });
  });

  describe("PendingAttachmentsPicker", () => {
    beforeEach(() => {
      lastUploadProps = null;
      buttonPropsList = [];
      popconfirmPropsList = [];
    });

    const pngFile = new File(["dummy png"], "receipt.png", {
      type: "image/png",
    });
    const csvFile = new File(["col1,col2"], "statement.csv", {
      type: "text/csv",
    });

    it("calls onFileAdded for each accepted file and not for rejected or duplicate files", () => {
      const onChange = vi.fn();
      const onFileAdded = vi.fn();
      const existingFile = new File(["existing"], "existing.pdf", {
        type: "application/pdf",
      });

      renderToString(
        <PendingAttachmentsPicker
          files={[existingFile]}
          onChange={onChange}
          onFileAdded={onFileAdded}
        />,
      );

      expect(lastUploadProps).not.toBeNull();

      // 1. Accepted file
      const acceptedFile = new File(["new file"], "new.png", {
        type: "image/png",
      });
      const acceptedResult = lastUploadProps.beforeUpload(acceptedFile);
      expect(acceptedResult).toBe(false);
      expect(onChange).toHaveBeenCalledWith([existingFile, acceptedFile]);
      expect(onFileAdded).toHaveBeenCalledWith(acceptedFile);

      onChange.mockClear();
      onFileAdded.mockClear();

      // 2. Duplicate file (same name and size)
      const dupFile = new File(["existing"], "existing.pdf", {
        type: "application/pdf",
      });
      const dupResult = lastUploadProps.beforeUpload(dupFile);
      expect(dupResult).toBe(false);
      expect(onChange).not.toHaveBeenCalled();
      expect(onFileAdded).not.toHaveBeenCalled();

      onChange.mockClear();
      onFileAdded.mockClear();

      // 3. Rejected file (invalid type)
      const invalidTypeFile = new File(["binary"], "bad.exe", {
        type: "application/x-msdownload",
      });
      const rejectedResult = lastUploadProps.beforeUpload(invalidTypeFile);
      expect(rejectedResult).toBe(false);
      expect(onChange).not.toHaveBeenCalled();
      expect(onFileAdded).not.toHaveBeenCalled();

      // 4. Rejected file (too large > 10MB)
      const largeFile = new File(
        [new Uint8Array(11 * 1024 * 1024)],
        "large.pdf",
        { type: "application/pdf" },
      );
      lastUploadProps.beforeUpload(largeFile);
      expect(onChange).not.toHaveBeenCalled();
      expect(onFileAdded).not.toHaveBeenCalled();
    });

    it("renders without AI button or status when analysis is omitted and existing tests pass", () => {
      const html = renderToString(
        <PendingAttachmentsPicker files={[pngFile]} onChange={vi.fn()} />,
      );

      expect(html).toContain("receipt.png");
      expect(html).toContain("Delete");
      expect(html).not.toContain("Analyze with AI");
      expect(html).not.toContain("Analisa dengan AI");
      expect(html).not.toContain("ant-tag");

      // No AI button rendered
      const aiBtn = buttonPropsList.find((p) =>
        String(p.children).includes("Analyze with AI"),
      );
      expect(aiBtn).toBeUndefined();

      // Delete button is present
      const deleteBtn = buttonPropsList.find((p) => p.danger);
      expect(deleteBtn).toBeDefined();
    });

    it("renders status badge and enabled 'Analyze with AI' button for analyzable files calling onAnalyze", () => {
      const onAnalyze = vi.fn();
      const statusMap = new Map<File, JobStatus>([[pngFile, "queued"]]);

      const html = renderToString(
        <PendingAttachmentsPicker
          files={[pngFile]}
          onChange={vi.fn()}
          analysis={{
            statusOf: (file) => statusMap.get(file),
            onAnalyze,
          }}
        />,
      );

      expect(html).toContain("receipt.png");
      expect(html).toContain("Queued");
      expect(html).toContain("Analyze with AI");

      const analyzeBtn = buttonPropsList.find((p) =>
        String(p.children).includes("Analyze with AI"),
      );
      expect(analyzeBtn).toBeDefined();
      expect(analyzeBtn.disabled).toBe(false);

      analyzeBtn.onClick();
      expect(onAnalyze).toHaveBeenCalledWith(pngFile);
    });

    it("disables 'Analyze with AI' button while that file is analyzing", () => {
      const onAnalyze = vi.fn();
      const html = renderToString(
        <PendingAttachmentsPicker
          files={[pngFile]}
          onChange={vi.fn()}
          analysis={{
            statusOf: () => "analyzing",
            onAnalyze,
          }}
        />,
      );

      expect(html).toContain("Analyzing...");
      const analyzeBtn = buttonPropsList.find((p) =>
        String(p.children).includes("Analyze with AI"),
      );
      expect(analyzeBtn).toBeDefined();
      expect(analyzeBtn.disabled).toBe(true);
    });

    it("renders status for done, failed, and paused states", () => {
      const statuses: JobStatus[] = ["done", "failed", "paused"];
      for (const status of statuses) {
        buttonPropsList = [];
        const html = renderToString(
          <PendingAttachmentsPicker
            files={[pngFile]}
            onChange={vi.fn()}
            analysis={{
              statusOf: () => status,
              onAnalyze: vi.fn(),
            }}
          />,
        );
        if (status === "done") {
          expect(html).toContain("Analyzed");
        } else if (status === "failed") {
          expect(html).toContain("Failed");
        } else if (status === "paused") {
          expect(html).toContain("Paused");
        }
        const analyzeBtn = buttonPropsList.find((p) =>
          String(p.children).includes("Analyze with AI"),
        );
        expect(analyzeBtn.disabled).toBe(false);
      }
    });

    it("renders 'Re-analyze' button wrapped in Popconfirm (and no redundant 'Analyze with AI' button) when onReanalyze is provided and status is done", () => {
      const onAnalyze = vi.fn();
      const onReanalyze = vi.fn();

      const html = renderToString(
        <PendingAttachmentsPicker
          files={[pngFile]}
          onChange={vi.fn()}
          analysis={{
            statusOf: () => "done",
            onAnalyze,
            onReanalyze,
          }}
        />,
      );

      expect(html).toContain("Re-analyze");
      expect(html).not.toContain("Analyze with AI");

      const aiBtn = buttonPropsList.find((p) =>
        String(p.children).includes("Analyze with AI"),
      );
      expect(aiBtn).toBeUndefined();

      // Popconfirm check
      expect(popconfirmPropsList).toHaveLength(1);
      const popconfirm = popconfirmPropsList[0];
      expect(popconfirm.title).toContain("Re-analyzing will overwrite");

      // Confirm action
      popconfirm.onConfirm();
      expect(onReanalyze).toHaveBeenCalledWith(pngFile);
    });

    it("does not render 'Re-analyze' button when status is not done", () => {
      const html = renderToString(
        <PendingAttachmentsPicker
          files={[pngFile]}
          onChange={vi.fn()}
          analysis={{
            statusOf: () => "queued",
            onAnalyze: vi.fn(),
            onReanalyze: vi.fn(),
          }}
        />,
      );

      expect(html).not.toContain("Re-analyze");
      expect(popconfirmPropsList).toHaveLength(0);
    });

    it("renders neither AI button nor status for non-analyzable files (e.g. .csv)", () => {
      const html = renderToString(
        <PendingAttachmentsPicker
          files={[csvFile]}
          onChange={vi.fn()}
          analysis={{
            statusOf: () => "done",
            onAnalyze: vi.fn(),
            onReanalyze: vi.fn(),
          }}
        />,
      );

      expect(html).toContain("statement.csv");
      expect(html).toContain("Delete");
      expect(html).not.toContain("Analyze with AI");
      expect(html).not.toContain("Re-analyze");
      expect(html).not.toContain("Analyzed");
      expect(html).not.toContain("ant-tag");
      expect(popconfirmPropsList).toHaveLength(0);

      // No AI button
      const aiBtn = buttonPropsList.find((p) =>
        String(p.children).includes("Analyze with AI"),
      );
      expect(aiBtn).toBeUndefined();

      // Delete button is present
      const deleteBtn = buttonPropsList.find((p) => p.danger);
      expect(deleteBtn).toBeDefined();
    });

    it("has matching i18n keys and non-empty translations in en and id", () => {
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(en.fin.cashflow.ai.analyzeButton).toBe("Analyze with AI");
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(id.fin.cashflow.ai.analyzeButton).toBe("Analisa dengan AI");
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(en.fin.cashflow.ai.reanalyzeButton).toBe("Re-analyze");
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(id.fin.cashflow.ai.reanalyzeButton).toBe("Analisa ulang");
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(en.fin.cashflow.ai.reanalyzeConfirm).toContain("overwrite");
      // @ts-expect-error ai might not be defined on cashflow yet
      expect(id.fin.cashflow.ai.reanalyzeConfirm).toContain("menimpa");

      const statusKeys: JobStatus[] = [
        "queued",
        "analyzing",
        "done",
        "failed",
        "paused",
      ];
      for (const key of statusKeys) {
        // @ts-expect-error ai might not be defined on cashflow yet
        expect(en.fin.cashflow.ai.status[key]).toBeDefined();
        // @ts-expect-error ai might not be defined on cashflow yet
        expect(id.fin.cashflow.ai.status[key]).toBeDefined();
        // @ts-expect-error ai might not be defined on cashflow yet
        expect(en.fin.cashflow.ai.status[key].length).toBeGreaterThan(0);
        // @ts-expect-error ai might not be defined on cashflow yet
        expect(id.fin.cashflow.ai.status[key].length).toBeGreaterThan(0);
      }
    });
  });
});
