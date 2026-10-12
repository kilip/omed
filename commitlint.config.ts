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
				"dash",
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
		// Scope is optional (bare `feat: ...` is fine)
		"scope-empty": [0],
	},
};

export default config;
