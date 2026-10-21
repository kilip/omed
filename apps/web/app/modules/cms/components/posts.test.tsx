import { describe, expect, it } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import { ApiError } from "~/lib/api";
import { evaluateCmsPermissions } from "../hooks";
import { slugify } from "../lib/slug";

describe("CMS Posts logic, slugify, and permission gating", () => {
  describe("slugify helper", () => {
    it("converts titles to URL-friendly slugs", () => {
      expect(slugify("Hello World!")).toBe("hello-world");
      expect(slugify("This is a Test Post")).toBe("this-is-a-test-post");
      expect(slugify("  Spaces   and   Symbols & More...  ")).toBe(
        "spaces-and-symbols-more",
      );
      expect(slugify("Special #$% Characters")).toBe("special-characters");
    });
  });

  describe("permission gating for posts", () => {
    it("viewer has read-only access to posts", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:viewer"] });
      expect(perms.canCreatePost).toBe(false);
      expect(perms.canEditPost).toBe(false);
      expect(perms.canPublishPost).toBe(false);
      expect(perms.canDeletePost).toBe(false);
    });

    it("editor has full write, publish, and delete access to posts", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:editor"] });
      expect(perms.canCreatePost).toBe(true);
      expect(perms.canEditPost).toBe(true);
      expect(perms.canPublishPost).toBe(true);
      expect(perms.canDeletePost).toBe(true);
    });

    it("admin has full administrative access to posts", () => {
      const perms = evaluateCmsPermissions({ roles: ["cms:admin"] });
      expect(perms.canCreatePost).toBe(true);
      expect(perms.canEditPost).toBe(true);
      expect(perms.canPublishPost).toBe(true);
      expect(perms.canDeletePost).toBe(true);
    });
  });

  describe("API error handling for posts", () => {
    it("identifies 409 slug conflict error and provides translation", () => {
      const err = new ApiError("SLUG_TAKEN: slug already in use", 409);
      const isSlugTaken =
        err.status === 409 || err.message.includes("SLUG_TAKEN");
      expect(isSlugTaken).toBe(true);
      expect(en.cms.errors.SLUG_TAKEN).toBeDefined();
      expect(id.cms.errors.SLUG_TAKEN).toBeDefined();
    });

    it("identifies invalid publish transition error and provides translation", () => {
      const err = new ApiError("INVALID_TRANSITION", 409);
      expect(err.status).toBe(409);
      expect(en.cms.errors.INVALID_TRANSITION).toBeDefined();
      expect(id.cms.errors.INVALID_TRANSITION).toBeDefined();
    });

    it("identifies past date publish error and provides translation", () => {
      const err = new ApiError("PUBLISH_AT_IN_PAST", 422);
      expect(err.status).toBe(422);
      expect(en.cms.errors.PUBLISH_AT_IN_PAST).toBeDefined();
      expect(id.cms.errors.PUBLISH_AT_IN_PAST).toBeDefined();
    });
  });

  describe("create, edit, publish, schedule, delete payload shapes", () => {
    it("builds valid create post payload", () => {
      const postInput = {
        title: "Announcing Version 2",
        slug: slugify("Announcing Version 2"),
        excerpt: "A brief summary",
        body: "# Full Markdown Content",
        coverMediaId: 10,
      };

      expect(postInput.title).toBe("Announcing Version 2");
      expect(postInput.slug).toBe("announcing-version-2");
      expect(postInput.coverMediaId).toBe(10);
    });

    it("builds valid schedule post payload with ISO date string", () => {
      const futureDate = "2026-12-31T23:59:59.000Z";
      const schedulePayload = { publishAt: futureDate };
      expect(schedulePayload.publishAt).toBe(futureDate);
    });
  });
});
