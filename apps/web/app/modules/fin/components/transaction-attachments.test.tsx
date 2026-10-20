import { describe, expect, it } from "vitest";
import {
  formatFileSize,
  getFileExtension,
  isImageAttachment,
  isPreviewableAttachment,
  validateAttachmentFile,
} from "./transaction-attachments";

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
});
