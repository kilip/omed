import { Outlet, redirect } from "react-router";
import { token as fetchToken, getSession } from "~/shared/auth";
import { getCachedAuth, setCachedAuth } from "~/shared/auth/cache";
import { type AuthContext, authContext } from "~/shared/contexts/auth";
import { AuthProvider } from "~/shared/providers/AuthProvider";
import { DashboardLayout } from "~/shared/ui/DashboardLayout";
import type { Route } from "./+types/_dash";

export const authMiddleware: Route.ClientMiddlewareFunction = async (
  { context },
  next,
) => {
  const cached = getCachedAuth();
  if (cached) {
    context.set(authContext, cached);
    return next();
  }

  const { data, error } = await getSession();
  if (data === null || error) throw redirect("/login");

  const result = await fetchToken();
  const value: AuthContext = {
    user: data.user,
    token: result.data?.token ?? "",
    session: data.session,
    loading: false,
  };
  setCachedAuth(value);
  context.set(authContext, value);
  return next();
};

export const clientMiddleware: Route.ClientMiddlewareFunction[] = [
  authMiddleware,
];

export async function clientLoader({ context }: Route.ClientLoaderArgs) {
  const data = context.get(authContext);
  return {
    ...data,
  };
}
export default function MainLayout({ loaderData }: Route.ComponentProps) {
  return (
    <AuthProvider
      user={loaderData.user}
      session={loaderData.session}
      token={loaderData.token}
      loading={loaderData.loading}
    >
      <DashboardLayout>
        <Outlet context={loaderData} />
      </DashboardLayout>
    </AuthProvider>
  );
}
