import { After, AfterAll, Before, Status } from "@cucumber/cucumber";
import { type CustomWorld, closeSharedBrowser } from "../support/world";

Before(async function (this: CustomWorld) {
  await this.init();
});

After(async function (this: CustomWorld, { result, pickle }) {
  const testId = pickle.tags
    .find(
      (tag) =>
        tag.name.startsWith("@COMMUNITY-") ||
        tag.name.startsWith("@AGENT-") ||
        tag.name.startsWith("@HOME-") ||
        tag.name.startsWith("@OIDC-") ||
        tag.name.startsWith("@PAGE-") ||
        tag.name.startsWith("@ROUTES-"),
    )
    ?.name.replace("@", "");

  if (result?.status === Status.FAILED && this.page) {
    const screenshot = await this.takeScreenshot(
      `${testId || "failure"}-${Date.now()}`,
    );
    this.attach(screenshot, "image/png");

    const html = await this.page.content();
    this.attach(html, "text/html");
    if (this.testContext.jsErrors.length > 0) {
      const errors = this.testContext.jsErrors.map((e) => e.message).join("\n");
      this.attach(`JavaScript Errors:\n${errors}`, "text/plain");
    }

    console.log(`❌ Failed: ${pickle.name}`);
    if (result.message) {
      console.log(`   Error: ${result.message}`);
    }
  } else if (result?.status === Status.FAILED) {
    console.log(`❌ Failed before page initialization: ${pickle.name}`);
    if (result.message) {
      console.log(`   Error: ${result.message}`);
    }
  } else if (result?.status === Status.PASSED) {
    console.log(`✅ Passed: ${pickle.name}`);
  }

  await this.cleanup();
});

AfterAll(async () => {
  console.log("\n🏁 Test suite completed");

  await closeSharedBrowser();
});
