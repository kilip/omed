# Product Requirements Document (PRD)
# Project OMED — Developer Blog Platform

**Version:** 1.0.0
**Status:** Draft
**Last Updated:** 2026-03-14
**Author:** Anthonius Munthi

---

> ⚠️ **THIS DOCUMENT IS THE SINGLE SOURCE OF TRUTH**
> All AI coding tools (Cursor, Windsurf, Antigravity, Stitch etc.) MUST follow this document exactly.
> Do NOT assume, infer, or hallucinate any implementation detail not explicitly stated here.
> When in doubt — STOP and re-read this document.

---

## Table of Contents

- [1. Project Overview](#1-project-overview)
- [2. Monorepo Structure](#2-monorepo-structure)
- [3. Tech Stack](#3-tech-stack)
  - [3.1 Engine — api/](#31-engine--api)
  - [3.2 Cockpit — ui/dash/](#32-cockpit--uidash)
  - [3.3 Blog — ui/blog/](#33-blog--uiblog)
  - [3.4 Shared Packages](#34-shared-packages)
- [4. Database](#4-database)
  - [4.1 Overview](#41-overview)
  - [4.2 EntGo Tables](#42-entgo-tables)
  - [4.3 Better-Auth Tables](#43-better-auth-tables)
  - [4.4 Redis](#44-redis)
- [5. API Specification](#5-api-specification)
  - [5.1 Base URL & Versioning](#51-base-url--versioning)
  - [5.2 Response Format](#52-response-format)
  - [5.3 Pagination](#53-pagination)
  - [5.4 Endpoints](#54-endpoints)
  - [5.5 API Documentation](#55-api-documentation)
- [6. Authentication & Security](#6-authentication--security)
  - [6.1 Auth Flow](#61-auth-flow)
  - [6.2 Multi-Tenant Strategy](#62-multi-tenant-strategy)
  - [6.3 CORS Configuration](#63-cors-configuration)
  - [6.4 Rate Limiting](#64-rate-limiting)
- [7. Architecture](#7-architecture)
  - [7.1 Engine — Clean Architecture](#71-engine--clean-architecture)
  - [7.2 Cockpit — Feature Sliced Design](#72-cockpit--feature-sliced-design)
  - [7.3 Blog — Astro Convention](#73-blog--astro-convention)
- [8. CI/CD Pipeline](#8-cicd-pipeline)
- [9. Docker Compose](#9-docker-compose)
- [10. Environment Variables](#10-environment-variables)
- [11. UI & Pages](#11-ui--pages)
  - [11.1 Cockpit Pages](#111-cockpit-pages)
  - [11.2 Blog Pages](#112-blog-pages)
- [12. TypeScript Client Generation](#12-typescript-client-generation)
- [13. Testing Strategy](#13-testing-strategy)
  - [13.1 Engine Testing](#131-engine-testing)
  - [13.2 Cockpit Testing](#132-cockpit-testing)
  - [13.3 Blog Testing](#133-blog-testing)
- [14. Taskfile Commands](#14-taskfile-commands)
- [15. Conventions](#15-conventions)
  - [15.1 File Naming](#151-file-naming)
  - [15.2 Git Conventions](#152-git-conventions)
  - [15.3 FSD Rules](#153-fsd-rules)
  - [15.4 shadcn/ui Rules](#154-shadcnui-rules)

---

## 1. Project Overview

**Project Name:** OMED
**Type:** Multi-tenant Developer Blog Platform
**Description:** A personal developer blog platform for writing and sharing coding experiences. Built from scratch with a hypercar-grade architecture. Supports multiple tenants — each tenant gets their own blog deployment.

### Key Characteristics

| Property | Value |
|---|---|
| **Repository** | Single monorepo (1 GitHub repo) |
| **Tenancy** | Multi-tenant, row-level isolation |
| **Blog Deploy** | Static Site Generation via GitHub Actions |
| **Primary Author** | Single super admin (owner) |
| **Tenant Onboarding** | Invite-only by super admin |

---

## 2. Monorepo Structure

```
root/
├── api/                          # Engine — Go REST API
│   ├── domain/
│   ├── usecase/
│   ├── repository/
│   ├── delivery/
│   │   ├── handler/
│   │   └── middleware/
│   ├── infrastructure/
│   └── main.go
├── ui/                           # Turborepo 2.7 workspace root
│   ├── dash/                     # Cockpit — Next.js 16
│   ├── blog/                     # Blog — Astro SSG
│   ├── packages/
│   │   ├── api-client/           # @omed/api-client — generated TS client
│   │   │   ├── src/
│   │   │   │   ├── generated/    # ⛔ DO NOT EDIT — auto-generated
│   │   │   │   │   └── schema.d.ts
│   │   │   │   ├── client.ts
│   │   │   │   └── index.ts
│   │   │   ├── package.json      # name: @omed/api-client
│   │   │   └── tsconfig.json
│   │   ├── util/                 # @omed/util — shared utilities
│   │   │   ├── src/
│   │   │   │   ├── config.ts
│   │   │   │   ├── singleton.ts
│   │   │   │   ├── zod.ts
│   │   │   │   └── index.ts
│   │   │   ├── package.json      # name: @omed/util
│   │   │   └── tsconfig.json
│   │   └── tsconfig/             # @omed/tsconfig — shared TS configs
│   │       ├── base.json
│   │       ├── nextjs.json
│   │       ├── astro.json
│   │       └── package.json      # name: @omed/tsconfig
│   ├── turbo.json                # ⚠️ Turborepo config lives in /ui/ NOT root
│   └── package.json              # Turborepo workspace root
├── Taskfile.yml                  # Root task orchestrator (taskfile.dev)
├── docker-compose.yml
└── .env.example
```

> ⚠️ **CRITICAL:** `turbo.json` lives in `/ui/`, NOT in the project root. Turborepo only manages the JavaScript/TypeScript workspace. The `api/` (Go) is managed independently via Taskfile.

---

## 3. Tech Stack

### 3.1 Engine — `api/`

| Component | Technology | Version |
|---|---|---|
| Language | Go | Latest stable |
| Framework | **Go Fiber** | **v3** (NOT v2) |
| ORM | EntGo | Latest |
| Database | PostgreSQL | 16+ |
| Cache | Redis | 7+ |
| Object Storage | MinIO | Latest |
| API Docs | swaggo/swag | Latest |
| Architecture | Clean Architecture Option 2 | — |

> ⛔ **CRITICAL — Go Fiber v3 Breaking Changes:**
>
> ```go
> // ✅ CORRECT import
> import "github.com/gofiber/fiber/v3"
>
> // ❌ WRONG — v2 is FORBIDDEN
> import "github.com/gofiber/fiber/v2"
> ```
>
> Additional v3 breaking changes:
> - `app.Static()` has been **REMOVED** → use `static.New()` from middleware/static
> - `c.Locals()` behavior changed → use dedicated `FromContext()` functions per middleware

---

### 3.2 Cockpit — `ui/dash/`

| Component | Technology | Version |
|---|---|---|
| Framework | **Next.js** | **16.x** |
| Language | TypeScript | 5+ |
| Architecture | Feature Sliced Design (FSD) | — |
| UI Components | shadcn/ui (Base UI + Mira + Lucide) | Latest |
| Auth | Better-Auth | Latest |
| Auth Session Store | Redis | — |
| Auth Persistence | Drizzle ORM → PostgreSQL | — |
| Server State | TanStack Query | Latest |
| API Client | openapi-react-query + @omed/api-client | — |
| Error Handling | Inline error messages in form/component | — |

> ⛔ **CRITICAL — Next.js 16 Breaking Change:**
>
> `middleware.ts` has been **DEPRECATED** in Next.js 16.
>
> ```
> ╔══════════════════════════════════════════════════════════════╗
> ║                    ⛔ CRITICAL WARNING ⛔                     ║
> ║                  DO NOT IGNORE THIS RULE                     ║
> ╠══════════════════════════════════════════════════════════════╣
> ║                                                              ║
> ║  Next.js 16 has DEPRECATED middleware.ts                     ║
> ║                                                              ║
> ║  ❌ NEVER create middleware.ts — IT WILL NOT WORK            ║
> ║  ❌ NEVER use `export function middleware()`                  ║
> ║  ❌ NEVER reference any Next.js docs below v16               ║
> ║                                                              ║
> ║  ✅ ALWAYS create proxy.ts at project root (ui/dash/proxy.ts)║
> ║  ✅ ALWAYS use `export function proxy()`                     ║
> ║                                                              ║
> ║  CORRECT IMPLEMENTATION:                                     ║
> ║                                                              ║
> ║  // ui/dash/proxy.ts                                         ║
> ║  export function proxy(request: Request) {                   ║
> ║    // auth proxy logic here                                  ║
> ║  }                                                           ║
> ║                                                              ║
> ║  If you create middleware.ts = YOU ARE DOING IT WRONG        ║
> ╚══════════════════════════════════════════════════════════════╝
> ```

---

### 3.3 Blog — `ui/blog/`

| Component | Technology |
|---|---|
| Framework | Astro SSG |
| Search | Pagefind (client-side, zero server) |
| Deploy Target | kilip.github.io |
| Deploy Method | GitHub Actions (triggered by cron-job.org webhook) |
| API Client | openapi-fetch + @omed/api-client |
| Error Handling | Static "Something went wrong" message |

---

### 3.4 Shared Packages

#### `@omed/api-client` — `ui/packages/api-client/`

Generated TypeScript API client. Source of truth: `api/docs/openapi.json`.

```typescript
// ui/packages/api-client/src/client.ts
import createClient from "openapi-fetch";
import type { paths } from "./generated/schema";

export const apiClient = createClient<paths>({
  baseUrl: process.env.NEXT_PUBLIC_API_URL ?? process.env.API_URL,
});
```

```typescript
// ui/packages/api-client/src/index.ts
export { apiClient } from "./client";
export type { paths, components } from "./generated/schema";
```

> ⛔ **CRITICAL — Generated files are READ-ONLY:**
> - `src/generated/schema.d.ts` — **DO NOT EDIT**
> - Always regenerate via `task client:generate`

#### `@omed/util` — `ui/packages/util/`

Shared utilities: config, singleton, zod. Pattern follows Zod-validated env config with singleton instances.

```typescript
// Pattern: always access config via `c` — never access process.env directly
import { c } from "@omed/util/config";

// ✅ CORRECT
const apiUrl = c.api.url;

// ❌ WRONG — never access process.env directly in app code
const apiUrl = process.env.NEXT_PUBLIC_API_URL;
```

#### `@omed/tsconfig` — `ui/packages/tsconfig/`

Shared TypeScript configurations. All apps MUST extend from this package.

```json
// ui/dash/tsconfig.json
{ "extends": "@omed/tsconfig/nextjs.json" }

// ui/blog/tsconfig.json
{ "extends": "@omed/tsconfig/astro.json" }
```

---

## 4. Database

### 4.1 Overview

| Property | Value |
|---|---|
| Database Engine | PostgreSQL 16+ |
| Schema | Single database, single `public` schema |
| ORM — App Tables | **EntGo** (managed by `api/`) |
| ORM — Auth Tables | **Drizzle** (managed by `ui/dash/`, Better-Auth auto-gen) |
| Session Store | **Redis** (Better-Auth session cache) |

> ⛔ **CRITICAL — ORM Boundaries (NEVER violate):**
>
> ```
> EntGo   → ONLY manages: articles, tags, article_tags, comments, media, tenants, tenant_users
> Drizzle → ONLY manages: users, accounts, sessions, verifications (Better-Auth auto-generated)
>
> ❌ EntGo MUST NOT touch Better-Auth tables
> ❌ Drizzle MUST NOT touch app tables
> ❌ NEVER manually edit Better-Auth generated schema
> ```

---

### 4.2 EntGo Tables

#### `tenants`

| Column | Type | Constraints |
|---|---|---|
| id | UUID | PK |
| name | VARCHAR | NOT NULL |
| slug | VARCHAR | UNIQUE, NOT NULL |
| owner_id | UUID | FK → users.id (Better-Auth) |
| created_at | TIMESTAMP | NOT NULL |

#### `tenant_users`

| Column | Type | Constraints |
|---|---|---|
| tenant_id | UUID | FK → tenants.id |
| user_id | UUID | FK → users.id (Better-Auth) |
| joined_at | TIMESTAMP | NOT NULL |

> Primary Key = composite (tenant_id, user_id)

#### `articles`

| Column | Type | Constraints |
|---|---|---|
| id | UUID | PK |
| title | VARCHAR | NOT NULL |
| slug | VARCHAR | UNIQUE, NOT NULL |
| content | TEXT | NOT NULL (Markdown) |
| cover_image | VARCHAR | MinIO URL |
| status | ENUM | draft \| review \| published \| archived |
| reading_time | INT | auto-calculated in minutes |
| meta_description | VARCHAR | SEO |
| meta_og_image | VARCHAR | SEO, MinIO URL |
| tenant_id | UUID | FK → tenants.id, NOT NULL |
| published_at | TIMESTAMP | NULLABLE |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |

#### `tags`

| Column | Type | Constraints |
|---|---|---|
| id | UUID | PK |
| name | VARCHAR | UNIQUE, NOT NULL |
| slug | VARCHAR | UNIQUE, NOT NULL |
| tenant_id | UUID | FK → tenants.id, NOT NULL |
| created_at | TIMESTAMP | NOT NULL |

#### `article_tags`

| Column | Type | Constraints |
|---|---|---|
| article_id | UUID | FK → articles.id |
| tag_id | UUID | FK → tags.id |

> Primary Key = composite (article_id, tag_id)
> ⚠️ No `tenant_id` here — isolation is guaranteed via `article_id → articles.tenant_id`

#### `comments`

| Column | Type | Constraints |
|---|---|---|
| id | UUID | PK |
| article_id | UUID | FK → articles.id, NOT NULL |
| author_name | VARCHAR | NOT NULL (guest commenter) |
| author_email | VARCHAR | NOT NULL (guest commenter) |
| content | TEXT | NOT NULL |
| is_approved | BOOLEAN | DEFAULT false |
| tenant_id | UUID | FK → tenants.id, NOT NULL |
| created_at | TIMESTAMP | NOT NULL |

#### `media`

| Column | Type | Constraints |
|---|---|---|
| id | UUID | PK |
| filename | VARCHAR | NOT NULL (original filename) |
| minio_key | VARCHAR | NOT NULL (path in MinIO bucket) |
| url | VARCHAR | NOT NULL (public URL) |
| mime_type | VARCHAR | NOT NULL |
| size | INT | NOT NULL (bytes) |
| tenant_id | UUID | FK → tenants.id, NOT NULL |
| created_at | TIMESTAMP | NOT NULL |

---

### 4.3 Better-Auth Tables

These tables are **auto-generated** by Better-Auth via Drizzle ORM. Do NOT manually create or modify them.

| Table | Managed By |
|---|---|
| `users` | Better-Auth + Drizzle |
| `accounts` | Better-Auth + Drizzle |
| `sessions` | Better-Auth + Drizzle |
| `verifications` | Better-Auth + Drizzle |

> To generate: run `task db:generate`

---

### 4.4 Redis

| Usage | Key Pattern | TTL |
|---|---|---|
| Better-Auth session cache | `session:{session_id}` | Per Better-Auth config |
| API response cache (Engine) | `cache:{endpoint}:{params}` | Per endpoint config |

---

## 5. API Specification

### 5.1 Base URL & Versioning

```
Development : http://localhost:8080
Production  : https://api.omed.dev (or configured domain)
```

No API versioning in URL for v1. Future versions will use `/v2/` prefix.

---

### 5.2 Response Format

**ALL responses from Go Fiber MUST follow this exact format:**

#### Success Response
```json
{
  "success": true,
  "data": { },
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 42
  }
}
```

> `meta` is only included for paginated list responses.

#### Error Response
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Article not found"
  }
}
```

#### Standard Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `NOT_FOUND` | 404 | Resource not found |
| `UNAUTHORIZED` | 401 | Missing or invalid auth |
| `FORBIDDEN` | 403 | Authenticated but no permission |
| `VALIDATION_ERROR` | 400 | Invalid request body/params |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

> ⛔ **CRITICAL:** Every handler MUST use this response format. No exceptions. No custom response shapes.

---

### 5.3 Pagination

```
Model   : Offset-based
Params  : ?page=1&limit=10
Default : page=1, limit=10
Maximum : limit=50
```

Example: `GET /articles?page=2&limit=10&status=published&tag=golang`

---

### 5.4 Endpoints

#### Articles

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/articles` | API Key | List articles with filter & pagination |
| GET | `/articles/:slug` | API Key | Get single article by slug |
| POST | `/articles` | Redis Session | Create new article |
| PUT | `/articles/:id` | Redis Session | Update article |
| DELETE | `/articles/:id` | Redis Session | Delete article |
| POST | `/articles/:id/publish` | Redis Session | Change article status |

**Query Params for `GET /articles`:**

| Param | Type | Default | Description |
|---|---|---|---|
| page | int | 1 | Page number |
| limit | int | 10 | Items per page (max 50) |
| status | string | — | Filter by status |
| tag | string | — | Filter by tag slug |

**Request Body for `POST /articles`:**
```json
{
  "title": "string",
  "slug": "string",
  "content": "string (markdown)",
  "cover_image": "string (MinIO URL)",
  "tags": ["tag-slug-1", "tag-slug-2"],
  "meta_description": "string",
  "meta_og_image": "string (MinIO URL)"
}
```

**Request Body for `POST /articles/:id/publish`:**
```json
{
  "status": "draft | review | published | archived"
}
```

---

#### Tags

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/tags` | API Key | List all tags for tenant |

---

#### Comments

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/comments` | Public | Submit a comment (guest) |
| GET | `/comments/:article_id` | Public | Get approved comments for article |

**Request Body for `POST /comments`:**
```json
{
  "article_id": "uuid",
  "author_name": "string",
  "author_email": "string",
  "content": "string"
}
```

> ⚠️ Comments require `is_approved = true` before being returned by `GET /comments/:article_id`.
> Approval is done via Cockpit Comment Moderation page.

---

### 5.5 API Documentation

| Property | Value |
|---|---|
| Tool | swaggo/swag |
| Output | `api/docs/openapi.json` |
| UI | Scalar UI served at `/docs` |
| Go Fiber adapter | `github.com/gofiber/swagger` |

> ⛔ **CRITICAL — Annotation Rules:**
>
> ```go
> // ✅ EVERY handler MUST have swaggo annotation
> // ✅ ALWAYS run `task docs:generate` after modifying any handler
> // ✅ Response structs MUST be defined in delivery/handler/response/
> // ❌ NEVER edit openapi.json manually
> // ❌ NEVER skip annotation on new handlers
> ```

Example annotation:
```go
// @Summary      Get all articles
// @Description  Get paginated list of articles filtered by tenant
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        page    query    int     false  "Page number"    default(1)
// @Param        limit   query    int     false  "Limit per page" default(10)
// @Param        status  query    string  false  "Article status"
// @Param        tag     query    string  false  "Filter by tag slug"
// @Success      200  {object}  response.ArticleListResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      429  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /articles [get]
func (h *ArticleHandler) GetArticles(c *fiber.Ctx) error {
```

---

## 6. Authentication & Security

### 6.1 Auth Flow

#### Cockpit (Admin Dashboard)
```
User login at Cockpit
    → Better-Auth handles authentication
        → Session stored in Redis (cache) + PostgreSQL via Drizzle (persist)
            → Go Engine verifies session by checking Redis
                → If Redis miss → fallback to PostgreSQL via Drizzle → re-populate Redis
```

#### Astro Blog (Build Time)
```
GitHub Action triggers build
    → Astro uses API_KEY from GitHub Secrets
        → Go Engine verifies API Key via X-API-Key header
            → Returns only published articles
```

#### Auth Headers

| Context | Header | Value |
|---|---|---|
| Cockpit → Engine | `Authorization` | `Bearer {session_id}` |
| Astro → Engine | `X-API-Key` | `{api_key}` |
| All requests | `X-Tenant-ID` | `{tenant_slug}` |

---

### 6.2 Multi-Tenant Strategy

| Property | Value |
|---|---|
| Model | Row-level isolation via `tenant_id` |
| Onboarding | Invite-only (super admin invites tenants) |
| Data Isolation | Tenant can ONLY CRUD their own data |
| Blog Deployment | Each tenant manages their own GitHub repo & Actions |
| Middleware | All queries auto-filtered by `WHERE tenant_id = ?` |

> ⛔ **CRITICAL:**
> ```
> ✅ EVERY query on tenanted tables MUST include WHERE tenant_id = ?
> ✅ tenant_id MUST be extracted from authenticated session/API key context
> ❌ NEVER query tenanted tables without tenant_id filter
> ❌ NEVER allow tenant A to access tenant B's data
> ```

---

### 6.3 CORS Configuration

```go
app.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "http://localhost:3000",       // Cockpit dev
        "http://localhost:4321",       // Astro dev
        "https://kilip.github.io",    // Astro production
        "https://dash.omed.dev",      // Cockpit production
    },
    AllowMethods: []string{
        "GET", "POST", "PUT", "DELETE", "OPTIONS",
    },
    AllowHeaders: []string{
        "Content-Type",
        "Authorization",
        "X-Tenant-ID",
        "X-API-Key",
    },
    AllowCredentials: true,
}))
```

> ⛔ **CRITICAL:** Do NOT use wildcard `*` for `AllowOrigins`. Always use the explicit list above.

---

### 6.4 Rate Limiting

Rate limiting is implemented using Go Fiber's built-in limiter with **Redis as storage**.

| Endpoint | Strategy | Limit | Reason |
|---|---|---|---|
| `GET /articles` | Per IP | 60 req/min | Normal browsing |
| `GET /articles/:slug` | Per IP | 60 req/min | Normal browsing |
| `GET /tags` | Per IP | 60 req/min | Low risk |
| `POST /comments` | Per IP | **10 req/min** | Anti-spam |
| `POST/PUT/DELETE /*` | Per Tenant ID | 30 req/min | Write operations |
| `POST /articles/:id/publish` | Per Tenant ID | 10 req/min | Extra strict |

Rate limit error response:
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests, please try again later"
  }
}
```

---

## 7. Architecture

### 7.1 Engine — Clean Architecture

The Engine follows **Clean Architecture Option 2** with an explicit middleware layer.

#### Layer Structure

```
domain/           → Pure structs & interfaces. NO external dependencies.
usecase/          → Business logic. Depends on domain interfaces only.
repository/       → Data access via EntGo. Implements domain interfaces.
delivery/
  handler/        → HTTP handlers (Go Fiber). Validate request → call usecase → format response.
  middleware/      → Auth, rate limiting, tenant extraction.
infrastructure/   → External connections (PostgreSQL, Redis, MinIO).
main.go           → Wire everything together.
```

#### Request Flow

```
HTTP Request
    ↓
delivery/middleware/
    → auth_middleware.go     (verify Redis session OR API Key)
    → tenant_middleware.go   (extract tenant_id, inject into context)
    → rate_limit_middleware.go
        ↓
delivery/handler/article_handler.go
    → validate request params
        ↓
usecase/article_usecase.go
    → business logic only (no HTTP, no DB)
        ↓
repository/article_repository.go
    → query PostgreSQL via EntGo
        ↓
domain/article.go
    → pure struct returned
        ↑
    response formatted at handler → { success, data, meta }
        ↓
HTTP Response
```

#### Folder Structure

```
api/
├── domain/
│   ├── Article.go
│   ├── Tag.go
│   ├── Comment.go
│   ├── Media.go
│   ├── Tenant.go
│   └── TenantUser.go
├── usecase/
│   ├── ArticleUsecase.go
│   ├── TagUsecase.go
│   ├── CommentUsecase.go
│   └── MediaUsecase.go
├── repository/
│   ├── ArticleRepository.go
│   ├── TagRepository.go
│   ├── CommentRepository.go
│   └── MediaRepository.go
├── delivery/
│   ├── handler/
│   │   ├── ArticleHandler.go
│   │   ├── TagHandler.go
│   │   ├── CommentHandler.go
│   │   ├── MediaHandler.go
│   │   └── response/            # Response structs for swaggo
│   │       ├── ArticleResponse.go
│   │       └── ErrorResponse.go
│   └── middleware/
│       ├── AuthMiddleware.go
│       ├── TenantMiddleware.go
│       ├── ApiKeyMiddleware.go
│       └── RateLimitMiddleware.go
├── infrastructure/
│   ├── Postgres.go
│   ├── Redis.go
│   └── Minio.go
├── docs/
│   └── openapi.json             # ⛔ DO NOT EDIT — generated by swag
└── main.go
```

> ⛔ **CRITICAL — Dependency Rules:**
> ```
> domain      → ZERO external dependencies
> usecase     → depends on domain interfaces ONLY
> repository  → depends on domain interfaces + EntGo
> handler     → depends on usecase interfaces ONLY
> middleware  → depends on infrastructure (Redis) ONLY
> main.go     → wires all layers together
>
> ❌ NEVER import delivery/ from usecase/
> ❌ NEVER import repository/ from usecase/
> ❌ NEVER import infrastructure/ from domain/
> ```

---

### 7.2 Cockpit — Feature Sliced Design

The Cockpit follows **FSD (Feature Sliced Design)** with **thin re-export, no wrapper** pattern. Reference: https://feature-sliced.design/docs/guides/tech/with-nextjs

#### Folder Structure

```
ui/dash/
├── app/                          # Next.js App Router — ROUTING ONLY
│   ├── (auth)/
│   │   └── login/
│   │       └── page.tsx          # ← thin re-export ONLY
│   ├── (dashboard)/
│   │   ├── page.tsx              # ← thin re-export ONLY
│   │   ├── articles/
│   │   │   ├── page.tsx          # ← thin re-export ONLY
│   │   │   └── [id]/
│   │   │       └── page.tsx      # ← thin re-export ONLY
│   │   ├── tags/
│   │   │   └── page.tsx          # ← thin re-export ONLY
│   │   ├── media/
│   │   │   └── page.tsx          # ← thin re-export ONLY
│   │   └── comments/
│   │       └── page.tsx          # ← thin re-export ONLY
│   └── layout.tsx
├── pages/                        # Empty folder — required by Next.js
│   └── README.md
├── proxy.ts                      # ⚠️ NOT middleware.ts (Next.js 16!)
└── src/
    ├── app/                      # Providers, global config
    ├── pages/                    # FSD pages — actual UI logic lives here
    │   ├── login/
    │   ├── dashboard/
    │   ├── articles/
    │   ├── article-edit/
    │   ├── tags/
    │   ├── media/
    │   └── comments/
    ├── widgets/                  # Composite UI blocks
    ├── features/                 # User interactions
    │   ├── articles/
    │   ├── tags/
    │   ├── comments/
    │   └── media/
    ├── entities/                 # Business entities & types
    │   ├── article/
    │   ├── tag/
    │   ├── comment/
    │   └── media/
    └── shared/
        ├── api/                  # Base fetcher setup
        ├── ui/                   # shadcn thin re-exports
        ├── lib/                  # Utils, constants
        └── types/                # Global TypeScript types
```

#### FSD Rules — Non-Negotiable

```
RULE 1 — app/ folder IS ROUTING ONLY
✅ app/page.tsx contains EXACTLY 1 line:
   export { XxxPage as default } from '@/pages/xxx';
❌ NEVER put JSX in app/ folder
❌ NEVER put hooks in app/ folder
❌ NEVER put logic in app/ folder
❌ NEVER import from anything other than @/pages/* in app/ folder

RULE 2 — Thin Re-export = 1 LINE, NO EXCEPTIONS
✅ index.ts contains export statements ONLY
❌ NEVER wrap logic in index.ts
❌ NEVER add props transformation in index.ts
❌ NEVER create HOC or wrapper in index.ts

RULE 3 — Layer Import Direction (STRICTLY enforced)
✅ pages → widgets → features → entities → shared
❌ NEVER import from a higher layer
❌ NEVER import between slices on the same layer

RULE 4 — shared/ui is shadcn PASS-THROUGH ONLY
✅ export { Button } from 'shadcn/ui/button';
❌ NEVER create <CustomButton> that wraps <Button>
❌ NEVER add default props in shared/ui
```

**Correct `app/` file (ALWAYS this pattern):**
```tsx
// app/(dashboard)/articles/page.tsx
export { ArticlesPage as default } from '@/pages/articles';
```

**Wrong `app/` file (NEVER do this):**
```tsx
// ❌ WRONG — logic in app/ folder
import { useArticles } from '@/features/articles';
export default function ArticlesPage() {
  const { data } = useArticles();
  return <div>{data.map(a => <div>{a.title}</div>)}</div>;
}
```

---

### 7.3 Blog — Astro Convention

```
ui/blog/
└── src/
    ├── components/     # Reusable UI components
    ├── layouts/        # Page layouts
    ├── pages/          # File-based routing
    ├── lib/            # API fetchers, Pagefind config, utils
    └── styles/         # Global styles
```

---

## 8. CI/CD Pipeline

### Publish Flow

```
1. Author writes article in Cockpit
      ↓
2. Cockpit → POST /articles to Engine (status: draft)
      ↓
3. Author reviews → POST /articles/:id/publish (status: published)
      ↓
4. Article sits in PostgreSQL (status: published)
      ↓
5. cron-job.org triggers weekly (configurable)
      ↓
6. cron-job.org hits GitHub Webhook
      ↓
7. GitHub Actions triggered
      ↓
8. Astro fetches GET /articles?status=published from Engine
   (using API_KEY from GitHub Secrets via X-API-Key header)
      ↓
9. Pagefind builds search index
      ↓
10. Astro builds static site
      ↓
11. Deploy to kilip.github.io ✅
```

### GitHub Actions Secrets Required

| Secret | Description |
|---|---|
| `API_URL` | Go Engine base URL |
| `API_KEY` | Static API key for Astro build |

---

## 9. Docker Compose

All services run via `docker-compose.yml` at project root. All data persisted via **bind mount** to `./data/`.

```yaml
services:
  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: omed
      POSTGRES_USER: omed
      POSTGRES_PASSWORD: omed

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - ./data/redis:/data

  minio:
    image: minio/minio
    ports:
      - "9000:9000"   # API
      - "9001:9001"   # Console
    volumes:
      - ./data/minio:/data
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: omed
      MINIO_ROOT_PASSWORD: omed_secret

  api:
    build: ./api
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      - minio

  dash:
    build: ./ui/dash
    ports:
      - "3000:3000"
    depends_on:
      - api

  blog:
    build: ./ui/blog
    ports:
      - "4321:4321"
    depends_on:
      - api
```

### Port Reference

| Service | Port | Notes |
|---|---|---|
| Go API (Engine) | 8080 | REST API + /docs |
| Cockpit (Next.js) | 3000 | Admin dashboard |
| Astro Blog | 4321 | Public blog (dev only) |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache + sessions |
| MinIO API | 9000 | Object storage |
| MinIO Console | 9001 | MinIO web UI |

---

## 10. Environment Variables

### Engine — `api/.env`

```env
APP_PORT=8080

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_NAME=omed
DB_USER=omed
DB_PASSWORD=omed

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=omed
MINIO_SECRET_KEY=omed_secret
MINIO_BUCKET=omed

# Auth
API_KEY_ASTRO=your_static_api_key_here
```

### Cockpit — `ui/dash/.env`

```env
NEXT_PUBLIC_API_URL=http://localhost:8080

# Better-Auth
BETTER_AUTH_SECRET=your_secret_here
BETTER_AUTH_URL=http://localhost:3000

# Redis (for Better-Auth session)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

### Blog — `ui/blog/.env` (also as GitHub Secrets)

```env
API_URL=https://api.omed.dev
API_KEY=your_static_api_key_here
```

> ⛔ **CRITICAL — Config Access Rules:**
> ```
> ✅ ALWAYS access config via `c` from @omed/util/config
> ✅ ALWAYS validate env vars with Zod schema in config.ts
> ❌ NEVER access process.env directly in app code
> ❌ NEVER hardcode any URL, key, or secret in code
> ```

---

## 11. UI & Pages

### 11.1 Cockpit Pages

| Route | FSD Page | Description |
|---|---|---|
| `/login` | `src/pages/login/` | Authentication page |
| `/` | `src/pages/dashboard/` | Stats overview (article count, comment count, etc.) |
| `/articles` | `src/pages/articles/` | Article list with status filter |
| `/articles/new` | `src/pages/article-edit/` | Create new article (Markdown editor) |
| `/articles/:id` | `src/pages/article-edit/` | Edit existing article |
| `/tags` | `src/pages/tags/` | Tag management (CRUD) |
| `/media` | `src/pages/media/` | Media upload & library |
| `/comments` | `src/pages/comments/` | Comment moderation (approve/reject) |

### 11.2 Blog Pages

| Route | Description |
|---|---|
| `/` | Home — list all published articles with pagination |
| `/articles/:slug` | Article detail — full content + comments |
| `/tags/:slug` | Tag filter — articles filtered by tag |
| `/about` | About / profile page |
| `/search` | Pagefind-powered search (client-side, zero server) |

---

## 12. TypeScript Client Generation

### Pipeline

```
api/docs/openapi.json  (generated by swaggo)
         ↓
   task client:generate
         ↓
   openapi-typescript
         ↓
ui/packages/api-client/src/generated/schema.d.ts
         ↓
┌─────────────────────────────────────────────────┐
│  ui/dash (Cockpit)                              │
│  import { apiClient } from "@omed/api-client"   │
│  + openapi-react-query → TanStack Query hooks   │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  ui/blog (Astro)                                │
│  import { apiClient } from "@omed/api-client"   │
│  + openapi-fetch → type-safe fetch              │
└─────────────────────────────────────────────────┘
```

### Packages

```bash
# packages/api-client
npm i -D openapi-typescript

# ui/dash
npm i openapi-fetch openapi-react-query

# ui/blog
npm i openapi-fetch
```

### Usage — Cockpit (TanStack Query)

```typescript
import { apiClient } from "@omed/api-client";
import { useQuery } from "openapi-react-query";

const { data, error } = useQuery(
  apiClient,
  "/articles",
  { params: { query: { page: 1, limit: 10, status: "published" } } }
);
```

### Usage — Astro Blog (openapi-fetch)

```typescript
import { apiClient } from "@omed/api-client";

const { data, error } = await apiClient.GET("/articles", {
  params: { query: { page: 1, limit: 10, status: "published" } }
});
```

> ⛔ **CRITICAL:**
> ```
> ✅ ALWAYS import apiClient from @omed/api-client
> ✅ ALWAYS use generated types from @omed/api-client
> ✅ ALWAYS run task docs:generate → task client:generate after API changes
>
> ❌ NEVER write manual fetch() calls to the Go API
> ❌ NEVER import directly from openapi-fetch in app code
> ❌ NEVER edit src/generated/schema.d.ts
> ❌ NEVER use @hey-api/openapi-ts (WRONG PACKAGE — different project!)
> ```

---

## 13. Testing Strategy

### 13.1 Engine Testing

| Type | Tool | Scope |
|---|---|---|
| Unit Test | Go built-in `testing` + `testify` | Per layer: domain, usecase, repository, handler |
| Mocking | `go.uber.org/mock` + `mockgen` | Auto-generate mocks from domain interfaces |
| Integration Test | `httptest` (Go built-in) | API endpoint testing |

> ⛔ **CRITICAL — Mocking Rules:**
> ```go
> // ✅ CORRECT import
> import "go.uber.org/mock/gomock"
>
> // ❌ WRONG — deprecated, do not use
> import "github.com/golang/mock/gomock"
> ```
>
> ```
> ✅ ALWAYS generate mocks via: task mock:generate
> ✅ Generated mocks live in domain/mocks/ — DO NOT EDIT
> ✅ ALWAYS run task mock:generate after modifying any interface in domain/
> ❌ NEVER write manual mocks
> ❌ NEVER edit generated mock files
> ```

**Mock generation annotation:**
```go
// In domain/ArticleRepository.go
//go:generate mockgen -source=ArticleRepository.go -destination=mocks/MockArticleRepository.go
type ArticleRepository interface {
    FindBySlug(ctx context.Context, slug string) (*Article, error)
    // ...
}
```

---

### 13.2 Cockpit Testing

| Type | Tool | Scope |
|---|---|---|
| Unit Test | Vitest + Testing Library | Components per FSD slice |
| Mocking | vitest-mock-extended | API client, hooks, stores |
| E2E | Playwright | Critical user flows |

**Critical E2E flows to test:**
- Login / logout
- Create article → publish
- Upload media
- Approve comment

---

### 13.3 Blog Testing

| Type | Tool | Scope |
|---|---|---|
| Unit Test | Vitest | `src/lib/` functions (fetchers, utils) |
| Mocking | vitest-mock-extended | API client |
| E2E | Playwright | Pages, search (Pagefind) |

**Critical E2E flows to test:**
- Home page loads articles
- Article detail renders correctly
- Search returns relevant results
- Tag filter works correctly

---

## 14. Taskfile Commands

All commands are defined in `Taskfile.yml` at project root using [taskfile.dev](https://taskfile.dev).

| Command | Description |
|---|---|
| `task dev` | Start all services (api + ui/dash + ui/blog) |
| `task build` | Build all |
| `task lint` | Lint all |
| `task test` | Run all tests |
| `task test:unit` | Run unit tests only |
| `task test:e2e` | Run Playwright E2E tests |
| `task test:coverage` | Generate coverage report |
| `task db:migrate` | Run EntGo migrations |
| `task db:generate` | Generate Better-Auth schema via Drizzle |
| `task docker:up` | Start all Docker services (postgres, redis, minio) |
| `task docs:generate` | Run `swag init` → generate `api/docs/openapi.json` |
| `task client:generate` | Run `openapi-typescript` → generate `@omed/api-client` |
| `task mock:generate` | Run `mockgen` → generate Go mocks in `domain/mocks/` |

> ⚠️ **Recommended workflow after API changes:**
> ```
> task docs:generate → task client:generate
> ```

---

## 15. Conventions

### 15.1 File Naming

| Context | Convention | Example |
|---|---|---|
| TypeScript/TSX files | PascalCase | `ArticleCard.tsx` |
| TypeScript hooks | PascalCase | `UseArticles.ts` |
| Go files | PascalCase | `ArticleHandler.go` |
| CSS/style files | PascalCase | `ArticleCard.module.css` |

```
✅ ArticleCard.tsx
✅ ArticleHandler.go
✅ UseArticles.ts

❌ article-card.tsx
❌ articleCard.tsx
❌ article_handler.go
```

---

### 15.2 Git Conventions

#### Commit Messages — Conventional Commits

```
feat:     new feature
fix:      bug fix
chore:    maintenance, dependencies
docs:     documentation
refactor: code refactoring (no behavior change)
test:     adding or updating tests
ci:       CI/CD changes
```

**Examples:**
```
feat: add article pagination endpoint
fix: comment not saving to postgres
chore: update go fiber to v3.1
docs: add API annotation to ArticleHandler
refactor: extract tenant middleware to separate file
test: add unit test for ArticleUsecase
```

#### Branch Naming

```
feat/{description}    → new feature
fix/{description}     → bug fix

✅ feat/article-list
✅ feat/multi-tenant-support
✅ fix/comment-not-saving
✅ fix/redis-session-timeout

❌ feature/article-list  (use feat/, not feature/)
❌ bugfix/comment        (use fix/, not bugfix/)
❌ my-branch             (not descriptive enough)
```

---

### 15.3 FSD Rules

See [Section 7.2](#72-cockpit--feature-sliced-design) for complete FSD rules. Summary:

```
✅ app/page.tsx = 1 line re-export from src/pages/ ONLY
✅ index.ts = thin re-export, NO logic
✅ shared/ui = shadcn pass-through ONLY
✅ Layer imports: pages → widgets → features → entities → shared

❌ JSX or logic in app/ folder
❌ Wrapper components in shared/ui
❌ Cross-slice imports on same layer
❌ Importing from higher layers
```

---

### 15.4 shadcn/ui Rules

#### Installation — MUST follow this exact process

```bash
# Step 1 — Initialize (ONCE per project)
npx shadcn create
# When prompted:
# → Component library : Base UI       ← NOT Radix
# → Style             : Mira          ← compact, dense UI
# → Icons             : Lucide

# Step 2 — Add components (ALWAYS use CLI)
npx shadcn add button
npx shadcn add input
npx shadcn add table
# etc.
```

> ⛔ **CRITICAL:**
> ```
> ✅ ALWAYS use `npx shadcn add <component>` to add new components
> ✅ Components are copied to src/shared/ui/ — edit there if customization needed
> ✅ shadcn auto-detects Base UI — no manual config needed
>
> ❌ NEVER run `npm install @radix-ui/*` manually
> ❌ NEVER run `npm install @base-ui-components/*` manually
> ❌ NEVER copy-paste component code from shadcn docs
> ❌ NEVER change component library after init
> ❌ NEVER create wrapper components around shadcn components in shared/ui
> ```

---

*End of Document*

---

> **Reminder to AI coding tools:**
> This PRD is the **single source of truth** for Project OMED.
> When you are unsure about any implementation detail — **re-read this document**.
> Do not assume. Do not hallucinate. Do not use patterns from other projects.
> **Every decision is documented here. Follow it exactly.**
