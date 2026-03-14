# Project OMED — Implementation Checklist

**Version:** 1.0.0
**Reference:** OMED_PRD.md
**Last Updated:** 2026-03-14

---

> ✅ = Done | 🔄 = In Progress | ⬜ = Not Started

---

## Phase 0 — Repository Setup

### 0.1 Initialize Monorepo
- [ ] Create GitHub repository `omed`
- [ ] Initialize root `Taskfile.yml` (taskfile.dev)
- [ ] Create root `docker-compose.yml` (postgres, redis, minio, api, dash, blog)
- [ ] Create root `.env.example` with all required variables
- [ ] Create root `.gitignore`
- [ ] Create `./data/` folder (bind mount target, add to .gitignore)

### 0.2 Initialize Turborepo Workspace
- [ ] Create `ui/package.json` (Turborepo workspace root)
- [ ] Create `ui/turbo.json` (⚠️ in `/ui/` NOT root)
- [ ] Create `ui/.gitignore`

---

## Phase 1 — Shared Packages

### 1.1 `@omed/tsconfig`
- [ ] Create `ui/packages/tsconfig/package.json` (`name: @omed/tsconfig`)
- [ ] Create `ui/packages/tsconfig/base.json`
- [ ] Create `ui/packages/tsconfig/nextjs.json`
- [ ] Create `ui/packages/tsconfig/astro.json`

### 1.2 `@omed/util`
- [ ] Create `ui/packages/util/package.json` (`name: @omed/util`)
- [ ] Create `ui/packages/util/tsconfig.json` (extends `@omed/tsconfig/base.json`)
- [ ] Install dependencies: `zod`, `dotenv`
- [ ] Create `ui/packages/util/src/singleton.ts`
- [ ] Create `ui/packages/util/src/zod.ts` (zod re-export)
- [ ] Create `ui/packages/util/src/config.ts` (Zod-validated env config with singleton)
- [ ] Create `ui/packages/util/src/index.ts` (thin re-export)

### 1.3 `@omed/api-client` (scaffold only — generated later)
- [ ] Create `ui/packages/api-client/package.json` (`name: @omed/api-client`)
- [ ] Create `ui/packages/api-client/tsconfig.json` (extends `@omed/tsconfig/base.json`)
- [ ] Install dependencies: `openapi-typescript`
- [ ] Create `ui/packages/api-client/src/client.ts` (createClient setup)
- [ ] Create `ui/packages/api-client/src/index.ts` (thin re-export)
- [ ] Create `ui/packages/api-client/src/generated/` folder with `.gitkeep`
- [ ] Add `src/generated/` to `.gitignore`

---

## Phase 2 — Infrastructure (Docker)

### 2.1 Docker Compose Services
- [ ] Configure PostgreSQL service (port 5432, bind mount `./data/postgres`)
- [ ] Configure Redis service (port 6379, bind mount `./data/redis`)
- [ ] Configure MinIO service (port 9000 + 9001, bind mount `./data/minio`)
- [ ] Configure Go API service (port 8080)
- [ ] Configure Cockpit service (port 3000)
- [ ] Configure Astro Blog service (port 4321)
- [ ] Add `task docker:up` to Taskfile
- [ ] Verify `task docker:up` starts all 6 services successfully

---

## Phase 3 — Engine (Go API)

### 3.1 Go Project Init
- [ ] Initialize Go module in `api/` (`go mod init github.com/omed/api`)
- [ ] Install Go Fiber **v3** (`go get github.com/gofiber/fiber/v3`)
  - ⚠️ MUST be v3, NOT v2
- [ ] Install EntGo (`go get entgo.io/ent`)
- [ ] Install Redis client (`go get github.com/redis/go-redis/v9`)
- [ ] Install MinIO client (`go get github.com/minio/minio-go/v7`)
- [ ] Install swaggo (`go get github.com/swaggo/swag`)
- [ ] Install gofiber/swagger (`go get github.com/gofiber/swagger`)
- [ ] Install uber-go/mock (`go get go.uber.org/mock`)
- [ ] Install testify (`go get github.com/stretchr/testify`)
- [ ] Install dotenv (`go get github.com/joho/godotenv`)

### 3.2 Infrastructure Layer
- [ ] Create `api/infrastructure/Postgres.go` (PostgreSQL connection)
- [ ] Create `api/infrastructure/Redis.go` (Redis connection)
- [ ] Create `api/infrastructure/Minio.go` (MinIO connection)
- [ ] Verify all 3 infrastructure connections work with `task docker:up`

### 3.3 Domain Layer
- [ ] Create `api/domain/Tenant.go` (struct + interface)
- [ ] Create `api/domain/TenantUser.go` (struct + interface)
- [ ] Create `api/domain/Article.go` (struct + interface)
- [ ] Create `api/domain/Tag.go` (struct + interface)
- [ ] Create `api/domain/Comment.go` (struct + interface)
- [ ] Create `api/domain/Media.go` (struct + interface)
- [ ] Add `//go:generate mockgen` annotations to all domain interfaces

### 3.4 EntGo Schema
- [ ] Initialize EntGo (`go run entgo.io/ent/cmd/ent new`)
- [ ] Create EntGo schema for `Tenant`
- [ ] Create EntGo schema for `TenantUser`
- [ ] Create EntGo schema for `Article` (with status ENUM: draft/review/published/archived)
- [ ] Create EntGo schema for `Tag`
- [ ] Create EntGo schema for `ArticleTag` (pivot)
- [ ] Create EntGo schema for `Comment`
- [ ] Create EntGo schema for `Media`
- [ ] Generate EntGo code (`go generate ./ent/...`)
- [ ] Add `task db:migrate` to Taskfile
- [ ] Run `task db:migrate` and verify all 7 tables created in PostgreSQL

### 3.5 Repository Layer
- [ ] Create `api/repository/ArticleRepository.go`
- [ ] Create `api/repository/TagRepository.go`
- [ ] Create `api/repository/CommentRepository.go`
- [ ] Create `api/repository/MediaRepository.go`
- [ ] Create `api/repository/TenantRepository.go`

### 3.6 Usecase Layer
- [ ] Create `api/usecase/ArticleUsecase.go`
- [ ] Create `api/usecase/TagUsecase.go`
- [ ] Create `api/usecase/CommentUsecase.go`
- [ ] Create `api/usecase/MediaUsecase.go`
- [ ] Create `api/usecase/TenantUsecase.go`

### 3.7 Delivery — Middleware
- [ ] Create `api/delivery/middleware/AuthMiddleware.go` (Redis session verify)
- [ ] Create `api/delivery/middleware/ApiKeyMiddleware.go` (X-API-Key verify)
- [ ] Create `api/delivery/middleware/TenantMiddleware.go` (extract + inject tenant_id)
- [ ] Create `api/delivery/middleware/RateLimitMiddleware.go` (Redis-based, per IP + per Tenant)
- [ ] Configure CORS middleware (exact origins, methods, headers per PRD)

### 3.8 Delivery — Response Structs
- [ ] Create `api/delivery/handler/response/ArticleResponse.go`
- [ ] Create `api/delivery/handler/response/TagResponse.go`
- [ ] Create `api/delivery/handler/response/CommentResponse.go`
- [ ] Create `api/delivery/handler/response/MediaResponse.go`
- [ ] Create `api/delivery/handler/response/ErrorResponse.go`

### 3.9 Delivery — Handlers
- [ ] Create `api/delivery/handler/ArticleHandler.go` with swaggo annotations
  - [ ] `GET /articles` (list + pagination + filter)
  - [ ] `GET /articles/:slug`
  - [ ] `POST /articles`
  - [ ] `PUT /articles/:id`
  - [ ] `DELETE /articles/:id`
  - [ ] `POST /articles/:id/publish`
- [ ] Create `api/delivery/handler/TagHandler.go` with swaggo annotations
  - [ ] `GET /tags`
- [ ] Create `api/delivery/handler/CommentHandler.go` with swaggo annotations
  - [ ] `POST /comments`
  - [ ] `GET /comments/:article_id`
- [ ] Create `api/delivery/handler/MediaHandler.go` with swaggo annotations

### 3.10 Main & API Docs
- [ ] Create `api/main.go` (wire all layers)
- [ ] Add `task docs:generate` to Taskfile (`swag init`)
- [ ] Run `task docs:generate` → verify `api/docs/openapi.json` generated
- [ ] Verify Scalar UI accessible at `http://localhost:8080/docs`
- [ ] Add `task mock:generate` to Taskfile (`mockgen --all`)

### 3.11 Engine Tests
- [ ] Run `task mock:generate` → verify mocks generated in `domain/mocks/`
- [ ] Write unit tests for `ArticleUsecase`
- [ ] Write unit tests for `TagUsecase`
- [ ] Write unit tests for `CommentUsecase`
- [ ] Write unit tests for `ArticleHandler`
- [ ] Write integration tests for all 9 API endpoints
- [ ] Run `task test:unit` → all tests pass
- [ ] Run `task test:coverage` → check coverage report

---

## Phase 4 — Cockpit (Next.js Dashboard)

### 4.1 Next.js Project Init
- [ ] Initialize Next.js 16 in `ui/dash/` with TypeScript + App Router
- [ ] Configure `ui/dash/tsconfig.json` to extend `@omed/tsconfig/nextjs.json`
- [ ] Add `@omed/util`, `@omed/api-client`, `@omed/tsconfig` as workspace dependencies
- [ ] Create `ui/dash/.env` with required variables

### 4.2 shadcn/ui Setup
- [ ] Run `npx shadcn create` in `ui/dash/`
  - [ ] Select component library: **Base UI** (NOT Radix)
  - [ ] Select style: **Mira**
  - [ ] Select icons: **Lucide**
- [ ] Verify `src/shared/ui/` contains generated components
- [ ] Add required components via CLI:
  - [ ] `npx shadcn add button`
  - [ ] `npx shadcn add input`
  - [ ] `npx shadcn add table`
  - [ ] `npx shadcn add dialog`
  - [ ] `npx shadcn add form`
  - [ ] `npx shadcn add tabs`
  - [ ] `npx shadcn add badge`
  - [ ] `npx shadcn add card`
  - [ ] `npx shadcn add dropdown-menu`
  - [ ] `npx shadcn add toast`

### 4.3 Better-Auth Setup
- [ ] Install Better-Auth (`npm i better-auth`)
- [ ] Install Drizzle ORM (`npm i drizzle-orm`, `npm i -D drizzle-kit`)
- [ ] Configure Better-Auth with Redis session + Drizzle persistence
- [ ] Add `task db:generate` to Taskfile
- [ ] Run `task db:generate` → verify Better-Auth tables generated in PostgreSQL
  - [ ] `users` table created
  - [ ] `accounts` table created
  - [ ] `sessions` table created
  - [ ] `verifications` table created
- [ ] Create `ui/dash/proxy.ts` (⚠️ NOT `middleware.ts`)
  - [ ] Verify `export function proxy()` — NOT `export function middleware()`

### 4.4 TanStack Query Setup
- [ ] Install TanStack Query (`npm i @tanstack/react-query`)
- [ ] Install openapi-react-query (`npm i openapi-react-query`)
- [ ] Configure TanStack Query provider in `src/app/`

### 4.5 FSD Structure Setup
- [ ] Create `ui/dash/src/app/` (providers, layouts)
- [ ] Create `ui/dash/src/pages/` with all 7 page slices:
  - [ ] `login/`
  - [ ] `dashboard/`
  - [ ] `articles/`
  - [ ] `article-edit/`
  - [ ] `tags/`
  - [ ] `media/`
  - [ ] `comments/`
- [ ] Create `ui/dash/src/widgets/`
- [ ] Create `ui/dash/src/features/` with slices: `articles/`, `tags/`, `comments/`, `media/`
- [ ] Create `ui/dash/src/entities/` with slices: `article/`, `tag/`, `comment/`, `media/`
- [ ] Create `ui/dash/src/shared/` with: `api/`, `ui/`, `lib/`, `types/`
- [ ] Create empty `ui/dash/pages/README.md` (required by Next.js)

### 4.6 App Router Setup
- [ ] Create `ui/dash/app/(auth)/login/page.tsx`
  - [ ] Verify: contains ONLY `export { LoginPage as default } from '@/pages/login'`
- [ ] Create `ui/dash/app/(dashboard)/page.tsx`
  - [ ] Verify: contains ONLY `export { DashboardPage as default } from '@/pages/dashboard'`
- [ ] Create `ui/dash/app/(dashboard)/articles/page.tsx`
- [ ] Create `ui/dash/app/(dashboard)/articles/[id]/page.tsx`
- [ ] Create `ui/dash/app/(dashboard)/tags/page.tsx`
- [ ] Create `ui/dash/app/(dashboard)/media/page.tsx`
- [ ] Create `ui/dash/app/(dashboard)/comments/page.tsx`
- [ ] Create `ui/dash/app/layout.tsx`

### 4.7 Pages Implementation
- [ ] Implement `src/pages/login/` (login form with inline error)
- [ ] Implement `src/pages/dashboard/` (stats overview)
- [ ] Implement `src/pages/articles/` (article list with status filter)
- [ ] Implement `src/pages/article-edit/` (Markdown editor — create & edit)
- [ ] Implement `src/pages/tags/` (tag CRUD)
- [ ] Implement `src/pages/media/` (MinIO upload + library)
- [ ] Implement `src/pages/comments/` (approve/reject comments)

### 4.8 Cockpit Tests
- [ ] Install Vitest + Testing Library
- [ ] Install vitest-mock-extended
- [ ] Install Playwright (`npx playwright install`)
- [ ] Write unit tests for key components per FSD slice
- [ ] Write E2E tests:
  - [ ] Login / logout flow
  - [ ] Create article → publish flow
  - [ ] Upload media flow
  - [ ] Approve comment flow
- [ ] Run `task test:unit` → all pass
- [ ] Run `task test:e2e` → all pass

---

## Phase 5 — Blog (Astro SSG)

### 5.1 Astro Project Init
- [ ] Initialize Astro in `ui/blog/`
- [ ] Configure `ui/blog/tsconfig.json` to extend `@omed/tsconfig/astro.json`
- [ ] Add `@omed/util`, `@omed/api-client` as workspace dependencies
- [ ] Create `ui/blog/.env` with required variables
- [ ] Install openapi-fetch (`npm i openapi-fetch`)

### 5.2 Pagefind Setup
- [ ] Install Pagefind (`npm i -D pagefind`)
- [ ] Configure Pagefind in `src/lib/`
- [ ] Add Pagefind build step to Astro build script

### 5.3 Blog Structure
- [ ] Create `src/components/` (reusable UI)
- [ ] Create `src/layouts/` (page layouts)
- [ ] Create `src/lib/` (API fetchers using `@omed/api-client`)
- [ ] Create `src/styles/` (global styles)

### 5.4 Pages Implementation
- [ ] Implement `/` — Home (article list with pagination)
- [ ] Implement `/articles/:slug` — Article detail + comments (client-side fetch)
- [ ] Implement `/tags/:slug` — Articles filtered by tag
- [ ] Implement `/about` — About/profile page
- [ ] Implement `/search` — Pagefind search page

### 5.5 Blog Tests
- [ ] Install Vitest
- [ ] Install vitest-mock-extended
- [ ] Install Playwright (`npx playwright install`)
- [ ] Write unit tests for `src/lib/` functions
- [ ] Write E2E tests:
  - [ ] Home page loads articles
  - [ ] Article detail renders correctly
  - [ ] Search returns relevant results
  - [ ] Tag filter works
- [ ] Run `task test:unit` → all pass
- [ ] Run `task test:e2e` → all pass

---

## Phase 6 — TypeScript Client Generation

### 6.1 Wire Up Client Generation
- [ ] Add `task docs:generate` to Taskfile (swag init → `api/docs/openapi.json`)
- [ ] Add `task client:generate` to Taskfile:
  - [ ] Run `openapi-typescript` → generate `ui/packages/api-client/src/generated/schema.d.ts`
- [ ] Run `task docs:generate` → verify `api/docs/openapi.json` exists
- [ ] Run `task client:generate` → verify `schema.d.ts` generated
- [ ] Verify `apiClient.GET("/articles")` works in Cockpit with full TypeScript types
- [ ] Verify `apiClient.GET("/articles")` works in Astro with full TypeScript types

---

## Phase 7 — CI/CD Pipeline

### 7.1 GitHub Actions — Blog Deploy
- [ ] Create `.github/workflows/deploy-blog.yml`
- [ ] Configure workflow trigger: `repository_dispatch` (for cron-job.org webhook)
- [ ] Add GitHub Secrets:
  - [ ] `API_URL`
  - [ ] `API_KEY`
- [ ] Configure Astro build step (fetch articles from Engine using API_KEY)
- [ ] Configure Pagefind index build step
- [ ] Configure deploy step to `kilip.github.io`
- [ ] Configure `cron-job.org` to hit GitHub webhook weekly

### 7.2 Verify Full Pipeline
- [ ] Manually trigger GitHub Action
- [ ] Verify Astro fetches published articles from Engine
- [ ] Verify Pagefind index built correctly
- [ ] Verify site deployed to `kilip.github.io`
- [ ] Verify search works on deployed site

---

## Phase 8 — Final Taskfile

### 8.1 Verify All Taskfile Commands
- [ ] `task dev` → starts all services
- [ ] `task build` → builds all
- [ ] `task lint` → lints all (zero errors)
- [ ] `task test` → all tests pass
- [ ] `task test:unit` → unit tests pass
- [ ] `task test:e2e` → E2E tests pass
- [ ] `task test:coverage` → coverage report generated
- [ ] `task db:migrate` → EntGo migrations run
- [ ] `task db:generate` → Drizzle generates Better-Auth schema
- [ ] `task docker:up` → all 6 Docker services running
- [ ] `task docs:generate` → `api/docs/openapi.json` generated
- [ ] `task client:generate` → `@omed/api-client` generated
- [ ] `task mock:generate` → Go mocks generated

---

## Phase 9 — QA & Launch Checklist

### 9.1 Security Review
- [ ] Verify CORS only allows configured origins (no wildcard)
- [ ] Verify rate limiting active on all public endpoints
- [ ] Verify tenant isolation — tenant A cannot access tenant B data
- [ ] Verify API Key only returns `published` articles
- [ ] Verify comments require `is_approved = true` to show publicly

### 9.2 Architecture Review
- [ ] Verify NO logic in `app/` folder (Cockpit)
- [ ] Verify ALL `app/page.tsx` files contain exactly 1 re-export line
- [ ] Verify NO `middleware.ts` exists in `ui/dash/` (must be `proxy.ts`)
- [ ] Verify NO `@radix-ui/*` packages installed manually
- [ ] Verify NO direct `process.env` access outside `config.ts`
- [ ] Verify NO manual edits to `src/generated/schema.d.ts`
- [ ] Verify NO manual edits to `domain/mocks/`
- [ ] Verify NO `github.com/golang/mock` imports (must be `go.uber.org/mock`)
- [ ] Verify NO `github.com/gofiber/fiber/v2` imports (must be v3)

### 9.3 Convention Review
- [ ] All TypeScript/Go files use PascalCase naming
- [ ] All git commits follow Conventional Commits format
- [ ] All branches follow `feat/` or `fix/` prefix
- [ ] All API responses follow `{ success, data, meta }` format
- [ ] All error responses follow `{ success, error: { code, message } }` format

### 9.4 Final Smoke Test
- [ ] Engine API accessible at `http://localhost:8080`
- [ ] Scalar UI accessible at `http://localhost:8080/docs`
- [ ] Cockpit accessible at `http://localhost:3000`
- [ ] Blog accessible at `http://localhost:4321`
- [ ] MinIO console accessible at `http://localhost:9001`
- [ ] Full article lifecycle: draft → review → published → visible on blog
- [ ] Comment flow: submit → approve in Cockpit → visible on blog
- [ ] Media upload: upload in Cockpit → MinIO stored → URL in article

---

*End of Checklist*

---

> **Total Phases:** 9
> **Reference Document:** OMED_PRD.md
> Follow PRD strictly. When unsure — re-read the PRD.
