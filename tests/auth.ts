import { auth, type Organization } from "@/auth";
import { serverDB } from "@/database";

export const authCtx = await auth.$context;
export const authTest = authCtx.test;

export async function ensureTestUser() {
  const user = authTest.createUser({
    name: "Test User",
    email: "test@example.com",
    emailVerified: true,
  });

  const row = await serverDB.query.user.findFirst({
    where: { email: user.email },
  });
  if (row) return row;
  return authTest.saveUser(user);
}

export async function ensureTestOrg(): Promise<Organization> {
  let created: Organization = {} as Organization;

  const exists = await serverDB.query.organization.findFirst({
    where: { slug: "test-org" },
  });

  if (exists) return exists;

  if (authTest.createOrganization && authTest.saveOrganization) {
    const org = authTest.createOrganization({
      name: "Test Organization",
      slug: "test-org",
    });

    created = (await authTest.saveOrganization(org)) as Organization;
  }
  return created;
}
