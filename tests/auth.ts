import { auth, type User } from "@/auth";
import { serverDB } from "@/database";

export const authCtx = await auth.$context;
export const authTest = authCtx.test;

export async function ensureTestUser(): Promise<User> {
  const user = authTest.createUser({
    name: "Test User",
    email: "test@example.com",
    emailVerified: true,
  });

  const row = await serverDB.query.users.findFirst({
    where: { email: user.email },
  });
  if (row) return row as unknown as User;
  return authTest.saveUser(user) as unknown as User;
}
