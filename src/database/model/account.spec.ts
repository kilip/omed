import { beforeEach, describe, expect, it } from "vitest";
import type { Organization, User } from "@/auth";
import { ensureTestOrg, ensureTestUser } from "../../../tests/auth";
import { getTestDB } from "../core/getTestDB";
import { accounts } from "../schema";
import { AccountModel } from "./account";

const db = await getTestDB();

describe("AccountModel", () => {
  let user: User;
  let org: Organization;
  let model: AccountModel;

  beforeEach(async () => {
    await db.delete(accounts);

    user = await ensureTestUser();
    org = await ensureTestOrg();

    model = new AccountModel(db, user.id, org.id);
  });

  it("should create new account", async () => {
    const created = await model.create({
      code: "5100",
      name: "Transportation",
      type: "expense",
    });
    expect(created).toBeDefined();
    expect(created.createdBy).toBe(user.id);
    expect(created.tenantId).toBe(org.id);
  });

  it("should update account", async () => {
    const account = await model.create({
      code: "5100",
      name: "Transportation",
      type: "expense",
    });

    account.name = "Transportation Updated";
    const updated = await model.update(account);
    expect(updated.name).toBe("Transportation Updated");
  });

  it("should delete account", async () => {
    const account = await model.create({
      code: "5100",
      name: "Transportation",
      type: "expense",
    });

    await model.delete(account.id);
    const exists = await model.findById(account.id);
    expect(exists).toBeUndefined();
  });

  it("should list account for tenants", async () => {
    await model.create({
      code: "5100",
      name: "Transportation",
      type: "expense",
    });

    const rows = await model.list();
    expect(rows.length).toBe(1);
  });
});
