import { describe, expect, it, vi } from "vitest";
import { UserPayload, users } from "../src";
import db from "../src/__mock__/db";

vi.mock("../src/db", async () => {
  const mock = (await import("../src/__mock__/db")).default;
  return {
    default: mock,
  };
});

const createdAt = new Date();
const updatedAt = new Date();

describe("users service", () => {
  it("should upsert user with password", async () => {
    const newUser: UserPayload = {
      email: "test@example.com",
      name: "Test User",
      password: "password",
    };

    const expectedUser: any = {
      ...newUser,
      id: BigInt(1),
      createdAt,
      updatedAt,
    };
    db.user.upsert.mockResolvedValue(expectedUser);

    const user = await users.upsert(newUser);

    expect(db.user.upsert).toBeCalled();
    expect(user).toEqual(expectedUser);
    expect(db.password.upsert).toBeCalled();
  });

  it("should find user by email", async () => {
    const user = await users.findUnique(
      { email: "test@example.com" },
      "withPassword"
    );

    expect(db.user.findUnique).toBeCalled();
    expect(db.user.findUnique).toBeCalledWith({
      include: { password: true },
      where: { email: "test@example.com" },
    });
  });
});
