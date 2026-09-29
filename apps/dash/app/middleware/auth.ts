import { redirect } from "react-router";
import { appContext } from "~/context";
import { getSession, token } from "~/shared/auth";
import type { Route } from "../+types/root";

async function fetchToken() {
  const { data } = await token();
  return data?.token ?? "";
}

export const authMiddleware: Route.ClientMiddlewareFunction = async (
  { context },
  next,
) => {
  const { data, error } = await getSession();
  if (error || !data) {
    throw redirect("/login");
  }

  const { user } = data;
  const token = await fetchToken();
  context.set(appContext, { user, token, loading: false });

  return next();
};
