import { Then, When } from "@cucumber/cucumber";
import { expect } from "expect";
import { SettingsPage } from "../pages";
import type { CustomWorld } from "./world";

Then(
  "the default currency should be {string}",
  async function (this: CustomWorld, expectedCurrency: string) {
    const settingsPage = new SettingsPage(this.page);
    const text = await settingsPage.getSelectedCurrency();
    expect(text).toContain(expectedCurrency);
  },
);

When(
  "I set the default currency to {string}",
  async function (this: CustomWorld, newCurrency: string) {
    const settingsPage = new SettingsPage(this.page);
    await settingsPage.setCurrency(newCurrency);
  },
);

Then(
  "I should see the message {string}",
  async function (this: CustomWorld, expectedMsg: string) {
    const settingsPage = new SettingsPage(this.page);
    const msg = await settingsPage.expectNoticeMessage(expectedMsg);
    expect(await msg.isVisible()).toBe(true);
  },
);

When(
  "I choose the language {string}",
  async function (this: CustomWorld, languageLabel: string) {
    const settingsPage = new SettingsPage(this.page);
    await settingsPage.selectLanguageRadio(languageLabel);
  },
);

When(
  "I choose the theme {string}",
  async function (this: CustomWorld, themeLabel: string) {
    const settingsPage = new SettingsPage(this.page);
    await settingsPage.selectThemeRadio(themeLabel);
  },
);

Then("the page should use the dark theme", async function (this: CustomWorld) {
  const settingsPage = new SettingsPage(this.page);
  const themeAttr = await settingsPage.getHtmlThemeAttribute();
  expect(themeAttr).toBe("dark");
});
