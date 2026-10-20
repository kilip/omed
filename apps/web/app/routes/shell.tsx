import { useEffect } from "react";
import { Outlet, redirect, useNavigate, useRevalidator } from "react-router";
import { NavBar } from "~/components/NavBar";
import i18n, { setLanguage } from "~/i18n";
import { authClient } from "~/lib/auth-client";
import type { Lang } from "~/lib/format";
import {
  refreshWorkspaces,
  requireWorkspaceId,
  setActiveWorkspaceId,
  type Workspace,
} from "~/lib/workspace";
import type { Route } from "./+types/shell";

export async function clientLoader() {
  const { data } = await authClient.getSession();
  if (!data) throw redirect("/login");

  let workspaces: Workspace[] = [];
  let activeId: number | undefined;
  let apiError: string | undefined;
  try {
    workspaces = await refreshWorkspaces();
    activeId = await requireWorkspaceId();
  } catch (error) {
    apiError = error instanceof Error ? error.message : String(error);
  }

  return { user: data.user, workspaces, activeId, apiError };
}

export default function Shell({ loaderData }: Route.ComponentProps) {
  const { user, workspaces, activeId } = loaderData;
  const navigate = useNavigate();
  const revalidator = useRevalidator();

  useEffect(() => {
    if (
      user.locale &&
      (user.locale === "en" || user.locale === "id") &&
      i18n.language !== user.locale
    ) {
      setLanguage(user.locale as Lang);
    }
  }, [user.locale]);

  const signOut = async () => {
    await authClient.signOut();
    navigate("/login");
  };

  return (
    <div className="flex min-h-screen flex-col">
      <NavBar
        user={user}
        workspaces={workspaces}
        activeId={activeId}
        onWorkspaceChange={(id) => {
          setActiveWorkspaceId(id);
          void revalidator.revalidate();
        }}
        onSignOut={signOut}
      />
      <main className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </main>
    </div>
  );
}
