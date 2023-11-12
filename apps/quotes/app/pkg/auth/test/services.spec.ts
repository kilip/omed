import db from "@omed/db/src/__mock__/db";
import { describe, expect, it, vi } from "vitest";
import { login } from "..";

vi.mock("@omed/db", async (importActual) => {
  const imports: typeof importActual = await importActual();
  const mock = (await import("@omed/db/src/__mock__/db")).default;
  return {
    ...imports,
    db: mock,
  };
});

describe("services", () => {
  it("login()", async () => {
    await login({ email: "foobar", password: "password" });
    expect(db.user.findUnique).toBeCalled();
  });
});
