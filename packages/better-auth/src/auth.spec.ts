import { describe, expect, it } from "vitest";
import { auth } from "./auth";
import type { Session } from "./type";

const context = await auth.$context;
const util = context.test;

describe("Auth", () => {
  it("should create default workspace for user", async () => {
    const user = util.createUser({
      name: "testing",
    });
    const created = await util.saveUser(user);
    expect(created).toBeDefined();

    const data = await util.login({
      userId: created.id,
    });

    const session = data.session as Session;
    expect(session).toBeDefined();
    expect(session?.activeTeamId).not.toBeNull();
  });
});
