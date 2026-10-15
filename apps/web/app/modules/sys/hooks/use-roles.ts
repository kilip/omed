import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { roleLabel } from "~/lib/roles";
import { rolesQueryOptions } from "../queries";

/**
 * Assignable roles of a workspace, plus a `label` resolver for any role ID and
 * ready-made options for an antd `Select`. Suspends until the roles are loaded;
 * route loaders should `ensureQueryData(rolesQueryOptions(wid))` first.
 */
export function useRoles(wid: number) {
  const { t } = useTranslation();
  const { data: roles } = useSuspenseQuery(rolesQueryOptions(wid));

  const label = useCallback(
    (id: string) => roleLabel(t, id, roles.find((r) => r.id === id)?.name),
    [t, roles],
  );

  const options = useMemo(
    () =>
      roles.flatMap((r) =>
        r.id ? [{ value: r.id, label: label(r.id), title: r.description }] : [],
      ),
    [roles, label],
  );

  return { roles, options, label };
}
