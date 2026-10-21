import { describe, expect, it } from "vitest";
import { CMS_ROLES, evaluateCmsPermissions } from "./use-cms-permissions";

describe("evaluateCmsPermissions", () => {
  it("gives full permissions to workspace owner", () => {
    const perms = evaluateCmsPermissions({ owner: true });
    expect(perms.isOwner).toBe(true);
    expect(perms.isAdmin).toBe(true);
    expect(perms.isEditor).toBe(true);
    expect(perms.isViewer).toBe(true);
    expect(perms.canCreatePost).toBe(true);
    expect(perms.canEditPost).toBe(true);
    expect(perms.canPublishPost).toBe(true);
    expect(perms.canDeletePost).toBe(true);
    expect(perms.canCreatePage).toBe(true);
    expect(perms.canEditPage).toBe(true);
    expect(perms.canPublishPage).toBe(true);
    expect(perms.canDeletePage).toBe(true);
    expect(perms.canUploadMedia).toBe(true);
    expect(perms.canEditMedia).toBe(true);
    expect(perms.canDeleteMedia).toBe(true);
  });

  it("gives full permissions to CMS admin", () => {
    const perms = evaluateCmsPermissions({ roles: [CMS_ROLES.ADMIN] });
    expect(perms.isAdmin).toBe(true);
    expect(perms.isEditor).toBe(true);
    expect(perms.isViewer).toBe(true);
    expect(perms.canCreatePost).toBe(true);
    expect(perms.canPublishPost).toBe(true);
    expect(perms.canCreatePage).toBe(true);
    expect(perms.canUploadMedia).toBe(true);
  });

  it("gives editor permissions to CMS editor", () => {
    const perms = evaluateCmsPermissions({ roles: [CMS_ROLES.EDITOR] });
    expect(perms.isAdmin).toBe(false);
    expect(perms.isEditor).toBe(true);
    expect(perms.isViewer).toBe(true);
    expect(perms.canCreatePost).toBe(true);
    expect(perms.canEditPost).toBe(true);
    expect(perms.canPublishPost).toBe(true);
    expect(perms.canDeletePost).toBe(true);
    expect(perms.canCreatePage).toBe(true);
    expect(perms.canEditPage).toBe(true);
    expect(perms.canPublishPage).toBe(true);
    expect(perms.canDeletePage).toBe(true);
    expect(perms.canUploadMedia).toBe(true);
    expect(perms.canEditMedia).toBe(true);
    expect(perms.canDeleteMedia).toBe(true);
  });

  it("gives read-only permissions to CMS viewer", () => {
    const perms = evaluateCmsPermissions({ roles: [CMS_ROLES.VIEWER] });
    expect(perms.isAdmin).toBe(false);
    expect(perms.isEditor).toBe(false);
    expect(perms.isViewer).toBe(true);
    expect(perms.canCreatePost).toBe(false);
    expect(perms.canEditPost).toBe(false);
    expect(perms.canPublishPost).toBe(false);
    expect(perms.canDeletePost).toBe(false);
    expect(perms.canCreatePage).toBe(false);
    expect(perms.canEditPage).toBe(false);
    expect(perms.canPublishPage).toBe(false);
    expect(perms.canDeletePage).toBe(false);
    expect(perms.canUploadMedia).toBe(false);
    expect(perms.canEditMedia).toBe(false);
    expect(perms.canDeleteMedia).toBe(false);
  });

  it("handles granular permissions", () => {
    const perms = evaluateCmsPermissions({
      roles: [],
      permissions: [
        { object: "cms:post", action: "create" },
        { object: "cms:media", action: "delete" },
      ],
    });
    expect(perms.isAdmin).toBe(false);
    expect(perms.isEditor).toBe(false);
    expect(perms.isViewer).toBe(false);
    expect(perms.canCreatePost).toBe(true);
    expect(perms.canEditPost).toBe(false);
    expect(perms.canDeleteMedia).toBe(true);
    expect(perms.canUploadMedia).toBe(false);
  });

  it("handles empty / default permissions", () => {
    const perms = evaluateCmsPermissions({});
    expect(perms.isOwner).toBe(false);
    expect(perms.isAdmin).toBe(false);
    expect(perms.isEditor).toBe(false);
    expect(perms.isViewer).toBe(false);
    expect(perms.canCreatePost).toBe(false);
  });
});
