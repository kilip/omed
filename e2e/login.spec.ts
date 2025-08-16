import { test, expect } from "@playwright/test";

test("has title", async ({ page }) => {
  await page.goto("http://localhost:5173/");

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle("Login to Omed");
  await expect(page.getByRole("heading")).toHaveText("Login to Omed");

  await page.locator("[name='email']").fill("admin@example.com");
  await page.locator("[name='password']").fill("admin");
  await page.locator("[name='submit']").click();

  await expect(page).toHaveURL("http://localhost:5173/");
  await expect(page).toHaveTitle("Dashboard");
  await expect(page.getByRole("heading", { level: 3 })).toContainText(
    "Omed Admin User",
  );
});
