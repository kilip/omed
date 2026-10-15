import { config } from "./config";

export interface TestUser {
  id: number;
  name: string;
  email: string;
  emailVerified: boolean;
  image?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TestSession {
  id: number;
  userId: number;
  token: string;
  expiresAt: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TestWorkspace {
  id: number;
  name: string;
  slug: string;
  description?: string;
  ownerId: number;
}

export interface PlaywrightCookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  httpOnly?: boolean;
  sameSite?: "Strict" | "Lax" | "None";
}

export interface TestEmailAddress {
  Name?: string;
  Address?: string;
}

export interface TestEmailMessage {
  From?: TestEmailAddress;
  To?: TestEmailAddress[];
  Cc?: TestEmailAddress[];
  Bcc?: TestEmailAddress[];
  Subject: string;
  TextBody: string;
  HTMLBody: string;
  Headers?: Record<string, string>;
}

export class TestClient {
  private baseUrl: string;

  constructor(baseUrl: string = config.apiUrl) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
  }

  private async request<T>(
    path: string,
    options: RequestInit = {},
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers = new Headers(options.headers);
    if (!headers.has("Content-Type") && options.body) {
      headers.set("Content-Type", "application/json");
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(
        `TestClient request failed [${response.status} ${response.statusText}] for ${url}: ${text}`,
      );
    }

    if (response.status === 204) {
      return undefined as unknown as T;
    }

    return (await response.json()) as T;
  }

  async createUser(data: {
    id?: number;
    name: string;
    email: string;
    emailVerified?: boolean;
    image?: string;
  }): Promise<TestUser> {
    return this.request<TestUser>("/test/users", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async createSession(data: {
    userId: number;
    token?: string;
    expiresAt?: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<TestSession> {
    return this.request<TestSession>("/test/sessions", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async createWorkspace(data: {
    name: string;
    slug: string;
    description?: string;
    ownerId: number;
  }): Promise<TestWorkspace> {
    return this.request<TestWorkspace>("/test/workspaces", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async addMember(data: {
    workspaceId: number;
    userId: number;
    role: string;
  }): Promise<void> {
    await this.request<void>("/test/members", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async deleteUser(userId: number): Promise<void> {
    await this.request<void>(`/test/users/${userId}`, {
      method: "DELETE",
    });
  }

  async reset(): Promise<void> {
    await this.request<void>("/test/reset", {
      method: "POST",
    });
  }

  async getSentEmails(): Promise<TestEmailMessage[]> {
    return this.request<TestEmailMessage[]>("/test/mailer/messages");
  }

  async getLastEmail(): Promise<TestEmailMessage> {
    return this.request<TestEmailMessage>("/test/mailer/last");
  }

  async resetEmails(): Promise<void> {
    await this.request<void>("/test/mailer/reset", {
      method: "POST",
    });
  }

  async setupAuthenticatedUser(options?: {
    name?: string;
    email?: string;
  }): Promise<{
    user: TestUser;
    session: TestSession;
    cookies: PlaywrightCookie[];
  }> {
    const user = await this.createUser({
      name: options?.name ?? "E2E Tester",
      email: options?.email ?? `e2e+${crypto.randomUUID()}@omed.test`,
      emailVerified: true,
    });

    const session = await this.createSession({
      userId: user.id,
    });

    const cookies: PlaywrightCookie[] = [
      {
        name: "omed_session",
        value: session.token,
        domain: "localhost",
        path: "/",
        httpOnly: true,
        sameSite: "Lax",
      },
    ];

    return { user, session, cookies };
  }

  async flushCoverage(): Promise<void> {
    try {
      await this.request<{ success: boolean }>("/test/coverage/flush", {
        method: "POST",
      });
    } catch {
      // Best-effort: server may not be up or test routes disabled.
    }
  }
}

export const testClient = new TestClient();
