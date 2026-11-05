import { describe, expect, it, vi } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import { COMMON_LOCALES, clientLoader } from "./settings";

vi.mock("~/lib/workspace", () => ({
  requireWorkspaceId: vi.fn().mockResolvedValue(10),
}));

describe("CMS Settings Route", () => {
  it("clientLoader resolves workspace ID", async () => {
    const res = await clientLoader();
    expect(res).toEqual({ wid: 10 });
  });

  it("contains standard COMMON_LOCALES", () => {
    expect(COMMON_LOCALES.some((l) => l.value === "en")).toBe(true);
    expect(COMMON_LOCALES.some((l) => l.value === "id")).toBe(true);
  });

  it("has matching i18n keys for settings in en and id", () => {
    expect(en.cms.settings.title).toBeDefined();
    expect(id.cms.settings.title).toBeDefined();
    expect(en.cms.settings.defaultLocaleLabel).toBeDefined();
    expect(id.cms.settings.defaultLocaleLabel).toBeDefined();
    expect(en.cms.settings.supportedLocalesLabel).toBeDefined();
    expect(id.cms.settings.supportedLocalesLabel).toBeDefined();
    expect(en.cms.settings.errors.defaultLocaleChangeBlocked).toBeDefined();
    expect(id.cms.settings.errors.defaultLocaleChangeBlocked).toBeDefined();
    expect(en.cms.settings.errors.localeInUse).toBeDefined();
    expect(id.cms.settings.errors.localeInUse).toBeDefined();
  });
});
