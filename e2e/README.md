# @omed/e2e

End-to-End (E2E) testing suite for Omed using [Cucumber](https://cucumber.io/) (`@cucumber/cucumber`) with [Playwright](https://playwright.dev/) and Page Object Model (POM).

Tests are written in Gherkin (`.feature`) files and executed via Cucumber in Chromium against live servers (`auth` on `:8001`, `dash` on `:3001`, and `finance` on `:8002`), with an isolated PostgreSQL database (`omed_e2e`).

## Directory Structure

```
e2e/
├── features/            # Gherkin feature files (.feature)
│   ├── auth/            # Access control and login page scenarios
│   ├── home.feature     # Dashboard navigation and overview
│   ├── language.feature # Language switching and persistence
│   └── settings.feature # Account settings (currency, language, theme)
├── pages/               # Page Object Model (POM)
│   ├── BasePage.ts      # Shared page base class and common interactions
│   ├── LoginPage.ts     # Login page locators and actions
│   ├── LayoutPage.ts    # Sidebar navigation & language switcher
│   ├── SettingsPage.ts  # Settings form (currency, language, theme)
│   └── index.ts         # Export barrel for pages
├── steps/               # Step definitions and hooks
│   ├── world.ts         # Cucumber CustomWorld with page object initializers & session injection
│   ├── common.steps.ts  # Generic navigation and assertions
│   ├── layout.steps.ts  # Navigation and layout steps
│   └── settings.steps.ts# Settings-specific steps
├── scripts/
│   └── reset-db.ts      # Recreates and pushes auth schema to omed_e2e
├── env.ts               # Test environment configuration
├── servers.ts           # Server manager for starting and stopping auth, dash, and finance
└── cucumber.json        # Cucumber configuration
```

## Running Tests

Ensure PostgreSQL is running (e.g. via Dev Container or local Docker container):

```bash
# 1. Install Playwright browser (first time)
bun --cwd e2e playwright install chromium

# 2. Run E2E tests (automatically resets omed_e2e and runs Cucumber)
bun run e2e

# 3. Dry run step definitions match
bun run --cwd e2e test:dry
```

> **Note:** E2E runs against ports `8001` (auth), `3001` (dash), and `8002` (finance). Make sure any running `bun run dev` server is stopped before running E2E.
