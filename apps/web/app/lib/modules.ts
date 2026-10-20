import type { ComponentType } from "react";

export type ModuleMenuItem = {
  key: string;
  /** Path relative to the module root. Empty string is the module index. */
  path?: string;
  /** i18n key. */
  labelKey: string;
  icon?: ComponentType;
  /** Whether this menu item should be hidden in a personal workspace. */
  hideInPersonalWorkspace?: boolean;
  children?: ModuleMenuItem[];
};

export type ModuleMeta = {
  /** Matches the Go module name and the first URL segment. */
  id: string;
  labelKey: string;
  descriptionKey: string;
  icon: ComponentType;
  order?: number;
  menu: ModuleMenuItem[];
};

export function defineModule(meta: ModuleMeta): ModuleMeta {
  return meta;
}

const found = import.meta.glob<{ default: ModuleMeta }>(
  "../modules/*/module.ts",
  { eager: true },
);

export const modules: ModuleMeta[] = Object.values(found)
  .map((m) => m.default)
  .sort((a, b) => (a.order ?? 100) - (b.order ?? 100));

export const findModule = (pathname: string): ModuleMeta | undefined => {
  const first = pathname.split("/").filter(Boolean)[0];
  return modules.find((m) => m.id === first);
};
