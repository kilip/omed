import { usePermissions } from "~/modules/core/hooks/use-permissions";

export const FIN_ROLES = {
  VIEWER: "fin:viewer",
  ACCOUNTANT: "fin:accountant",
  ADMIN: "fin:admin",
} as const;

export function useFinPermissions(wid: number) {
  const permissions = usePermissions(wid);

  const isOwner = Boolean(permissions.owner);
  const roles = permissions.roles ?? [];
  const isAdmin = isOwner || roles.includes(FIN_ROLES.ADMIN);
  const isAccountant = isAdmin || roles.includes(FIN_ROLES.ACCOUNTANT);
  const isViewer = isAccountant || roles.includes(FIN_ROLES.VIEWER);

  const hasPermission = (object: string, action: string) => {
    if (isAdmin) return true;
    return (permissions.permissions ?? []).some(
      (p) =>
        (p.object === object || p.object === "fin:*") &&
        (p.action === action || p.action === "*"),
    );
  };

  const canCreateTransaction =
    isAccountant || hasPermission("fin:transaction", "create");
  const canEditTransaction =
    isAccountant || hasPermission("fin:transaction", "update");
  const canDeleteTransaction =
    isAccountant || hasPermission("fin:transaction", "delete");

  const canCreateEntry = isAccountant || hasPermission("fin:entry", "create");
  const canEditEntry = isAccountant || hasPermission("fin:entry", "update");
  const canPostEntry = isAccountant || hasPermission("fin:entry", "update");
  const canDeleteEntry = isAccountant || hasPermission("fin:entry", "delete");
  const canVoidEntry = isAccountant || hasPermission("fin:entry", "update");
  const canReverseEntry = isAccountant || hasPermission("fin:entry", "create");

  return {
    ...permissions,
    isOwner,
    isAdmin,
    isAccountant,
    isViewer,
    canCreateTransaction,
    canEditTransaction,
    canDeleteTransaction,
    canCreateEntry,
    canEditEntry,
    canPostEntry,
    canDeleteEntry,
    canVoidEntry,
    canReverseEntry,
  };
}
