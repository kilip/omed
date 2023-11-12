import { User } from "@prisma/client";
import db from "./db";

export type UserPayload = Pick<User, "name" | "email"> & {
  password?: string;
  avatarUrl?: string;
};

const upsert = async (
  { name, email, avatarUrl, password }: UserPayload,
  currentUser?: User
) => {
  const currentEmail = currentUser?.email ?? email;

  const user = await db.user.upsert({
    where: {
      email: currentEmail,
    },
    update: {
      name,
      avatarUrl,
      email,
    },
    create: {
      name,
      email,
      avatarUrl,
    },
  });

  if (password) {
    updatePassword(user, password);
  }
  return user;
};

const updatePassword = async (user: User, password: string) => {
  const userId = user.id;
  return await db.password.upsert({
    where: {
      userId,
    },
    update: {
      password,
    },
    create: {
      password,
      userId,
    },
  });
};

export const users = {
  upsert,
  updatePassword,
};
