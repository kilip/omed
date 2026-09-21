import { type Organization, testUtils } from "better-auth/plugins";
import type { User } from "@/auth";
import { getTestDB } from "@/database/core/getTestDB";
import { defineConfig } from "@/libs/better-auth/defineConfig";

const db = await getTestDB();
export const auth = defineConfig({
  plugins: [],
  db,
});

export const authContext = await auth.$context;
export const authTestUtils = authContext.test;

export async function getTestUser(): Promise<{
  user: User;
  org: Organization;
}> {
  const data = authTestUtils.createUser({
    name: "Test User",
    email: "test@example.com",
    emailVerified: true,
  });
  const savedUser = await authTestUtils.saveUser(data);

  const createOrganization = authTestUtils.createOrganization;
  if (!createOrganization) {
    throw new Error("createOrganization is not available in auth test utils");
  }

  const orgData = createOrganization({
    name: "Test Organization",
    slug: "test-user-org",
    owner: savedUser,
  });

  const saveOrganization = authTestUtils.saveOrganization;
  if (!saveOrganization) {
    throw new Error("saveOrganization is not available in auth test utils");
  }

  const savedOrg = (await saveOrganization(orgData)) as unknown as Organization;
  if (authTestUtils.addMember) {
    await authTestUtils.addMember({
      userId: savedUser.id,
      organizationId: savedOrg.id,
      role: "owner",
    });
  }

  return { user: savedUser, org: savedOrg };
}
