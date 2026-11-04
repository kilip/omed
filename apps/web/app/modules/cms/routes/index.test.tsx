import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import CmsIndex, { cmsLauncherItems } from "./index";

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

describe("CmsIndex launcher component", () => {
  it("renders tiles linking to cms posts, pages, and media", () => {
    const html = renderToString(<CmsIndex />);

    expect(html).toContain("home-backdrop");
    expect(html).toContain("app-tile");
    expect(html).toContain('href="/cms/posts"');
    expect(html).toContain('href="/cms/pages"');
    expect(html).toContain('href="/cms/media"');
    expect(html).toContain("cms.menu.posts");
    expect(html).toContain("cms.menu.pages");
    expect(html).toContain("cms.menu.media");
  });

  it("exports launcher items matching cms sub-routes", () => {
    expect(cmsLauncherItems).toHaveLength(3);
    expect(cmsLauncherItems[0]).toMatchObject({
      key: "posts",
      to: "/cms/posts",
      labelKey: "cms.menu.posts",
    });
    expect(cmsLauncherItems[1]).toMatchObject({
      key: "pages",
      to: "/cms/pages",
      labelKey: "cms.menu.pages",
    });
    expect(cmsLauncherItems[2]).toMatchObject({
      key: "media",
      to: "/cms/media",
      labelKey: "cms.menu.media",
    });
  });
});
