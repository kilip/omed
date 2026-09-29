import { Outlet, redirect } from "react-router";
import { appContext } from "~/context";
import { getSession, token } from "~/shared/auth";
import { DashboardLayout } from "~/shared/ui/DashboardLayout";
import type { Route } from "./+types/_dash";

async function fetchToken() {
  const { data } = await token();
  return data?.token ?? "";
}

export const authMiddleware: Route.ClientMiddlewareFunction = async (
  { context },
  next,
) => {
  let configured = false;
  try {
    context.get(appContext);
    configured = true;
  } catch (e: unknown) {
    configured = false;
  }
  if (!configured) {
    const { data, error } = await getSession();
    if (error || !data) {
      throw redirect("/login");
    }

    const { user } = data;
    const token = await fetchToken();
    context.set(appContext, { user, token, loading: false });
  }

  return next();
};

export const clientMiddleware: Route.ClientMiddlewareFunction[] = [
  authMiddleware,
];
export async function clientLoader({ context }: Route.ClientLoaderArgs) {
  const data = context.get(appContext);
  return { ...data };
}

export default function MainLayout({ loaderData }: Route.ComponentProps) {
  const { user } = loaderData;
  return (
    <DashboardLayout user={user}>
      <Outlet context={{ ...loaderData }} />
    </DashboardLayout>
  );
}
