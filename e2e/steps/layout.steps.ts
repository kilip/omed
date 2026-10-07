import { Then, When } from "@cucumber/cucumber";
import { expect } from "expect";
import { LayoutPage } from "../pages";
import type { CustomWorld } from "./world";

Then(
  "I should see the main navigation with {string}, {string}, {string}, {string}",
  async function (
    this: CustomWorld,
    item1: string,
    item2: string,
    item3: string,
    item4: string,
  ) {
    const layoutPage = new LayoutPage(this.page);
    for (const label of [item1, item2, item3, item4]) {
      const navItem = layoutPage.getNavigationItem(label);
      await navItem.waitFor({ state: "visible" });
      expect(await navItem.isVisible()).toBe(true);
    }
  },
);

When(
  "I click {string} in the main navigation",
  async function (this: CustomWorld, itemName: string) {
    const layoutPage = new LayoutPage(this.page);
    await layoutPage.clickNavigationItem(itemName);
  },
);

When(
  "I switch the language to {string}",
  async function (this: CustomWorld, langLabel: string) {
    const layoutPage = new LayoutPage(this.page);
    await layoutPage.switchLanguage(langLabel);
  },
);
