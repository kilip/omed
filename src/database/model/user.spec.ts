import { beforeEach, describe, expect, it } from "vitest";
import type { User } from "@/auth";
import { ensureTestUser } from "../../../tests/auth";
import { getTestDB } from "../core/getTestDB";
import { organizations, users } from "../schema";
import { UserModel } from "./user";

const db = await getTestDB();
const model = new UserModel(db);

describe("UserModel", () => {
  let user: User;

  beforeEach(async () => {
    await db.delete(organizations);
    await db.delete(users);
    user = await ensureTestUser();
  });

  it("should create user personal organization", async () => {
    const org = await model.initPersonalOrganization(user);
    expect(org).toBeDefined();
    expect(org.isPersonal).toBeTruthy();
  });

  it("should find user personal organization", async () => {
    await model.initPersonalOrganization(user);
    const org = await model.findPersonalOrganization(user.id);
    expect(org).toBeDefined();
  });

  it("should create user personal team", async () => {
    const team = await model.initPersonalTeam(user);
    expect(team).toBeDefined();
    expect(team.isPersonal).toBeTruthy();
  });

  it("should find user personal team", async () => {
    let team = await model.findPersonalTeam(user.id);
    expect(team).toBeUndefined();

    await model.initPersonalTeam(user);
    team = await model.findPersonalTeam(user.id);
    expect(team).toBeDefined();
  });

  it("should create user personal workspace", async () => {
    const ws = await model.initPersonalWorkspace(user);
    expect(ws).toBeDefined();
    expect(ws?.isPersonal).toBeTruthy();
  });

  it("should find user personal workspace", async () => {
    let ws = await model.findPersonalWorkspace(user.id);
    expect(ws).toBeUndefined();

    await model.initPersonalWorkspace(user);
    ws = await model.findPersonalWorkspace(user.id);
    expect(ws).toBeDefined();
  });
});
