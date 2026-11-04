import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class AgentLauncherPage extends BasePage {
  readonly path = "/agent";

  get title(): Locator {
    return this.page.getByRole("heading", {
      name: /^(AI Agent)$/i,
      exact: true,
    });
  }

  get subtitle(): Locator {
    return this.page.getByText(
      /^(AI model providers and agent configuration|Penyedia model AI dan konfigurasi agen)$/i,
    );
  }

  launcherTile(name: string | RegExp): Locator {
    return this.page.locator(".app-tile").filter({ hasText: name });
  }

  async expectLoaded(): Promise<void> {
    await this.title.waitFor({ state: "visible", timeout: 10_000 });
    await this.subtitle.waitFor({ state: "visible", timeout: 10_000 });
    await this.launcherTile(/^(Providers|Penyedia)$/i)
      .first()
      .waitFor({ state: "visible", timeout: 10_000 });
    await this.launcherTile(/^(Settings|Pengaturan)$/i)
      .first()
      .waitFor({ state: "visible", timeout: 10_000 });
  }

  async clickTile(name: string): Promise<void> {
    const tile = this.launcherTile(name);
    await tile.first().waitFor({ state: "visible", timeout: 10_000 });
    await tile.first().click();
  }
}
