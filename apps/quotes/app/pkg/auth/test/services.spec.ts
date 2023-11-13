import type { UserWithPassword } from "@omed/db";
import users from "@omed/db/src/__mock__/users";
import bcrypt from "bcryptjs";
import { describe, expect, it, vi } from "vitest";
import { login } from "..";

vi.mock("@omed/db", async (importActual) => {
  const imports: typeof importActual = await importActual();
  return {
    ...imports,
    users: users,
  };
});

const hash = bcrypt.hashSync("password", 12);

describe("services", () => {
  it("login()", async () => {
    const expected = { password: { hash } } as unknown as UserWithPassword;
    users.findUnique.mockResolvedValue(expected);
    const user = await login({ email: "foobar", password: "password" });

    expect(users.findUnique).toBeCalled();
    expect(user).toEqual(expected);
  });
});
