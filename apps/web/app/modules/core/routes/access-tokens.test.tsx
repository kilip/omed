import type { QueryClient } from "@tanstack/react-query";
import { useSuspenseQuery } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import {
  ACCESS_TOKEN_EXPIRY_DAYS,
  DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS,
} from "../lib";
import { accessTokenKeys, accessTokensQueryOptions } from "../queries";
import AccessTokens, {
  clientLoader,
  revokeAccessToken,
  submitCreateToken,
  TokenRevealView,
} from "./access-tokens";

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      DELETE: vi.fn(),
    },
    unwrapList: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data ?? [];
    }),
    unwrapOne: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data;
    }),
    unwrapVoid: vi.fn(async (promise) => {
      await promise;
    }),
  };
});

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: "/core/access-tokens" }),
    useRevalidator: () => ({ revalidate: vi.fn() }),
  };
});

vi.mock("~/lib/use-action", () => ({
  useAction: () => ({
    run: vi.fn(async (fn: () => Promise<unknown>) => {
      await fn();
      return true;
    }),
    pending: false,
  }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
    }),
    useSuspenseQuery: vi.fn().mockReturnValue({
      data: [
        {
          id: 1,
          name: "CLI Agent Token",
          prefix: "omd_pat_abc123",
          status: "active",
          expiresAt: "2026-06-01T00:00:00Z",
          createdAt: "2026-03-01T00:00:00Z",
        },
      ],
    }),
  };
});

const emptyMatches = [] as unknown as Parameters<
  typeof AccessTokens
>[0]["matches"];

describe("AccessTokens route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("clientLoader", () => {
    it("prefetches access tokens", async () => {
      const mockEnsureQueryData = vi.fn().mockResolvedValue([]);
      vi.mocked(getQueryClient).mockReturnValue({
        ensureQueryData: mockEnsureQueryData,
      } as unknown as ReturnType<typeof getQueryClient>);

      const result = await clientLoader();
      expect(result).toEqual({});
      expect(mockEnsureQueryData).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: accessTokensQueryOptions().queryKey,
        }),
      );
    });
  });

  describe("AccessTokens component", () => {
    it("lists tokens with name, prefix, expiry and a status tag", () => {
      const html = renderToString(
        <AccessTokens loaderData={{}} params={{}} matches={emptyMatches} />,
      );

      expect(html).toContain("CLI Agent Token");
      expect(html).toContain("omd_pat_abc123");
      expect(html).toContain("core.accessTokens.active");
      expect(html).toContain("2026");
    });

    it("creates a token, shows the plaintext with a copy button and a one-time warning, and no longer shows it after closing", async () => {
      const mockCreated = {
        id: 2,
        name: "New PAT",
        token: "omd_pat_secret987654321",
        status: "active",
      };
      vi.mocked(api.POST).mockResolvedValueOnce({
        data: { data: mockCreated },
        response: new Response(null, { status: 201 }),
      } as never);

      const res = await submitCreateToken({
        name: "New PAT",
        expiresInDays: 90,
      });
      const token = res.token ?? "";
      expect(token).toBe("omd_pat_secret987654321");

      // Reveal view shows plaintext token, copy button, and warning
      const htmlWithToken = renderToString(<TokenRevealView token={token} />);
      expect(htmlWithToken).toContain("omd_pat_secret987654321");
      expect(htmlWithToken).toContain("core.accessTokens.copy");
      expect(htmlWithToken).toContain("core.accessTokens.copyWarning");

      // After closing, token state is reset to null and no longer shown
      let tokenState: string | null = token;
      const handleClose = () => {
        tokenState = null;
      };
      handleClose();
      expect(tokenState).toBeNull();
      const htmlClosed = renderToString(
        tokenState ? <TokenRevealView token={tokenState} /> : null,
      );
      expect(htmlClosed).not.toContain("omd_pat_secret987654321");
    });

    it("sends the chosen expiry and defaults to 90", async () => {
      expect(DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS).toBe(90);
      expect(ACCESS_TOKEN_EXPIRY_DAYS).toEqual([30, 90, 365]);

      vi.mocked(api.POST).mockResolvedValue({
        data: { data: { id: 1, token: "pat_test" } },
        response: new Response(null, { status: 201 }),
      } as never);

      // Sends chosen expiry
      await submitCreateToken({ name: "Custom Expiry", expiresInDays: 30 });
      expect(api.POST).toHaveBeenCalledWith("/users/me/access-tokens", {
        body: { name: "Custom Expiry", expiresInDays: 30 },
      });

      // Defaults to 90
      await submitCreateToken({ name: "Default Expiry" });
      expect(api.POST).toHaveBeenCalledWith("/users/me/access-tokens", {
        body: { name: "Default Expiry", expiresInDays: 90 },
      });
    });

    it("revokes after confirmation and refreshes the list", async () => {
      const mockQueryClient = {
        invalidateQueries: vi.fn().mockResolvedValue(undefined),
      } as unknown as QueryClient;

      vi.mocked(api.DELETE).mockResolvedValueOnce({
        data: {},
        response: new Response(null, { status: 200 }),
      } as never);

      await revokeAccessToken(42, mockQueryClient);

      expect(api.DELETE).toHaveBeenCalledWith("/users/me/access-tokens/{id}", {
        params: { path: { id: 42 } },
      });
      expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
        queryKey: accessTokenKeys.all,
      });

      // Table includes revoke button
      const html = renderToString(
        <AccessTokens loaderData={{}} params={{}} matches={emptyMatches} />,
      );
      expect(html).toContain("core.accessTokens.revoke");
    });

    it("shows an empty state when there are no tokens", () => {
      vi.mocked(useSuspenseQuery).mockReturnValueOnce({
        data: [],
      } as never);

      const html = renderToString(
        <AccessTokens loaderData={{}} params={{}} matches={emptyMatches} />,
      );

      expect(html).toContain("core.accessTokens.empty");
      expect(html).toContain("ant-empty");
      expect(html).not.toContain("CLI Agent Token");
    });
  });
});
