import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { requireWorkspaceId } from "~/lib/workspace";
import { useAgentPermissions } from "../hooks/use-agent-permissions";
import { useProviders, useSettings } from "../queries";
import AgentSettingsPage, {
  buildProviderOptions,
  buildUpdateSettingsPayload,
  clientLoader,
  mapSettingsError,
} from "./settings";

const mockNavigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: "/agent/settings" }),
  };
});

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn().mockResolvedValue(10),
  getStoredWorkspaceId: vi.fn().mockReturnValue(10),
}));

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

const mockMutateAsync = vi.fn();

vi.mock("../queries", () => ({
  useSettings: vi.fn(),
  useUpdateSettings: vi.fn(() => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  })),
  useProviders: vi.fn(),
  settingsQueryOptions: vi.fn((wid: number) => ({
    queryKey: ["agent", "workspaces", wid, "settings"],
  })),
  providersQueryOptions: vi.fn((wid: number) => ({
    queryKey: ["agent", "workspaces", wid, "providers"],
  })),
}));

vi.mock("../hooks/use-agent-permissions", () => ({
  useAgentPermissions: vi.fn(),
}));

describe("AgentSettingsPage", () => {
  const wid = 10;
  const mockProviders = [
    {
      id: 1,
      name: "Anthropic Claude",
      kind: "anthropic",
      model: "claude-3-7-sonnet",
      isDefault: true,
      capabilities: { vision: true, structuredOutput: true, documents: true },
    },
    {
      id: 2,
      name: "OpenAI GPT-4",
      kind: "openai_compat",
      model: "gpt-4o",
      isDefault: false,
      capabilities: { vision: true, structuredOutput: true, documents: false },
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(requireWorkspaceId).mockResolvedValue(wid);
    vi.mocked(useProviders).mockReturnValue({
      data: mockProviders,
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);
    vi.mocked(useAgentPermissions).mockReturnValue({
      canManageSettings: true,
      canUseFinancial: true,
      canManage: true,
      canRead: true,
      isAdmin: true,
      isViewer: true,
      isOwner: true,
    });
  });

  it("clientLoader resolves workspace ID via requireWorkspaceId", async () => {
    const result = await clientLoader();
    expect(requireWorkspaceId).toHaveBeenCalled();
    expect(result).toEqual({ wid: 10 });
  });

  describe("Payload building and error mapping helpers", () => {
    it("buildProviderOptions includes Provider default plus every workspace provider", () => {
      const options = buildProviderOptions(mockProviders, (k) => k);
      expect(options).toEqual([
        { value: 0, label: "agent.financial.defaultProvider" },
        {
          value: 1,
          label: "Anthropic Claude (agent.providers.defaultBadge)",
        },
        { value: 2, label: "OpenAI GPT-4" },
      ]);
    });

    it("buildUpdateSettingsPayload sends null for Provider default", () => {
      expect(
        buildUpdateSettingsPayload({
          financialEnabled: true,
          financialProviderId: null,
        }),
      ).toEqual({
        financialEnabled: true,
        financialProviderId: null,
      });

      expect(
        buildUpdateSettingsPayload({
          financialEnabled: false,
          financialProviderId: undefined,
        }),
      ).toEqual({
        financialEnabled: false,
        financialProviderId: null,
      });

      expect(
        buildUpdateSettingsPayload({
          financialEnabled: true,
          financialProviderId: 1,
        }),
      ).toEqual({
        financialEnabled: true,
        financialProviderId: 1,
      });
    });

    it("mapSettingsError maps FINANCIAL_AGENT_NO_PROVIDER to form-level error message", () => {
      const err = { code: "FINANCIAL_AGENT_NO_PROVIDER", status: 422 };
      const mapped = mapSettingsError(err, (k) => k);
      expect(mapped.code).toBe("FINANCIAL_AGENT_NO_PROVIDER");
      expect(mapped.message).toBe("agent.financial.errors.noProvider");
    });

    it("mapSettingsError handles generic errors", () => {
      const err = new Error("Network error");
      const mapped = mapSettingsError(err, (k) => k);
      expect(mapped.message).toBe("Network error");
    });
  });

  describe("Rendering", () => {
    it("renders Switch reflecting financialEnabled true", () => {
      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: true, financialProviderId: 1 },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const html = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(html).toContain("ant-switch-checked");
      expect(html).toContain("agent.financial.enabled");
    });

    it("renders Switch reflecting financialEnabled false", () => {
      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: false, financialProviderId: null },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const html = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(html).not.toContain("ant-switch-checked");
    });

    it("renders Select with Provider default plus every workspace provider", () => {
      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: true, financialProviderId: null },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const html = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(html).toContain("agent.financial.defaultProvider");

      // When a provider is selected
      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: true, financialProviderId: 1 },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const htmlSelected = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(htmlSelected).toContain("Anthropic Claude");
    });

    it("displays privacy notice that uploaded documents are sent to the configured LLM provider", () => {
      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: true, financialProviderId: 1 },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const html = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(html).toContain("agent.financial.privacyNotice");
    });

    it("user without canManageSettings sees a read-only page with disabled controls and no save button", () => {
      vi.mocked(useAgentPermissions).mockReturnValue({
        canManageSettings: false,
        canUseFinancial: false,
        canManage: false,
        canRead: true,
        isAdmin: false,
        isViewer: true,
        isOwner: false,
      });

      vi.mocked(useSettings).mockReturnValue({
        data: { financialEnabled: true, financialProviderId: 1 },
        isLoading: false,
      } as unknown as ReturnType<typeof useSettings>);

      const html = renderToString(
        <AgentSettingsPage loaderData={{ wid }} matches={[] as never} />,
      );

      expect(html).toContain("ant-switch-disabled");
      expect(html).toContain("ant-select-disabled");
      expect(html).not.toContain("agent.financial.save");
    });
  });
});
