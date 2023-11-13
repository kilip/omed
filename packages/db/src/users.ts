import { Password, User } from "@prisma/client";
import { db } from "./db";

export type UserPayload = Pick<User, "name" | "email"> & {
  password?: string;
  avatarUrl?: string;
};

export type UserWithPassword = User & {
  password: Password;
};

export type UserResultType = {
  user: User;
  withPassword: UserWithPassword;
};

export async function upsert(
  { name, email, avatarUrl, password }: UserPayload,
  currentUser?: User
) {
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
}

export async function updatePassword(user: User, password: string) {
  const userId = user.id;
  return await db.password.upsert({
    where: {
      userId,
    },
    update: {
      hash: password,
    },
    create: {
      hash: password,
      userId,
    },
  });
}

type UserFindFirstResponse<T> = T extends keyof UserResultType
  ? UserResultType[T] | null
  : UserResultType["user"] | null;

async function findUnique<T extends keyof UserResultType | undefined>(
  filter: Pick<User, "email"> | Pick<User, "id">,
  type?: T
): Promise<UserFindFirstResponse<T>> {
  let include = {};

  if (type === "withPassword") {
    include = {
      password: true,
    };
  }

  const user = await db.user.findUnique({
    where: {
      ...filter,
    },
    include,
  });

  return user as UserFindFirstResponse<T>;
}

export const users = {
  upsert,
  updatePassword,
  findUnique,
};
