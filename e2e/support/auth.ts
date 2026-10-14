import { type TestSession, type TestUser, testClient } from "./test-client";

export interface LegacyTestHelpers {
  createUser: (data: { name: string; email: string }) => Promise<TestUser>;
  saveUser: (userPromise: Promise<TestUser> | TestUser) => Promise<TestUser>;
  login: (options: { userId: number }) => Promise<{
    session: TestSession;
    cookies: Array<{
      name: string;
      value: string;
      domain: string;
      path: string;
      httpOnly: boolean;
      sameSite: "Lax";
    }>;
  }>;
  deleteUser: (id: number) => Promise<void>;
}

/**
 * Test helpers backed by apps/api development test endpoints (/test/*).
 * Replaces Better Auth testUtils so tests don't require apps/auth.
 */
export const getTestHelpers = async (): Promise<LegacyTestHelpers> => {
  return {
    createUser: async (data: { name: string; email: string }) => {
      return testClient.createUser({
        name: data.name,
        email: data.email,
        emailVerified: true,
      });
    },
    saveUser: async (userPromise: Promise<TestUser> | TestUser) => {
      return userPromise;
    },
    login: async ({ userId }: { userId: number }) => {
      const session = await testClient.createSession({ userId });
      return {
        session,
        cookies: [
          {
            name: "token",
            value: session.token,
            domain: "localhost",
            path: "/",
            httpOnly: true,
            sameSite: "Lax",
          },
          {
            name: "better-auth.session_token",
            value: session.token,
            domain: "localhost",
            path: "/",
            httpOnly: true,
            sameSite: "Lax",
          },
        ],
      };
    },
    deleteUser: async (id: number) => {
      await testClient.deleteUser(id);
    },
  };
};

export { testClient };
