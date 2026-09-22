import { beforeEach, describe, expect, it } from "vitest";
import type { User } from "@/auth";
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
    model = new AccountModel(db);
  });

  it("should create new account", async () => {
    expect(user.activeWorkspace).not.toBeNull();

    const acc = await model.create({
      code: "1000",
      name: "Transportation",
      type: "expense",
      createdBy: user.id,
      tenantId: user.activeWorkspace,
    });

    expect(acc).toBeDefined();
  });
});
