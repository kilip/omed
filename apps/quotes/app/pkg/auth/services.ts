import type { UserPayload } from "@omed/db";
import { users } from "@omed/db";
import { withZod } from "@remix-validated-form/with-zod";
import bcrypt from "bcryptjs";
import { AuthorizationError } from "remix-auth";
import z from "zod";

// app/services/auth.server.ts

export const LoginValidator = withZod(
  z.object({
    email: z.string().min(1, "Please enter your registered mail address"),
    password: z.string().min(1, "Please entel your password!"),
  })
);

export async function login({
  email,
  password,
}: {
  email: string;
  password: string;
}) {
  const user = await users.findUnique({ email }, "withPassword");
  const errorMsg = "Incorrect email or password";

  if (!user) return { error: new AuthorizationError(errorMsg) };

  const validated = await bcrypt.compare(password, user.password.hash);
  if (!validated) return { error: new AuthorizationError(errorMsg) };

  return user;
}

export async function register(payload: UserPayload) {
  if (payload.password) {
    const hash = await bcrypt.hash(payload.password, 12);
    payload.password = hash;
  }

  users.upsert(payload);
}
