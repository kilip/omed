import type { UseQueryResult } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { permissionKeys } from "../queries";
import { usePermissions } from "./use-permissions";

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQuery: vi.fn(),
  };
});

describe("usePermissions", () => {
  const wid = 7;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns permissions data when available", () => {
    const mockData = {
      roles: ["admin"],
      permissions: ["core:workspace:manage"],
      owner: true,
    };
    vi.mocked(useQuery).mockReturnValue({
      data: mockData,
    } as unknown as UseQueryResult<typeof mockData>);

    const result = usePermissions(wid);

    expect(useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: permissionKeys.all(wid),
      }),
    );
    expect(result).toEqual(mockData);
  });

  it("returns default fallback when data is undefined", () => {
    vi.mocked(useQuery).mockReturnValue({
      data: undefined,
    } as unknown as UseQueryResult<undefined>);

    const result = usePermissions(wid);

    expect(useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: permissionKeys.all(wid),
      }),
    );
    expect(result).toEqual({ roles: [], permissions: [], owner: false });
  });
});
