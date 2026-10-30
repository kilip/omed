import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { setActiveDecimalStyle } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { finSettingsQueryOptions } from "../queries";
import FinLayout, { clientLoader } from "./layout";

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn(),
}));

vi.mock("~/lib/query-client", () => ({
  getQueryClient: vi.fn(),
}));

vi.mock("~/lib/format", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/format")>();
  return {
    ...actual,
    setActiveDecimalStyle: vi.fn(),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Outlet: () => <div data-testid="outlet">Fin Layout Outlet</div>,
    redirect: vi.fn((path: string) => {
      const response = new Response(null, {
        status: 302,
        headers: { Location: path },
      });
      return response;
    }),
  };
});

describe("FinLayout clientLoader", () => {
  const wid = 12;
  const mockEnsureQueryData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(requireWorkspaceId).mockResolvedValue(wid);
    vi.mocked(getQueryClient).mockReturnValue({
      ensureQueryData: mockEnsureQueryData,
    } as unknown as ReturnType<typeof getQueryClient>);
  });

  it("redirects to /fin/onboarding when settings.onboarding is false", async () => {
    mockEnsureQueryData.mockResolvedValueOnce({
      baseCurrency: "IDR",
      decimalStyle: "comma",
      onboarding: false,
    });

    const res = await clientLoader();

    expect(requireWorkspaceId).toHaveBeenCalled();
    expect(getQueryClient).toHaveBeenCalled();
    expect(mockEnsureQueryData).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: finSettingsQueryOptions(wid).queryKey,
      }),
    );
    expect(setActiveDecimalStyle).toHaveBeenCalledWith("comma");

    // Loader returns redirect response
    expect(res).toBeInstanceOf(Response);
    expect((res as Response).status).toBe(302);
    expect((res as Response).headers.get("Location")).toBe("/fin/onboarding");
  });

  it("returns wid and settings and sets active decimal style when onboarding is true", async () => {
    const settings = {
      baseCurrency: "USD",
      decimalStyle: "dot" as const,
      onboarding: true,
    };
    mockEnsureQueryData.mockResolvedValueOnce(settings);

    const result = await clientLoader();

    expect(requireWorkspaceId).toHaveBeenCalled();
    expect(setActiveDecimalStyle).toHaveBeenCalledWith("dot");
    expect(result).toEqual({ wid, settings });
  });

  it("defaults active decimal style to comma if decimalStyle is unset", async () => {
    const settings = {
      baseCurrency: "USD",
      decimalStyle: undefined,
      onboarding: true,
    };
    mockEnsureQueryData.mockResolvedValueOnce(settings);

    const result = await clientLoader();

    expect(setActiveDecimalStyle).toHaveBeenCalledWith("comma");
    expect(result).toEqual({ wid, settings });
  });
});

describe("FinLayout component", () => {
  it("renders Outlet", () => {
    const html = renderToString(<FinLayout />);
    expect(html).toContain("Fin Layout Outlet");
  });
});
