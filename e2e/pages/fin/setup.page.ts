import type { Locator } from "playwright";
import { BasePage } from "../base.page";

export class FinSetupPage extends BasePage {
  readonly path = "/fin/onboarding";

  get nextButton(): Locator {
    return this.page.getByRole("button", { name: /next|lanjut/i });
  }

  get finishButton(): Locator {
    return this.page.getByRole("button", {
      name: /complete setup|selesaikan setup|apply template/i,
    });
  }

  get applyButton(): Locator {
    return this.finishButton;
  }

  get fiscalYearInput(): Locator {
    return this.page.getByRole("spinbutton", {
      name: /fiscal year|tahun fiskal/i,
    });
  }

  get periodNameInput(): Locator {
    return this.page.getByRole("textbox", {
      name: /period name|nama periode/i,
    });
  }

  async clickNext(): Promise<void> {
    await this.nextButton.waitFor({ state: "visible" });
    await this.nextButton.click();
  }

  async clickFinish(): Promise<void> {
    await this.finishButton.waitFor({ state: "visible" });
    await this.finishButton.click();
    await this.page.waitForURL(
      (url) => url.pathname === "/fin" || url.pathname === "/fin/",
      { timeout: 15_000 },
    );
  }

  async clickApplyTemplate(): Promise<void> {
    await Promise.race([
      this.nextButton.waitFor({ state: "visible" }).catch(() => {}),
      this.finishButton.waitFor({ state: "visible" }).catch(() => {}),
    ]);
    if (await this.nextButton.isVisible()) {
      await this.selectDecimalStyle("dot");
      await this.nextButton.click();
    }
    await this.finishButton.waitFor({ state: "visible" });
    await this.finishButton.click();
    await this.page.waitForURL(
      (url) => url.pathname === "/fin" || url.pathname === "/fin/",
      { timeout: 15_000 },
    );
  }

  async selectDecimalStyle(style: "dot" | "comma"): Promise<void> {
    const text =
      style === "dot"
        ? /dot decimal|format titik/i
        : /comma decimal|format koma/i;
    const card = this.page
      .locator(".ant-col .ant-card")
      .filter({ hasText: text });
    await card.waitFor({ state: "visible" });
    await card.click();
  }

  async expectPeriodFormVisible(): Promise<void> {
    await this.fiscalYearInput.waitFor({ state: "visible" });
    await this.periodNameInput.waitFor({ state: "visible" });
  }

  get previewTable(): Locator {
    return this.page.locator(".ant-table");
  }

  async expectAccountInPreview(code: string): Promise<void> {
    await this.previewTable
      .locator(`text=${code}`)
      .first()
      .waitFor({ state: "visible" });
  }
}
