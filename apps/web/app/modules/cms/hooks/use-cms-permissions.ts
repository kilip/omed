import { usePermissions } from "~/modules/core/hooks/use-permissions";

export const CMS_ROLES = {
  VIEWER: "cms:viewer",
  EDITOR: "cms:editor",
  ADMIN: "cms:admin",
} as const;

export type PermissionsInput = {
  roles?: string[];
  permissions?: { object?: string; action?: string }[];
  owner?: boolean;
};

export function evaluateCmsPermissions(permissions: PermissionsInput) {
  const isOwner = Boolean(permissions.owner);
  const roles = permissions.roles ?? [];
  const isAdmin = isOwner || roles.includes(CMS_ROLES.ADMIN);
  const isEditor = isAdmin || roles.includes(CMS_ROLES.EDITOR);
  const isViewer = isEditor || roles.includes(CMS_ROLES.VIEWER);

  const hasPermission = (object: string, action: string) => {
    if (isAdmin) return true;
    return (permissions.permissions ?? []).some(
      (p) =>
        (p.object === object || p.object === "cms:*") &&
        (p.action === action || p.action === "*"),
    );
  };

  const canCreatePost = isEditor || hasPermission("cms:post", "create");
  const canEditPost = isEditor || hasPermission("cms:post", "update");
  const canPublishPost = isEditor || hasPermission("cms:post", "update");
  const canDeletePost = isEditor || hasPermission("cms:post", "delete");

  const canCreatePage = isEditor || hasPermission("cms:page", "create");
  const canEditPage = isEditor || hasPermission("cms:page", "update");
  const canPublishPage = isEditor || hasPermission("cms:page", "update");
  const canDeletePage = isEditor || hasPermission("cms:page", "delete");

  const canUploadMedia = isEditor || hasPermission("cms:media", "create");
  const canEditMedia = isEditor || hasPermission("cms:media", "update");
  const canDeleteMedia = isEditor || hasPermission("cms:media", "delete");

  return {
    ...permissions,
    isOwner,
    isAdmin,
    isEditor,
    isViewer,
    canCreatePost,
    canEditPost,
    canPublishPost,
    canDeletePost,
    canCreatePage,
    canEditPage,
    canPublishPage,
    canDeletePage,
    canUploadMedia,
    canEditMedia,
    canDeleteMedia,
  };
}

export function useCmsPermissions(wid: number) {
  const permissions = usePermissions(wid);
  return evaluateCmsPermissions(permissions);
}
