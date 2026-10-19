import type { components } from "@omed/openapi";
import { api, unwrapList } from "./api";

export type Workspace = components["schemas"]["model.WorkspaceResponse"];

const STORAGE_KEY = "omed.workspace";

let workspaces: Promise<Workspace[]> | null = null;

export function getStoredWorkspaceId(): number | undefined {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const id = raw ? Number(raw) : Number.NaN;
    return Number.isFinite(id) ? id : undefined;
  } catch {
    return undefined;
  }
}

export function setActiveWorkspaceId(id: number) {
  try {
    localStorage.setItem(STORAGE_KEY, String(id));
  } catch {
    // Ignore storage failures; the selection then lasts for this load only.
  }
}

/** Loads the workspaces the caller can see and remembers the promise. */
export function refreshWorkspaces(): Promise<Workspace[]> {
  workspaces = unwrapList(
    api.POST("/workspaces/search", { body: { limit: 100 } }),
  );
  return workspaces;
}

export type ActiveWorkspace = Workspace & { id: number };

/**
 * Resolves the active workspace. Falls back to the personal workspace, then
 * to the first one, and remembers the choice.
 */
export async function getActiveWorkspace(): Promise<ActiveWorkspace> {
  const list = await (workspaces ?? refreshWorkspaces());
  const stored = getStoredWorkspaceId();
  const active =
    list.find((w) => w.id === stored) ??
    list.find((w) => w.personal) ??
    list[0];

  if (active?.id === undefined) {
    throw new Error("No workspace available");
  }
  setActiveWorkspaceId(active.id);
  return active as ActiveWorkspace;
}

/**
 * Resolves the active workspace id. Falls back to the personal workspace, then
 * to the first one, and remembers the choice.
 */
export async function requireWorkspaceId(): Promise<number> {
  const active = await getActiveWorkspace();
  return active.id;
}
