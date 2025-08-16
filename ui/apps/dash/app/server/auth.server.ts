import { commitSession, getSession } from "./session.server";

export async function verifyUser(request: Request) {
  const session = await getSession(request.headers.get("Cookie"));

  const authUser = session.get("user");
  const headers = new Headers({
    "Set-Cookie": await commitSession(session),
  });

  return { authUser, headers };
}
