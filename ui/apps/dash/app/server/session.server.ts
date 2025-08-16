import { createCookieSessionStorage } from "react-router";
import invariant from "tiny-invariant";
import type { AuthenticatedUser } from "~/types";

export type SessionData = {
  user: AuthenticatedUser;
};

type SessionFlashData = {
  error: string;
  success: string;
  warning: string;
  info: string;
};

const secrets = process.env.OMED_SECRET || "secret";
// invariant(secrets, "OMED_SECRET env not configured");

const { getSession, commitSession, destroySession } =
  createCookieSessionStorage<SessionData, SessionFlashData>({
    cookie: {
      name: "omed_session",
      httpOnly: true,
      // 7 days
      maxAge: 604800,
      path: "/",
      sameSite: "lax",
      secure: true,
      secrets: [secrets],
    },
  });

export { getSession, commitSession, destroySession };
