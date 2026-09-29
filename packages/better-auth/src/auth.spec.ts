import { describe, expect, it } from "vitest";
import { auth } from "./auth";

const context = await auth.$context;
const util = context.test;

describe("Auth", () => {
  it("should create default workspace for user", async () => {
    const user = util.createUser({
      name: "testing",
    });
    const created = await util.saveUser(user);
    expect(created).toBeDefined();
  });
});
