/**
 * @type {import('@cucumber/cucumber').IConfiguration}
 */

export default {
  format: [
    "progress-bar",
    "html:test-reports/cucumber.html",
    "json:test-reports/cucumber.json",
  ],
  parallel: 2,
  paths: ["e2e/features/**/*.feature"],
  publishQuiet: true,
  require: ["e2e/steps/**/*.ts", "e2e/support/**/*.ts"],
  requireModule: ["tsx/cjs"],
  retry: 0,
  tags: "not @skip",
  timeout: 30_000,
};
