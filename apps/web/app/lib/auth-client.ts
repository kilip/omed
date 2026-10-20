import { env } from "./env";

export interface User {
  id: number;
  name: string;
  email: string;
  avatar?: string;
  image?: string;
  locale?: string;
  currency?: string;
  emailVerified?: boolean;
}

export interface Session {
  id: number;
  token: string;
  userId: number;
  expiresAt: string;
}

export interface SessionData {
  user: User;
  session: Session;
}

export const authClient = {
  async getSession(): Promise<{
    data: SessionData | null;
    error: Error | null;
  }> {
    try {
      const res = await fetch(`${env.apiUrl}/auth/get-session`, {
        credentials: "include",
      });
      if (!res.ok) return { data: null, error: null };
      const body = await res.json();
      if (!body?.user) return { data: null, error: null };
      const user: User = {
        ...body.user,
        image: body.user.avatar || body.user.image,
      };
      return { data: { user, session: body.session }, error: null };
    } catch (err) {
      return {
        data: null,
        error: err instanceof Error ? err : new Error(String(err)),
      };
    }
  },

  signIn: {
    async social({
      provider,
      callbackURL,
      locale,
    }: {
      provider: "google" | "github";
      callbackURL: string;
      locale?: string;
    }): Promise<{ error?: { message: string } }> {
      try {
        const res = await fetch(`${env.apiUrl}/auth/sign-in/social`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            provider,
            callbackURL,
            callbackUrl: callbackURL,
            locale,
          }),
        });
        if (!res.ok) {
          const body = await res.json().catch(() => null);
          return {
            error: { message: body?.error?.message || "Social login failed" },
          };
        }
        const data = await res.json();
        if (data?.url) {
          window.location.href = data.url;
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

  async updateUser(data: {
    name?: string;
    avatar?: string;
    locale?: string;
    currency?: string;
  }): Promise<{ data?: User; error?: Error }> {
    try {
      const res = await fetch(`${env.apiUrl}/users/me`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        return {
          error: new Error(body?.error?.message || "Failed to update profile"),
        };
      }
      const body = await res.json();
      return { data: body.data };
    } catch (err) {
      return {
        error: err instanceof Error ? err : new Error(String(err)),
      };
    }
  },

  async signOut(): Promise<{ error?: Error }> {
    try {
      await fetch(`${env.apiUrl}/auth/sign-out`, {
        method: "POST",
        credentials: "include",
      });
      return {};
    } catch (err) {
      return { error: err instanceof Error ? err : new Error(String(err)) };
    }
  },
};
