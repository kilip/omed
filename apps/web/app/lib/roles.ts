import type { components } from "@omed/openapi";
import type { TFunction } from "i18next";
import { modules } from "./modules";

export type Role = components["schemas"]["model.RoleResponse"];

/**
 * i18n key of a role ID: "fin:accountant" becomes "role.fin.accountant".
 * The ":" is avoided because i18next reads it as a namespace separator.
 */
export const roleKey = (id: string) => `role.${id.replace(":", ".")}`;

/**
 * Display label of a role: the translation for its ID when there is one, then the
 * display name from the API, then the raw ID.
 */
export function roleLabel(t: TFunction, id: string, name?: string): string {
  return t(roleKey(id), { defaultValue: name || id });
}

/**
 * Display label of a module (e.g. "core" -> "Workspaces", "fin" -> "Finance").
 */
export function moduleLabel(t: TFunction, moduleId: string): string {
  const meta = modules.find((m) => m.id === moduleId);
  if (meta) return t(meta.labelKey);
  return t(`${moduleId}.label`, { defaultValue: moduleId });
}
