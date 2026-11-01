# Omed

[![codecov](https://img.shields.io/codecov/c/github/kilip/omed)](https://codecov.io/gh/kilip/omed)

**Omed** is a modular, multi-workspace business platform that brings together team collaboration, double-entry accounting, content management, and AI agent integrations in one place.

---

## Key Features

### 🏢 Workspaces & Team Management (`Core`)
- **Multi-Workspace Organization:** Every user gets a personal workspace and can create dedicated workspaces for different businesses, teams, or projects.
- **Team Collaboration & Invitations:** Invite team members via email and manage pending invitations.
- **Role-Based Access Control:** Assign granular roles per workspace so members only see and manage the modules and data they are authorized for.
- **Social Sign-In:** Quick authentication with Google and GitHub.
- **Personal Access Tokens (PAT):** Generate time-bounded access tokens (`30`, `90`, or `365` days) to connect external AI assistants and tools safely.
- **Bilingual Interface:** Built-in support for English (`en`) and Indonesian (`id`).

### 💰 Finance & Accounting (`Fin`)
- **Guided Financial Onboarding:** Step-by-step setup wizard for base currency, fiscal periods, and starter Chart of Accounts templates.
- **Chart of Accounts & Fiscal Periods:** Organize accounts in a hierarchical tree, configure opening balances, and manage fiscal periods.
- **Income & Expense Tracking:** Record multi-item income and expense documents with file attachments, search, and draft-or-post workflows.
- **Double-Entry Bookkeeping:** Create balanced journal vouchers and business transactions with support for posting, reversing, and voiding entries.
- **General Ledger & Trial Balance:** Inspect account-level running balances with direct links back to journal entries and generate trial balance summaries.

### 📝 Content Management (`CMS`)
- **Posts & Standalone Pages:** Write and manage blog posts and static pages with a live-preview Markdown editor, excerpts, cover media, and automatic slug generation.
- **Publishing Workflow:** Save drafts, schedule content for future publication, publish immediately, or unpublish at any time.
- **Media Library:** Drag-and-drop file uploads with metadata management (titles, alt text, captions), public/private visibility toggles, and protection against deleting media currently in use.
- **Public Content API:** Built-in read-only public endpoints for serving published posts, pages, and media to external websites or frontends.

### 🤖 AI & Agent Integration (`Agent` & `MCP`)
- **LLM Provider Management:** Configure workspace AI providers—including **Anthropic**, **OpenAI**, **DeepSeek**, **Ollama**, and custom OpenAI-compatible endpoints—with encrypted API credentials, default model selection, and live connection testing.
- **Built-In MCP Server:** Connect AI clients (such as Claude Desktop or other Model Context Protocol tools) directly to Omed using a Personal Access Token to query monthly financial overviews, inspect account balances, and search cashflow records.

---

## Quick Start

The fastest way to run Omed locally is with the included **Devcontainer** (or GitHub Codespaces), which sets up all required tools and services automatically.

```bash
git clone https://github.com/kilip/omed.git
cd omed

bun install                            # install dependencies & git hooks
cd apps/api && go mod tidy && cd ../.. # sync Go dependencies

bun run db:migrate                     # apply database migrations
bun run dev                            # start web (port 3000) & api (port 8001)
```

Once running, open **<http://localhost:3000>** in your browser.

---

## Documentation

Detailed technical guides live in the [`docs/`](./docs) directory:

- **[Architecture & Modules](./docs/architecture.md)** — Monorepo layout, tech stack, module routes, permissions, and MCP server specification.
- **[Configuration & Environment Variables](./docs/configuration.md)** — Full list of API (`OMED_*`) and Web (`VITE_*` / `OMED_WEB_*`) environment variables.
- **[Development Guide](./docs/development.md)** — Prerequisites, local setup, database migrations (Ent + Atlas), OpenAPI type generation, and testing.
- **[Deployment Guide](./docs/deployment.md)** — Docker image builds, runtime configuration, production migrations, and Kubernetes Helm chart deployment.
- **[Branding & Visual Guidelines](./BRANDING.md)** — Design tokens, typography, and dashboard UI guidelines.
- **[AI Agent Guidelines](./AGENTS.md)** — Conventions and workflows for AI coding assistants.
- **[Contributing Guide](./CONTRIBUTING.md)** — Pull request workflow, branch naming, commit conventions, and code standards.

## License

[MIT](./LICENSE) © Anthonius

---

Made with ❤️ and ☕
