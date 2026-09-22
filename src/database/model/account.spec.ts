import { beforeEach, describe, expect, it } from "vitest";
import type { User } from "@/auth";
import { appEnv } from "@/config";
import { ensureTestUser } from "../../../tests/auth";
import { getTestDB } from "../core/getTestDB";
import { accounts } from "../schema";
import { AccountModel } from "./account";

const db = await getTestDB();

describe("AccountModel", () => {
  let user: User;
  let model: AccountModel;

  beforeEach(async () => {
    await db.delete(accounts);

    user = await ensureTestUser();
    model = new AccountModel(db, user.id, user.activeWorkspace);
  });

  it("should create new account", async () => {
    expect(user.activeWorkspace).not.toBeNull();
    expect(appEnv.TESTING).toBeTruthy();
    const acc = await model.create({
      code: "1000",
      name: "Transportation",
      type: "expense",
    });

    expect(acc).toBeDefined();
  });
});
