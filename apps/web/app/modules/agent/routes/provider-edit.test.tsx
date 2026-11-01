import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { requireWorkspaceId } from "~/lib/workspace";
import { type ProviderResponse, useProvider } from "../queries";
import type { Route } from "./+types/provider-edit";
import ProviderEditRoute, { clientLoader } from "./provider-edit";

const emptyMatches = [] as unknown as Route.ComponentProps["matches"];

const mockNavigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: "/agent/providers/42/edit" }),
    useParams: () => ({ id: "42" }),
  };
});

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn(),
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

let lastProviderFormProps: {
  provider?: ProviderResponse | null;
  wid?: number;
  onSuccess?: () => void;
  onCancel?: () => void;
} | null = null;

vi.mock("../components/provider-form", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../components/provider-form")>();
  return {
    ...actual,
    ProviderForm: (props: typeof lastProviderFormProps) => {
      lastProviderFormProps = props;
      return actual.ProviderForm(props as never);
    },
  };
});

vi.mock("../queries", () => ({
  useProvider: vi.fn(),
  useCreateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useProviderModels: () => ({ data: [], isFetching: false, isError: false }),
  useProbeProviderModels: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

describe("ProviderEditRoute", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    lastProviderFormProps = null;
  });

  describe("clientLoader", () => {
    it("returns resolved workspace id and parsed provider id from params", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(15);
      const res = await clientLoader({
        params: { id: "42" },
      } as unknown as Route.ClientLoaderArgs);
      expect(res).toEqual({ wid: 15, id: 42 });
    });
  });

  describe("component", () => {
    it("renders loading state when provider is loading", () => {
      vi.mocked(useProvider).mockReturnValue({
        data: undefined,
        isLoading: true,
      } as unknown as ReturnType<typeof useProvider>);

      const html = renderToString(
        <ProviderEditRoute
          loaderData={{ wid: 15, id: 42 }}
          params={{ id: "42" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("agent.providers.routes.editTitle");
      expect(html).toContain("ant-spin");
    });

    it("renders not found state when provider data is missing", () => {
      vi.mocked(useProvider).mockReturnValue({
        data: undefined,
        isLoading: false,
      } as unknown as ReturnType<typeof useProvider>);

      const html = renderToString(
        <ProviderEditRoute
          loaderData={{ wid: 15, id: 42 }}
          params={{ id: "42" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("agent.providers.routes.editTitle");
      expect(html).toContain("agent.providers.errors.notFound");
      expect(html).toContain("common.cancel");
    });

    it("renders edit form when provider is loaded", () => {
      const mockProvider: ProviderResponse = {
        id: 42,
        name: "OpenAI GPT-4o",
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
        keyHint: "sk-proj",
        isDefault: true,
        createdAt: "2026-03-01T00:00:00Z",
        updatedAt: "2026-03-01T00:00:00Z",
      };

      vi.mocked(useProvider).mockReturnValue({
        data: mockProvider,
        isLoading: false,
      } as unknown as ReturnType<typeof useProvider>);

      const html = renderToString(
        <ProviderEditRoute
          loaderData={{ wid: 15, id: 42 }}
          params={{ id: "42" }}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("agent.providers.routes.editTitle");
      expect(html).toContain("agent.providers.title");
      expect(html).toContain("OpenAI GPT-4o");
      expect(html).toContain("gpt-4o");
      expect(html).toContain("common.save");
      expect(html).toContain("common.cancel");
    });

    it("wires onSuccess and onCancel to navigate to /agent/providers", () => {
      const mockProvider: ProviderResponse = {
        id: 42,
        name: "OpenAI GPT-4o",
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
        keyHint: "sk-proj",
        isDefault: true,
        createdAt: "2026-03-01T00:00:00Z",
        updatedAt: "2026-03-01T00:00:00Z",
      };

      vi.mocked(useProvider).mockReturnValue({
        data: mockProvider,
        isLoading: false,
      } as unknown as ReturnType<typeof useProvider>);

      renderToString(
        <ProviderEditRoute
          loaderData={{ wid: 15, id: 42 }}
          params={{ id: "42" }}
          matches={emptyMatches}
        />,
      );

      expect(lastProviderFormProps).not.toBeNull();
      expect(lastProviderFormProps?.wid).toBe(15);
      expect(lastProviderFormProps?.provider).toEqual(mockProvider);

      lastProviderFormProps?.onSuccess?.();
      expect(mockNavigate).toHaveBeenCalledWith("/agent/providers");

      mockNavigate.mockClear();
      lastProviderFormProps?.onCancel?.();
      expect(mockNavigate).toHaveBeenCalledWith("/agent/providers");
    });
  });
});
