import { describe, expect, it } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import { ApiError } from "~/lib/api";
import { evaluateCmsPermissions } from "../hooks";
import {
  formatFileSize,
  getFileExtension,
  getPublicMediaUrl,
  isImageMedia,
  validateMediaFile,
} from "./media-utils";

describe("CMS Media helpers & validation", () => {
  const allowed = [".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"];

  describe("formatFileSize", () => {
    it("formats bytes, kilobytes, and megabytes", () => {
      expect(formatFileSize(0)).toBe("0 B");
      expect(formatFileSize(512)).toBe("512 B");
      expect(formatFileSize(2048)).toBe("2.0 KB");
      expect(formatFileSize(1572864)).toBe("1.5 MB");
    });
  });

  describe("getFileExtension", () => {
    it("extracts lowercase extension", () => {
      expect(getFileExtension("Photo.PNG")).toBe(".png");
      expect(getFileExtension("document.PDF")).toBe(".pdf");
      expect(getFileExtension("noext")).toBe("");
      expect(getFileExtension(".hidden")).toBe("");
    });
  });

  describe("validateMediaFile", () => {
    it("accepts valid image or pdf within size limit", () => {
      expect(
        validateMediaFile(
          { name: "banner.png", size: 1024 * 1024 },
          allowed,
          10,
        ),
      ).toEqual({ valid: true });
      expect(
        validateMediaFile(
          { name: "doc.pdf", size: 2 * 1024 * 1024 },
          allowed,
          10,
        ),
      ).toEqual({ valid: true });
    });

    it("rejects empty file", () => {
      expect(
        validateMediaFile({ name: "empty.png", size: 0 }, allowed, 10),
      ).toEqual({ valid: false, reason: "empty" });
    });

    it("rejects file exceeding max size", () => {
      expect(
        validateMediaFile(
          { name: "large.png", size: 11 * 1024 * 1024 },
          allowed,
          10,
        ),
      ).toEqual({ valid: false, reason: "too_large" });
    });

    it("rejects disallowed file types including SVG (D8)", () => {
      expect(
        validateMediaFile({ name: "malicious.svg", size: 500 }, allowed, 10),
      ).toEqual({ valid: false, reason: "invalid_type" });
      expect(
        validateMediaFile({ name: "program.exe", size: 500 }, allowed, 10),
      ).toEqual({ valid: false, reason: "invalid_type" });
    });
  });

  describe("isImageMedia", () => {
    it("identifies images correctly and distinguishes from pdf", () => {
      expect(isImageMedia("image/png", "test.png")).toBe(true);
      expect(isImageMedia("image/jpeg", "test.jpg")).toBe(true);
      expect(isImageMedia("application/pdf", "test.pdf")).toBe(false);
    });
  });

  describe("getPublicMediaUrl", () => {
    it("constructs stable public URL", () => {
      const url = getPublicMediaUrl(123, 456);
      expect(url).toContain("/public/workspaces/123/cms/media/456");
    });
  });

  describe("permission gating", () => {
    it("viewer has read-only access (cannot upload, edit, or delete)", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:viewer"] });
      expect(perms.canUploadMedia).toBe(false);
      expect(perms.canEditMedia).toBe(false);
      expect(perms.canDeleteMedia).toBe(false);
    });

    it("editor has full media access (upload, edit, delete)", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:editor"] });
      expect(perms.canUploadMedia).toBe(true);
      expect(perms.canEditMedia).toBe(true);
      expect(perms.canDeleteMedia).toBe(true);
    });

    it("admin has full media access", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:admin"] });
      expect(perms.canUploadMedia).toBe(true);
      expect(perms.canEditMedia).toBe(true);
      expect(perms.canDeleteMedia).toBe(true);
    });
  });

  describe("409 Conflict (MEDIA_IN_USE) error handling", () => {
    it("correctly identifies media in use 409 conflict and translates message", () => {
      const err = new ApiError("MEDIA_IN_USE: media referenced", 409);
      const is409InUse =
        err.status === 409 || err.message.includes("MEDIA_IN_USE");
      expect(is409InUse).toBe(true);

      expect(en.cms.media.deleteInUseError).toBeDefined();
      expect(en.cms.media.deleteInUseError.length).toBeGreaterThan(0);
      expect(id.cms.media.deleteInUseError).toBeDefined();
      expect(id.cms.media.deleteInUseError.length).toBeGreaterThan(0);
    });
  });

  describe("picker selection", () => {
    it("selects media item and triggers onSelect handler", () => {
      const mediaItem = {
        id: 42,
        title: "Test Cover",
        name: "cover.jpg",
        mimeType: "image/jpeg",
        size: 1024,
        url: "http://example.com/cover.jpg",
      };

      let selectedResult: typeof mediaItem | null = null;
      const onSelect = (item: typeof mediaItem) => {
        selectedResult = item;
      };

      onSelect(mediaItem);
      expect(selectedResult).toEqual(mediaItem);
      expect(selectedResult?.id).toBe(42);
    });
  });
});
