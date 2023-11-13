import type { UserPayload } from "@omed/db";
import { users } from "@omed/db";
import bcrypt from "bcryptjs";

export async function login({
  email,
  password,
}: {
  email: string;
  password: string;
}) {
  const user = await users.findUnique({ email }, "withPassword");

  if (!user) return null;

  const validated = await bcrypt.compare(password, user.password.hash);
  if (!validated) return null;

  return user;
}

export async function register(payload: UserPayload) {
  if (payload.password) {
    const hash = await bcrypt.hash(payload.password, 12);
    payload.password = hash;
  }

  users.upsert(payload);
}
