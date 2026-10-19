const config = {
  extends: ["@commitlint/config-conventional"],
  rules: {
    // Enforce scopes that match the monorepo structure
    "scope-enum": [
      2,
      "always",
      [
        // apps
        "auth",
        "web",
        "api",
        "e2e",
        // infra / cross-cutting
        "deps",
        "ci",
        "release",
        "tools",
        "docs",
      ],
    ],
    // Scope is required
    "scope-empty": [2, "never"],
  },
};

export default config;
