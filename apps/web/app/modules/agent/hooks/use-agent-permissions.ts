import { usePermissions } from "~/modules/core/hooks/use-permissions";

export const AGENT_ROLES = {
  VIEWER: "agent:viewer",
  ADMIN: "agent:admin",
} as const;

export type PermissionsInput = {
  roles?: string[];
  permissions?: { object?: string; action?: string }[];
  owner?: boolean;
};

export function evaluateAgentPermissions(permissions: PermissionsInput) {
  const isOwner = Boolean(permissions.owner);
  const roles = permissions.roles ?? [];
  const isAdmin = isOwner || roles.includes(AGENT_ROLES.ADMIN);
  const isViewer = isAdmin || roles.includes(AGENT_ROLES.VIEWER);

  const hasPermission = (object: string, action: string) => {
    if (isAdmin) return true;
    return (permissions.permissions ?? []).some(
      (p) =>
        (p.object === object || p.object === "agent:*") &&
        (p.action === action || p.action === "*"),
    );
  };

  const canManage =
    isAdmin ||
    (hasPermission("agent:provider", "create") &&
      hasPermission("agent:provider", "update") &&
      hasPermission("agent:provider", "delete"));

  const canRead =
    canManage || isViewer || hasPermission("agent:provider", "read");

  return {
    ...permissions,
    isOwner,
    isAdmin,
    isViewer,
    canRead,
    canManage,
  };
}

export function useAgentPermissions(wid: number) {
  const permissions = usePermissions(wid);
  return evaluateAgentPermissions(permissions);
}
