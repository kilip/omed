import type { User } from "@/auth";
import { type NewUser, serverDB } from "@/database";
import { getTestDB } from "@/database/core/getTestDB";
import { users } from "@/database/schema";

const db = await getTestDB();

export async function ensureTestUser(): Promise<User> {
  const user = {
    name: "Test User",
    email: "test@example.com",
    emailVerified: true,
  } satisfies NewUser;

  const existing = await serverDB.query.users.findFirst({
    where: { email: user.email },
  });

  if (existing) return existing as User;

  const [row] = (await db
    .insert(users)
    .values(user)
    .returning()) as unknown as User[];

  return row;
}
