import { commitSession, getSession } from "~/server/session.server";
import type { Route } from "./+types/login";
import { data, redirect, redirectDocument } from "react-router";
import Login from "~/components/Login";
import type { LoggedInResponse, Resource } from "~/types";

export async function loader({ request }: Route.LoaderArgs) {
  const session = await getSession(request.headers.get("Cookie"));
  if (session.has("user")) {
    return redirect("/");
  }

  return data(
    {
      error: session.get("error"),
    },
    {
      headers: {
        "Set-Cookie": await commitSession(session),
      },
    },
  );
}

export async function action({ request }: Route.ActionArgs) {
  const session = await getSession(request.headers.get("Cookie"));
  const form = await request.formData();
  const email = form.get("email");
  const password = form.get("password");

  const response = await fetch("http://localhost:3000/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application.json",
    },
    body: JSON.stringify({
      email,
      password,
    }),
  });

  if (response.status != 200) {
    session.flash("error", "Invalid username/password");
    return redirect("/login");
  }

  const json = (await response.json()) as Resource<LoggedInResponse>;
  session.set("user", {
    id: json.data.userId,
    name: json.data.name,
    avatar: json.data.avatar,
    token: json.data.token,
  });

  return redirect("/", {
    headers: {
      "Set-Cookie": await commitSession(session),
    },
  });
}

export default function LoginRoute({ loaderData }: Route.ComponentProps) {
  const { error } = loaderData;

  return <Login error={error} />;
}
