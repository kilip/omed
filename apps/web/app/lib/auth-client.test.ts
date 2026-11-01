import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import { authClient } from "./auth-client";

vi.mock("./api", () => ({
  api: {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  },
}));

type ApiGetResult = Awaited<ReturnType<typeof api.GET>>;
type ApiPostResult = Awaited<ReturnType<typeof api.POST>>;
type ApiPutResult = Awaited<ReturnType<typeof api.PUT>>;

describe("authClient", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getSession", () => {
    it("returns user and session on successful response", async () => {
      const mockUser = {
        id: 1,
        name: "Test User",
        email: "test@example.com",
        image: "https://example.com/avatar.png",
      };
      const mockSession = {
        id: 10,
        userId: 1,
        expiresAt: "2026-12-31T00:00:00Z",
      };

      vi.mocked(api.GET).mockResolvedValueOnce({
        data: { user: mockUser, session: mockSession },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      const result = await authClient.getSession();

      expect(api.GET).toHaveBeenCalledWith("/auth/get-session");
      expect(result).toEqual({
        data: { user: mockUser, session: mockSession },
        error: null,
      });
    });

    it("returns null data and null error on 401 Unauthorized", async () => {
      vi.mocked(api.GET).mockResolvedValueOnce({
        error: { error: { message: "Unauthorized" } },
        response: new Response(null, { status: 401 }),
      } as unknown as ApiGetResult);

      const result = await authClient.getSession();

      expect(api.GET).toHaveBeenCalledWith("/auth/get-session");
      expect(result).toEqual({
        data: null,
        error: null,
      });
    });

    it("returns null data and null error when user is absent", async () => {
      vi.mocked(api.GET).mockResolvedValueOnce({
        data: {},
        response: new Response(null, { status: 200 }),
      } as unknown as ApiGetResult);

      const result = await authClient.getSession();

      expect(result).toEqual({
        data: null,
        error: null,
      });
    });

    it("returns error on server failure", async () => {
      vi.mocked(api.GET).mockResolvedValueOnce({
        error: { error: { message: "Server error" } },
        response: new Response(null, {
          status: 500,
          statusText: "Internal Server Error",
        }),
      } as unknown as ApiGetResult);

      const result = await authClient.getSession();

      expect(result.data).toBeNull();
      expect(result.error).toBeInstanceOf(Error);
      expect(result.error?.message).toBe("Server error");
    });

    it("catches network exceptions", async () => {
      vi.mocked(api.GET).mockRejectedValueOnce(new Error("Network failure"));

      const result = await authClient.getSession();

      expect(result).toEqual({
        data: null,
        error: expect.any(Error),
      });
      expect(result.error?.message).toBe("Network failure");
    });
  });

  describe("signIn.social", () => {
    it("redirects window to url on success", async () => {
      const originalWindow = globalThis.window;
      // @ts-expect-error mock window
      globalThis.window = {
        location: { href: "" },
      } as unknown as Window & typeof globalThis;

      try {
        vi.mocked(api.POST).mockResolvedValueOnce({
          data: { url: "https://auth.google.com/oauth" },
          response: new Response(null, { status: 200 }),
        } as unknown as ApiPostResult);

        const result = await authClient.signIn.social({
          provider: "google",
          callbackURL: "http://localhost:5173",
          locale: "en",
        });

        expect(api.POST).toHaveBeenCalledWith("/auth/sign-in/social", {
          body: {
            provider: "google",
            callbackURL: "http://localhost:5173",
            locale: "en",
          },
        });
        expect(globalThis.window.location.href).toBe(
          "https://auth.google.com/oauth",
        );
        expect(result).toEqual({});
      } finally {
        globalThis.window = originalWindow;
      }
    });

    it("returns error when redirect URL is missing", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        data: {},
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPostResult);

      const result = await authClient.signIn.social({
        provider: "github",
        callbackURL: "http://localhost:5173",
      });

      expect(result).toEqual({
        error: { message: "No redirect URL received" },
      });
    });

    it("returns error on failure", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        error: { error: { message: "Provider not configured" } },
        response: new Response(null, { status: 400 }),
      } as unknown as ApiPostResult);

      const result = await authClient.signIn.social({
        provider: "google",
        callbackURL: "http://localhost:5173",
      });

      expect(result).toEqual({
        error: { message: "Provider not configured" },
      });
    });
  });

  describe("updateUser", () => {
    it("updates user via PUT /users/me", async () => {
      const updatedUser = {
        id: 1,
        name: "New Name",
        email: "test@example.com",
        locale: "id",
      };

      vi.mocked(api.PUT).mockResolvedValueOnce({
        data: { data: updatedUser },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPutResult);

      const result = await authClient.updateUser({
        name: "New Name",
        locale: "id",
      });

      expect(api.PUT).toHaveBeenCalledWith("/users/me", {
        body: {
          name: "New Name",
          locale: "id",
        },
      });
      expect(result).toEqual({ data: updatedUser });
    });

    it("returns error when update fails", async () => {
      vi.mocked(api.PUT).mockResolvedValueOnce({
        error: { error: { message: "Invalid locale" } },
        response: new Response(null, { status: 400 }),
      } as unknown as ApiPutResult);

      const result = await authClient.updateUser({
        locale: "id",
      });

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error?.message).toBe("Invalid locale");
    });
  });

  describe("signOut", () => {
    it("calls POST /auth/sign-out", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { success: true },
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPostResult);

      const result = await authClient.signOut();

      expect(api.POST).toHaveBeenCalledWith("/auth/sign-out");
      expect(result).toEqual({});
    });

    it("returns error when sign-out fails", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        error: { error: { message: "Failed to clear session" } },
        response: new Response(null, { status: 500 }),
      } as unknown as ApiPostResult);

      const result = await authClient.signOut();

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error?.message).toBe("Failed to clear session");
    });
  });

  describe("otp", () => {
    it("otp.request returns data on success", async () => {
      const mockResponse = {
        codeLength: 6,
        resendAfterSeconds: 60,
        success: true,
        ttlSeconds: 600,
      };

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: mockResponse,
        response: new Response(null, { status: 202 }),
      } as unknown as ApiPostResult);

      const result = await authClient.otp.request({
        email: "user@example.com",
        locale: "en",
      });

      expect(api.POST).toHaveBeenCalledWith("/auth/otp/request", {
        body: {
          email: "user@example.com",
          locale: "en",
        },
      });
      expect(result).toEqual({ data: mockResponse });
    });

    it("otp.request surfaces AUTH_OTP_UNAVAILABLE code and message", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        error: {
          error: {
            code: "AUTH_OTP_UNAVAILABLE",
            message: "email sign-in is not available",
          },
        },
        response: new Response(null, { status: 503 }),
      } as unknown as ApiPostResult);

      const result = await authClient.otp.request({
        email: "user@example.com",
      });

      expect(result).toEqual({
        error: {
          code: "AUTH_OTP_UNAVAILABLE",
          message: "email sign-in is not available",
        },
      });
    });

    it("otp.verify returns session data", async () => {
      const mockSession = {
        session: {
          id: 10,
          userId: 1,
          expiresAt: "2026-12-31T00:00:00Z",
        },
        user: {
          id: 1,
          name: "Test User",
          email: "user@example.com",
        },
      };

      vi.mocked(api.POST).mockResolvedValueOnce({
        data: mockSession,
        response: new Response(null, { status: 200 }),
      } as unknown as ApiPostResult);

      const result = await authClient.otp.verify({
        email: "user@example.com",
        code: "123456",
      });

      expect(api.POST).toHaveBeenCalledWith("/auth/otp/verify", {
        body: {
          email: "user@example.com",
          code: "123456",
        },
      });
      expect(result).toEqual({ data: mockSession });
    });

    it("otp.verify surfaces AUTH_OTP_INVALID code", async () => {
      vi.mocked(api.POST).mockResolvedValueOnce({
        error: {
          error: {
            code: "AUTH_OTP_INVALID",
            message: "invalid or expired code",
          },
        },
        response: new Response(null, { status: 400 }),
      } as unknown as ApiPostResult);

      const result = await authClient.otp.verify({
        email: "user@example.com",
        code: "000000",
      });

      expect(result).toEqual({
        error: {
          code: "AUTH_OTP_INVALID",
          message: "invalid or expired code",
        },
      });
    });

    it("otp.verify converts a thrown network error", async () => {
      vi.mocked(api.POST).mockRejectedValueOnce(new Error("Network failure"));

      const result = await authClient.otp.verify({
        email: "user@example.com",
        code: "123456",
      });

      expect(result).toEqual({
        error: { message: "Network failure" },
      });
    });
  });
});
