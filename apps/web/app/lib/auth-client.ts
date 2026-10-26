import type { components } from "@omed/openapi";
import { api } from "./api";

export type User = components["schemas"]["model.UserData"];
export type Session = components["schemas"]["model.SessionData"];
export type SessionResponse = components["schemas"]["model.SessionResponse"];
export type SignInSocialRequest =
  components["schemas"]["model.SignInSocialRequest"];
export type SignInSocialResponse =
  components["schemas"]["model.SignInSocialResponse"];
export type UpdateUserRequest =
  components["schemas"]["model.UpdateUserRequest"];
export type UserResponse = components["schemas"]["model.UserResponse"];

export interface SessionData {
  user: User;
  session?: Session;
}

export const authClient = {
  async getSession(): Promise<{
    data: SessionData | null;
    error: Error | null;
  }> {
    try {
      const { data, error, response } = await api.GET("/auth/get-session");
      if (response.status === 401) {
        return { data: null, error: null };
      }
      if (error || !response.ok || !data?.user) {
        if (!response.ok) {
          const errBody = error as { error?: { message?: string } } | undefined;
          return {
            data: null,
            error: new Error(
              errBody?.error?.message ||
                response.statusText ||
                "Failed to get session",
            ),
          };
        }
        return { data: null, error: null };
      }
      return {
        data: {
          user: data.user,
          session: data.session,
        },
        error: null,
      };
    } catch (err) {
      return {
        data: null,
        error: err instanceof Error ? err : new Error(String(err)),
      };
    }
  },

  signIn: {
    async social(
      body: SignInSocialRequest,
    ): Promise<{ error?: { message: string } }> {
      try {
        const { data, error, response } = await api.POST(
          "/auth/sign-in/social",
          { body },
        );
        if (error || !response.ok) {
          const errBody = error as { error?: { message?: string } } | undefined;
          return {
            error: {
              message:
                errBody?.error?.message ||
                response.statusText ||
                "Social login failed",
            },
          };
        }
        if (data?.url) {
          if (typeof window !== "undefined") {
            window.location.href = data.url;
          }
          return {};
        }
        return { error: { message: "No redirect URL received" } };
      } catch (err) {
        return {
          error: { message: err instanceof Error ? err.message : String(err) },
        };
      }
    },
  },

  async updateUser(
    data: UpdateUserRequest,
  ): Promise<{ data?: UserResponse; error?: Error }> {
    try {
      const {
        data: resData,
        error,
        response,
      } = await api.PUT("/users/me", {
        body: data,
      });
      if (error || !response.ok) {
        const errBody = error as { error?: { message?: string } } | undefined;
        return {
          error: new Error(
            errBody?.error?.message ||
              response.statusText ||
              "Failed to update profile",
          ),
        };
      }
      return { data: resData?.data };
    } catch (err) {
      return {
        error: err instanceof Error ? err : new Error(String(err)),
      };
    }
  },

  async signOut(): Promise<{ error?: Error }> {
    try {
      const { error, response } = await api.POST("/auth/sign-out");
      if (error || !response.ok) {
        const errBody = error as { error?: { message?: string } } | undefined;
        return {
          error: new Error(
            errBody?.error?.message ||
              response.statusText ||
              "Failed to sign out",
          ),
        };
      }
      return {};
    } catch (err) {
      return { error: err instanceof Error ? err : new Error(String(err)) };
    }
  },
};
