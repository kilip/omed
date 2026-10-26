import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import type { Role } from "~/lib/roles";
import { RoleSelector } from "./role-selector";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { defaultValue?: string } | string) => {
      if (typeof options === "object" && options?.defaultValue) {
        return options.defaultValue;
      }
      if (typeof options === "string") return options;
      return key;
    },
  }),
}));

describe("RoleSelector", () => {
  const mockRoles: Role[] = [
    {
      id: "core:admin",
      module: "core",
      name: "Workspace Admin",
      description: "Full workspace permissions",
    },
    {
      id: "core:member",
      module: "core",
      name: "Workspace Member",
      description: "Standard member permissions",
    },
    {
      id: "fin:accountant",
      module: "fin",
      name: "Accountant",
      description: "Can post journal entries",
    },
  ];

  it("renders role groups, names, descriptions, and tags", () => {
    const html = renderToString(
      <RoleSelector roles={mockRoles} value={["core:admin"]} />,
    );

    // Module labels and count tag
    expect(html).toContain("Workspace Admin");
    expect(html).toContain("Full workspace permissions");
    expect(html).toContain("Workspace Member");
    expect(html).toContain("Accountant");
    expect(html).toContain("core:admin");
    expect(html).toContain("fin:accountant");
  });

  it("renders with empty roles gracefully", () => {
    const html = renderToString(<RoleSelector roles={[]} />);
    expect(html).toContain("flex flex-col gap-3");
  });

  it("marks checked item with themed border and tint background", () => {
    const html = renderToString(
      <RoleSelector roles={mockRoles} value={["fin:accountant"]} />,
    );

    expect(html).toContain("var(--omed-primary)");
    expect(html).toContain("var(--omed-tint)");
  });

  it("handles disabled prop", () => {
    const html = renderToString(
      <RoleSelector roles={mockRoles} value={["core:admin"]} disabled />,
    );

    expect(html).toContain("ant-radio-disabled");
  });
});
