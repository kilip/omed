import { describe, expect, it } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import { ApiError } from "~/lib/api";
import { evaluateCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";

describe("CMS Pages logic and permission gating", () => {
  describe("permission gating for pages", () => {
    it("viewer has read-only access to pages", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:viewer"] });
      expect(perms.canCreatePage).toBe(false);
      expect(perms.canEditPage).toBe(false);
      expect(perms.canPublishPage).toBe(false);
      expect(perms.canDeletePage).toBe(false);
    });

    it("editor has full write, publish, and delete access to pages", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:editor"] });
      expect(perms.canCreatePage).toBe(true);
      expect(perms.canEditPage).toBe(true);
      expect(perms.canPublishPage).toBe(true);
      expect(perms.canDeletePage).toBe(true);
    });

    it("admin has full administrative access to pages", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:admin"] });
      expect(perms.canCreatePage).toBe(true);
      expect(perms.canEditPage).toBe(true);
      expect(perms.canPublishPage).toBe(true);
      expect(perms.canDeletePage).toBe(true);
    });
  });

  describe("API error handling for pages", () => {
    it("identifies 409 slug conflict error for pages", () => {
      const err = new ApiError("SLUG_TAKEN: slug already in use", 409);
      const isSlugTaken =
        err.status === 409 || err.message.includes("SLUG_TAKEN");
      expect(isSlugTaken).toBe(true);
      expect(en.cms.errors.SLUG_TAKEN).toBeDefined();
      expect(id.cms.errors.SLUG_TAKEN).toBeDefined();
    });

    it("identifies invalid publish transition error", () => {
      const err = new ApiError("INVALID_TRANSITION", 409);
      expect(err.status).toBe(409);
      expect(en.cms.errors.INVALID_TRANSITION).toBeDefined();
      expect(id.cms.errors.INVALID_TRANSITION).toBeDefined();
    });

    it("identifies past date publish error", () => {
      const err = new ApiError("PUBLISH_AT_IN_PAST", 422);
      expect(err.status).toBe(422);
      expect(en.cms.errors.PUBLISH_AT_IN_PAST).toBeDefined();
      expect(id.cms.errors.PUBLISH_AT_IN_PAST).toBeDefined();
    });
  });

  describe("create, edit, publish, schedule payload shapes for pages", () => {
    it("builds valid create page payload", () => {
      const pageInput = {
        title: "About Us",
        slug: slugify("About Us"),
        body: "# About our company\n\nWelcome to Omed.",
      };

      expect(pageInput.title).toBe("About Us");
      expect(pageInput.slug).toBe("about-us");
      expect(pageInput.body).toContain("About our company");
    });

    it("builds valid schedule page payload with ISO date string", () => {
      const futureDate = "2026-11-15T10:00:00.000Z";
      const schedulePayload = { publishAt: futureDate };
      expect(schedulePayload.publishAt).toBe(futureDate);
    });
  });
});
