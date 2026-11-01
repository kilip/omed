import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAgentPermissions } from "../hooks/use-agent-permissions";
import { useProviders } from "../queries";
import type { Route } from "./+types/providers";
import AgentProvidersPage from "./providers";

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn().mockResolvedValue(10),
  getStoredWorkspaceId: vi.fn().mockReturnValue(10),
}));

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: "/agent/providers" }),
  };
});

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => key,
      i18n: { language: "en" },
    }),
  };
});

vi.mock("../queries", () => ({
  useProviders: vi.fn(),
  useCreateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSetDefaultProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useTestProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("../hooks/use-agent-permissions", () => ({
  useAgentPermissions: vi.fn(),
}));

describe("AgentProvidersPage route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders provider table with default marker and key hint", () => {
    vi.mocked(useAgentPermissions).mockReturnValue({
      canManage: true,
      canRead: true,
      isAdmin: true,
      isViewer: true,
      isOwner: true,
    });

    vi.mocked(useProviders).mockReturnValue({
      data: [
        {
          id: 1,
          name: "Production Anthropic",
          kind: "anthropic",
          baseUrl: "https://api.anthropic.com",
          model: "claude-3-7-sonnet",
          capabilities: {
            toolCalling: true,
            vision: true,
            structuredOutput: true,
            streaming: true,
          },
          hasKey: true,
          keyHint: "1234",
          isDefault: true,
          createdAt: "2026-03-01T00:00:00Z",
          updatedAt: "2026-03-01T00:00:00Z",
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    const html = renderToString(
      <AgentProvidersPage
        loaderData={{ wid: 10 }}
        params={{}}
        matches={emptyMatches}
      />,
    );

    expect(html).toContain("Production Anthropic");
    expect(html).toContain("claude-3-7-sonnet");
    expect(html).toContain("1234");
    expect(html).toContain("agent.providers.defaultBadge");
  });

  it("viewer sees no add, edit, delete, test, or set-default controls", () => {
    vi.mocked(useAgentPermissions).mockReturnValue({
      canManage: false,
      canRead: true,
      isAdmin: false,
      isViewer: true,
      isOwner: false,
    });

    vi.mocked(useProviders).mockReturnValue({
      data: [
        {
          id: 2,
          name: "OpenAI GPT-4",
          kind: "openai_compat",
          baseUrl: "https://api.openai.com/v1",
          model: "gpt-4o",
          capabilities: {
            toolCalling: true,
            vision: true,
            structuredOutput: true,
            streaming: true,
          },
          hasKey: true,
          keyHint: "5678",
          isDefault: false,
          createdAt: "2026-03-01T00:00:00Z",
          updatedAt: "2026-03-01T00:00:00Z",
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    const html = renderToString(
      <AgentProvidersPage
        loaderData={{ wid: 10 }}
        params={{}}
        matches={emptyMatches}
      />,
    );

    expect(html).toContain("OpenAI GPT-4");
    expect(html).not.toContain("agent.providers.actions.add");
    expect(html).not.toContain("agent.providers.actions.edit");
    expect(html).not.toContain("agent.providers.actions.delete");
    expect(html).not.toContain("agent.providers.actions.setDefault");
    expect(html).not.toContain("agent.providers.actions.test");
  });

  it("manager sees add and edit actions", () => {
    vi.mocked(useAgentPermissions).mockReturnValue({
      canManage: true,
      canRead: true,
      isAdmin: false,
      isViewer: false,
      isOwner: false,
    });

    vi.mocked(useProviders).mockReturnValue({
      data: [
        {
          id: 5,
          name: "Test Gemini",
          kind: "gemini",
          model: "gemini-2.0-flash",
          capabilities: { streaming: true },
          isDefault: false,
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    const html = renderToString(
      <AgentProvidersPage
        loaderData={{ wid: 10 }}
        params={{}}
        matches={emptyMatches}
      />,
    );

    expect(html).toContain("agent.providers.actions.add");
    expect(html).toContain("agent.providers.actions.edit");
  });
});
