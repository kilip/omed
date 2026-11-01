import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { requireWorkspaceId } from "~/lib/workspace";
import type { Route } from "./+types/provider-new";
import ProviderNewRoute, { clientLoader } from "./provider-new";

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
    useLocation: () => ({ pathname: "/agent/providers/new" }),
    useParams: () => ({}),
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
  useCreateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useProviderModels: () => ({ data: [], isFetching: false, isError: false }),
  useProbeProviderModels: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

describe("ProviderNewRoute", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    lastProviderFormProps = null;
  });

  describe("clientLoader", () => {
    it("returns resolved workspace id", async () => {
      vi.mocked(requireWorkspaceId).mockResolvedValueOnce(12);
      const res = await clientLoader();
      expect(res).toEqual({ wid: 12 });
      expect(requireWorkspaceId).toHaveBeenCalled();
    });
  });

  describe("component", () => {
    it("renders create provider page with form and breadcrumb", () => {
      const html = renderToString(
        <ProviderNewRoute
          loaderData={{ wid: 10 }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(html).toContain("agent.providers.routes.createTitle");
      expect(html).toContain("agent.providers.title");
      expect(html).toContain("agent.providers.fields.name");
      expect(html).toContain("common.save");
      expect(html).toContain("common.cancel");
    });

    it("wires onSuccess and onCancel to navigate to /agent/providers", () => {
      renderToString(
        <ProviderNewRoute
          loaderData={{ wid: 10 }}
          params={{}}
          matches={emptyMatches}
        />,
      );

      expect(lastProviderFormProps).not.toBeNull();
      expect(lastProviderFormProps?.wid).toBe(10);

      lastProviderFormProps?.onSuccess?.();
      expect(mockNavigate).toHaveBeenCalledWith("/agent/providers");

      mockNavigate.mockClear();
      lastProviderFormProps?.onCancel?.();
      expect(mockNavigate).toHaveBeenCalledWith("/agent/providers");
    });
  });
});
