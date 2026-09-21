import type { Organization } from "better-auth/plugins";
import { afterAll, beforeEach, describe, expect, it } from "vitest";
import type { User } from "@/auth";
import { getTestUser } from "../../../tests/user";
import { getTestDB } from "../core/getTestDB";
import { accounts, organization, user } from "../schema";
import { AccountModel } from "./account";

const db = await getTestDB();

afterAll(async () => {
  await db.delete(accounts);
  await db.delete(organization);
  await db.delete(user);
});

describe("AccountModel", () => {
  let model: AccountModel;
  let testUser: User;
  let org: Organization;

  beforeEach(async () => {
    const { user: tUser, org: tOrg } = await getTestUser();

    testUser = tUser;
    org = tOrg;
    model = new AccountModel(db, testUser.id, org.id);
  });

  it("should create new account", async () => {
    const acc = await model.create({ name: "test", type: "asset" });

    expect(acc).toBeDefined();
    expect(acc.tenantId).toBe(org.id);
    expect(acc.createdBy).toBe(testUser.id);
  });
});
