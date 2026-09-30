import type { BrowserContext } from "playwright";
import { e2eConfig } from "./config";

export type WorkspaceRole = "owner" | "admin" | "member";
export type GlobalRole = "superadmin" | "user";
export type AuthRole = WorkspaceRole | GlobalRole | string;

export interface TestAuthUser {
  id: string;
  email: string;
  name: string;
  role: AuthRole;
}

/**
 * Creates a test user with specified role using Better Auth testUtils,
 * retrieves session cookies, and injects them into the Playwright BrowserContext.
 */
export async function loginAsRole(
  context: BrowserContext,
  role: AuthRole = "owner",
  options?: { email?: string; name?: string },
): Promise<{ user: TestAuthUser; cleanup: () => Promise<void> }> {
  const { auth } = await import("@omed/better-auth");
  const ctx = await auth.$context;
  const testUtils = ctx.test;

  if (!testUtils) {
    throw new Error(
      "Better Auth testUtils plugin is not initialized. Ensure DEVELOPMENT mode is enabled.",
    );
  }

  const normalizedRole = role.toLowerCase().trim();
  const email = options?.email ?? `${normalizedRole}-${Date.now()}@example.com`;
  const name = options?.name ?? `Test ${role}`;
  const isSuperadmin = normalizedRole === "superadmin";

  // 1. Create and save user via testUtils
  const user = testUtils.createUser({
    email,
    name,
    emailVerified: true,
    ...(isSuperadmin ? { role: "admin" } : {}),
  });
  await testUtils.saveUser(user);

  // 2. Adjust organization member role if workspace role specified
  const adapter = ctx.adapter;
  if (["admin", "member"].includes(normalizedRole)) {
    const member = await adapter.findOne<{ id: string; role: string }>({
      model: "member",
      where: [{ field: "userId", value: user.id }],
    });
    if (member) {
      await adapter.update({
        model: "member",
        where: [{ field: "id", value: member.id }],
        update: { role: normalizedRole },
      });
    }
  }

  // 3. Generate session cookies via testUtils
  const cookies = await testUtils.getCookies({
    userId: user.id,
    domain: e2eConfig.cookieDomain,
  });

  // 4. Inject cookies into Playwright context
  await context.addCookies(cookies);

  return {
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      role,
    },
    cleanup: async () => {
      try {
        await testUtils.deleteUser(user.id);
      } catch (err) {
        // Silently catch cleanup errors to avoid masking test assertions
        console.warn(`Failed to cleanup test user ${user.id}:`, err);
      }
    },
  };
}
