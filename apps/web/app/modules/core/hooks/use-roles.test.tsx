import type { UseSuspenseQueryResult } from "@tanstack/react-query";
import { useSuspenseQuery } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { roleKeys } from "../queries";
import { useRoles } from "./use-roles";

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { defaultValue?: string } | string) => {
      if (typeof options === "object" && options?.defaultValue) {
        return options.defaultValue;
      }
      if (typeof options === "string") {
        return options;
      }
      return key;
    },
  }),
}));

describe("useRoles hook", () => {
  const wid = 5;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("resolves roles, labels, and select options", () => {
    const mockRoles = [
      { id: "core:admin", name: "Administrator", description: "Admin access" },
      { id: "core:member", name: "Member", description: "Normal member" },
      { id: "", name: "Unnamed", description: "Empty id should be filtered" },
    ];

    vi.mocked(useSuspenseQuery).mockReturnValue({
      data: mockRoles,
    } as unknown as UseSuspenseQueryResult<typeof mockRoles>);

    let result: ReturnType<typeof useRoles> | undefined;
    function TestComponent() {
      result = useRoles(wid);
      return null;
    }

    renderToString(<TestComponent />);

    expect(useSuspenseQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: roleKeys.all(wid),
      }),
    );
    expect(result).toBeDefined();
    expect(result?.roles).toEqual(mockRoles);

    // Test label resolver
    expect(result?.label("core:admin")).toBe("Administrator");
    expect(result?.label("core:member")).toBe("Member");
    expect(result?.label("unknown:role")).toBe("unknown:role");

    // Test options formatting (excluding empty id)
    expect(result?.options).toEqual([
      {
        value: "core:admin",
        label: "Administrator",
        title: "Admin access",
      },
      {
        value: "core:member",
        label: "Member",
        title: "Normal member",
      },
    ]);
  });
});
