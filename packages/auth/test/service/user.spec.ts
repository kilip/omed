import { beforeEach, describe, expect, it } from "vitest";
import { organizations, users } from "../../src/drizzle";
import { UserService } from "../../src/service";
import type { User } from "../../src/type";
import { testDB } from "../getTestDB";

const service = new UserService(testDB);

describe("UserService", () => {
  let user!: User;

  beforeEach(async () => {
    await testDB.delete(organizations);
    await testDB.delete(users);
    const [newUser] = await testDB
      .insert(users)
      .values({
        name: "Test User",
        email: "test@example.com",
        emailVerified: true,
        createdAt: new Date(),
      })
      .returning();
    user = newUser;
  });

  it("should init personal organization", async () => {
    expect(await service.findPersonalOrganization(user.id)).toBeUndefined();

    const org = await service.initPersonalOrganization(user);
    expect(org).toBeDefined();
    expect(org.isPersonal).toBeTruthy();

    const reinit = await service.initPersonalOrganization(user);
    expect(reinit.id).toBe(org.id);

    expect(await service.findPersonalOrganization(user.id)).toBeDefined();
  });

  it("should init personal team", async () => {
    expect(await service.findPersonalTeam(user.id)).toBeUndefined();

    const newTeam = await service.initPersonalTeam(user);
    expect(newTeam).toBeDefined();
    expect(newTeam.isPersonal).toBeTruthy();

    // ensure only created once
    const reinit = await service.initPersonalTeam(user);
    expect(reinit.id).toBe(newTeam.id);

    expect(await service.findPersonalTeam(user.id)).toBeDefined();
  });
});
