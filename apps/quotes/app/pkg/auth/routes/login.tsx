import authenticator from "../authenticator";
import Login from "../components/Login";

import type { ActionFunctionArgs, LoaderFunctionArgs } from "@remix-run/node";

export async function loader({ request }: LoaderFunctionArgs) {
  return await authenticator.isAuthenticated(request, {
    successRedirect: "/",
  });
}

export async function action({ request }: ActionFunctionArgs) {
  return await authenticator.authenticate("user-pass", request, {
    successRedirect: "/",
    failureRedirect: "/login",
  });
}

export default function LoginRoute() {
  return (
    <main className="flex min-h-screen items-center justify-center">
      <div className="flex">
        <Login />
      </div>
    </main>
  );
}
