import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class CmsLauncherPage extends BasePage {
  readonly path = "/cms";

  get title(): Locator {
    return this.page.getByRole("heading", {
      name: /^(Content|Konten)$/i,
      exact: true,
    });
  }

  get subtitle(): Locator {
    return this.page.getByText(
      /^(Posts, pages, and media library|Artikel, halaman, dan pustaka media)$/i,
    );
  }

  launcherTile(name: string | RegExp): Locator {
    return this.page.locator(".app-tile").filter({ hasText: name });
  }

  async expectLoaded(): Promise<void> {
    await this.title.waitFor({ state: "visible", timeout: 10_000 });
    await this.subtitle.waitFor({ state: "visible", timeout: 10_000 });
    await this.launcherTile(/^(Posts|Artikel)$/i)
      .first()
      .waitFor({ state: "visible", timeout: 10_000 });
    await this.launcherTile(/^(Pages|Halaman)$/i)
      .first()
      .waitFor({ state: "visible", timeout: 10_000 });
    await this.launcherTile(/^(Media)$/i)
      .first()
      .waitFor({ state: "visible", timeout: 10_000 });
  }

  async clickTile(name: string): Promise<void> {
    const tile = this.launcherTile(name);
    await tile.first().waitFor({ state: "visible", timeout: 10_000 });
    await tile.first().click();
  }
}
