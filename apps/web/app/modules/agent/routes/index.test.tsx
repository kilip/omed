import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import AgentIndex, { agentLauncherItems } from "./index";

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

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({
      children,
      to,
      className,
    }: {
      children: React.ReactNode;
      to: string;
      className?: string;
    }) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
  };
});

describe("AgentIndex launcher component", () => {
  it("renders tiles linking to agent providers and settings", () => {
    const html = renderToString(<AgentIndex />);

    expect(html).toContain("home-backdrop");
    expect(html).toContain("app-tile");
    expect(html).toContain('href="/agent/providers"');
    expect(html).toContain('href="/agent/settings"');
    expect(html).toContain("agent.menu.providers");
    expect(html).toContain("agent.menu.settings");
  });

  it("exports launcher items matching agent sub-routes", () => {
    expect(agentLauncherItems).toHaveLength(2);
    expect(agentLauncherItems[0]).toMatchObject({
      key: "providers",
      to: "/agent/providers",
      labelKey: "agent.menu.providers",
    });
    expect(agentLauncherItems[1]).toMatchObject({
      key: "settings",
      to: "/agent/settings",
      labelKey: "agent.menu.settings",
    });
  });
});
