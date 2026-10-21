# TASKS: CMS module (`cms`)

Implementation plan for the `cms` module (Posts, Pages, Media) in `apps/api` and `apps/web`, plus e2e coverage. Written for AI coding agents.

Read first: [AGENTS.md](./AGENTS.md) (single source of truth for conventions), [README.md](./README.md) (architecture), [BRANDING.md](./BRANDING.md) (dashboard design tokens). Do not restate those rules here; follow them.

## Working agreement

1. Pick the first unchecked task whose `Depends` are all checked. Do one task at a time.
2. Check the filesystem before assuming a file, package, or script exists (AGENTS.md section 1). The `fin` module is the reference implementation: when a task says "mirror fin", open the equivalent fin file and copy its structure, naming, error handling, and test style.
3. One task = one logical change = one or more commits following the commit convention (`type(scope): <emoji> subject`). Tick the task's checkbox in the same commit as the final piece of work.
4. A task is done only when its **Done when** list is true and these pass for the parts it touched: `cd apps/api && go test ./...`, `bun run lint`, `bun run typecheck`, `bun run test`.
5. If the repo contradicts a task (missing file, different API, conflicting decision), do not improvise. Stop, add a note under **Blockers** at the bottom, and move to another unblocked task.
6. Do not touch unrelated files. Do not fix unrelated failing tests; list them under **Blockers**.

## Scope

In scope:

- **Posts**: title, slug, excerpt, Markdown body, optional cover image.
- **Pages**: title, slug, Markdown body.
- **Media**: file library backed by the existing `core` file service (S3).
- **Publishing**: `DRAFT`, `SCHEDULED`, `PUBLISHED`, with scheduled publish.
- **Admin API + dashboard UI** (workspace-scoped, authenticated, role-based).
- **Public read API** (unauthenticated, read-only).

Out of scope (do not build): categories/tags, revisions/history, comments, page hierarchy, SEO fields, i18n of content, a rendered public site, webhooks, background workers, Redis/Kafka usage, rate limiting or cache headers for the public API.

## Decisions taken

These are fixed for this plan. If one is wrong, the owner edits this section before work starts.

| # | Decision |
| --- | --- |
| D1 | Three Ent entities: `cms_post`, `cms_page`, `cms_media`. Posts have an optional cover image (`cover_media_id`); pages do not. |
| D2 | `body` is Markdown source stored as text. The API is headless and returns raw Markdown. |
| D3 | Stored status is `DRAFT` or `PUBLISHED`, plus one nullable `publish_at` (the go-live time). `SCHEDULED` is not stored: it is `PUBLISHED` with `publish_at` in the future, computed as `effectiveStatus`. **No worker/scheduler.** See "Visibility rule". |
| D4 | Public API lives under `/public/workspaces/:wid/cms/...`, `GET` only, no auth. Workspaces are addressed by numeric ID (workspaces have no slug). Global CORS only allows `OMED_TRUSTED_ORIGINS`, so browsers on other origins are blocked unless listed; server-side consumers are unaffected. No separate CORS policy in v1. |
| D5 | Media files are stored through `core` `FileService` with `entity_type = "cms:media"`. `cms_media` holds metadata (title, alt text, caption, `is_public`). `is_public` defaults to `true`. |
| D6 | Slug is unique per workspace among non-deleted rows, separately for posts and pages. Auto-generated from the title when omitted; editable. |
| D7 | Roles: `cms:viewer` (read), `cms:editor` (create, update, delete, publish), `cms:admin` (everything). "Publish" is covered by the `update` action. |
| D8 | Media limits use new config: `OMED_CMS_MEDIA_MAX_SIZE_MB` (default `10`) and `OMED_CMS_MEDIA_ALLOWED_TYPES` (default `.png,.jpg,.jpeg,.webp,.gif,.pdf`). SVG is excluded on purpose (script injection risk). |
| D9 | Commit scopes: use `api`, `web`, `e2e`, `docs`. Do **not** add a `cms` scope to `commitlint.config.ts`. |
| D10 | Web Markdown editor: Ant Design `Input.TextArea` with a Write/Preview tab. Preview uses `react-markdown` (new dependency, no raw HTML rendering). |

### Visibility rule (D3)

A post/page is **publicly visible** when it is not soft-deleted, `status = PUBLISHED`, and `publish_at <= now()`.

`effectiveStatus` (returned by the admin API):

- `DRAFT` when `status = DRAFT`.
- `SCHEDULED` when `status = PUBLISHED` and `publish_at > now()`.
- `PUBLISHED` when `status = PUBLISHED` and `publish_at <= now()`.

`publish_at` is required whenever `status = PUBLISHED` and is `NULL` for drafts.

State transitions (service layer, evaluated on `effectiveStatus`):

- `publish`: from `DRAFT` or `SCHEDULED` -> `status = PUBLISHED`, `publish_at = now`.
- `schedule(publishAt)`: from `DRAFT` or `SCHEDULED` -> `status = PUBLISHED`, `publish_at = publishAt`; `publishAt` must be in the future (else `422`).
- `unpublish`: from `PUBLISHED` or `SCHEDULED` -> `status = DRAFT`, `publish_at = NULL`.
- Anything else (for example `publish` or `schedule` on an already live item) returns the module's invalid-transition domain error.

Public ordering is `publish_at DESC, id DESC`. The public list cursor is composite, `"<publish_at unix micros>_<id>"`, with the predicate `publish_at < t OR (publish_at = t AND id < id)`. A malformed cursor returns `base.ErrInvalidCursor`. Admin search cursors stay ID-based, like `fin` (`id DESC`, `limit + 1`).

### Tenant isolation (read before touching the API)

Workspace filtering in this codebase is **not automatic per entity**:

- `internal/core/mutator.go` filters queries with a hard-coded `switch q.Type()` over entity types. Entities not listed there are **not workspace-filtered**. New cms entities must be added (T1.2).
- The same filter is **fail-open**: when no workspace ID is in the context it returns without filtering. Authenticated routes get the workspace ID from `az.Require` (`platform/authz/context.go`). Public routes have no such middleware, so public handlers must parse `:wid`, put it in the context with `base.WithWorkspaceID`, **and** repositories must add an explicit `workspace_id` predicate on every public query (defense in depth).
- The audit mutator calls `base.UserFromContext`, which assumes an authenticated user. Public code paths must be read-only.

## API surface (target)

Admin (authenticated, per `fin` conventions: `POST .../search` registered before `/:id`):

```
/workspaces/:wid/cms/posts        search, create, get, update, delete, :id/publish, :id/schedule, :id/unpublish
/workspaces/:wid/cms/pages        same as posts
/workspaces/:wid/cms/media        upload (multipart), search, get, update metadata, delete
```

Public (no auth):

```
GET /public/workspaces/:wid/cms/posts            list (limit, cursor)
GET /public/workspaces/:wid/cms/posts/:slug
GET /public/workspaces/:wid/cms/pages            list (limit, cursor)
GET /public/workspaces/:wid/cms/pages/:slug
GET /public/workspaces/:wid/cms/media/:id        302 to a presigned URL (only if is_public and not deleted)
```

Rules for the public API:

- Public DTOs are separate from admin DTOs. They never expose status, audit fields, `createdBy`/`updatedBy`, or workspace internals.
- Anything not visible returns `404`, never `403`. An unknown workspace returns an empty list / `404`, never a different error that leaks existence.
- Cover image in a public post is `{ id, alt, url }` where `url` is the stable public media URL (`OMED_API_URL` + path), not a presigned URL.

Permission objects: `cms:post`, `cms:page`, `cms:media` with actions `read`, `create`, `update`, `delete`.

---

## Phase 0: Baseline

- [x] **T0.1 Verify baseline**
  - Depends: none
  - Do: `bun install`; run `cd apps/api && go test ./...`, `bun run lint`, `bun run typecheck`, `bun run test`. Record any pre-existing failures under **Blockers**. No code changes.
  - Done when: baseline results are known.

## Phase 1: API foundation

- [x] **T1.1 Ent schemas**
  - Depends: T0.1
  - Files: `apps/api/ent/schema/mix_publish.go`, `cms_media.go`, `cms_post.go`, `cms_page.go` (naming mirrors `fin_*.go` / `core_*.go`).
  - Do:
    - `PublishMixin`: `status` enum (`DRAFT`,`PUBLISHED`, default `DRAFT`), `publish_at` (optional, nillable time).
    - `cms_media`: `file_id` (edge to `File`, unique), `title`, `alt_text`, `caption` (optional strings), `is_public` (bool, default `true`).
    - `cms_post`: `title` (not empty), `slug` (not empty), `excerpt` (optional), `body` (`field.Text`, default empty), `cover_media_id` (optional nillable, edge to media), plus `PublishMixin`.
    - `cms_page`: `title`, `slug`, `body`, plus `PublishMixin`.
    - All entities use `entsql.Schema("omed")` and `WorkspaceMixin`, `AuditMixin`, `SoftDeleteMixin`, like `fin_period.go`.
    - Indexes: unique `(workspace_id, slug)` with `entsql.IndexWhere("deleted_at IS NULL")` on posts and pages; `(workspace_id, status, publish_at)`; `(workspace_id, deleted_at)`.
    - Declare `id` exactly like existing schemas (`field.Int64("id")`); do not invent another ID scheme.
    - Regenerate Ent with `go generate ./ent/...` from `apps/api` (see `ent/generate.go`). Commit generated code.
  - Done when: `go build ./...` passes; migration runs through `client.Schema.Create` in an integration test boot; rows can be created.
  - Commit: `feat(api): ✨ add cms ent schemas`

- [x] **T1.2 Module skeleton, authz, wiring**
  - Depends: T1.1
  - Files: `apps/api/internal/cms/{module.go,routes.go}`, `internal/cms/domain/{authz.go,error.go}`, edit `internal/config/app.go`.
  - Do:
    - Mirror `internal/fin/module.go` and `routes.go`: `Config` (FiberApp, Logger, EntCli, FileService, ApiURL, media limits), `CmsModule`, `New`, `SetAuthorizer`, `PreBoot`, `Boot`.
    - `domain/authz.go`: `ObjPost`, `ObjPage`, `ObjMedia` constants and `authz.AddRole` in `init()` for `cms:viewer`, `cms:editor` (inherits viewer), `cms:admin` (inherits editor, `cms:*` wildcard). Do **not** grant `core:file` permissions: media goes through cms endpoints, and `core:file write` would let editors delete any workspace file through the core file endpoints. Only add `core:file read` if the web ends up calling a core file endpoint.
    - `domain/error.go`: mirror `internal/fin/domain/error.go` (errors registered with `base.AddToErrorMap` and an HTTP status). Needed: slug taken, invalid transition, publish time in the past, media in use, media type/size not allowed.
    - **Tenant isolation:** add `ent.TypeCmsPost`, `ent.TypeCmsPage`, `ent.TypeCmsMedia` (use the actual generated constant names) to the `switch` in `workspaceInterceptor.filter` in `internal/core/mutator.go`, and extend `mutator_test.go` so each new type is verified to be filtered by workspace.
    - Register in `config/app.go`: construct after `finMod`, pass `coreMod.FileService`, call `SetAuthorizer`, append to `modules`.
    - Routes group is empty for now; `Boot` must succeed.
  - Done when: app boots; unit test asserts the three roles exist with the expected inheritance (see `internal/platform/authz/registry.go` for the lookup API); `mutator_test.go` proves queries on the three cms entity types are workspace-filtered.
  - Commit: `feat(api): ✨ add cms module skeleton and roles`

- [x] **T1.3 Media config**
  - Depends: T1.2
  - Do: add `CMSMediaMaxSizeMB` and `CMSMediaAllowedTypes` to `internal/config/config.go`, mirroring `AttachmentMaxSizeMB`/`AttachmentAllowedTypes` (env tags, default fallbacks, tests in `config_test.go`). Add both vars to `apps/api/.env.example` and the README environment table. Pass them into `cms.Config` in `app.go`.
  - Done when: defaults apply when unset; config tests cover env override and fallback.
  - Commit: `feat(api): ✨ add cms media config`

## Phase 2: Media (API)

- [x] **T2.1 Media model, repository, service**
  - Depends: T1.3
  - Files: `internal/cms/{model,repository,service}/media*.go` plus `*_test.go` next to each.
  - Do:
    - DTOs: upload request, update request (`title`, `altText`, `caption`, `isPublic`), search request (`query`, `mimeType`, `cursor`, `limit` 1..100, same shape as `fin` `SearchTransactionRequest`), response (includes file name, mime type, size, `isPublic`, `url` presigned for admin use, `auditMeta`).
    - Repository: only place that touches Ent. Create, get, search (cursor pagination like `fin`), update, soft delete, `IsReferenced(ctx, id)` (non-deleted post with `cover_media_id = id`).
    - Service: define a narrow `FileServiceContract` in `cms/service` (Upload, GetByID, PresignDownload, Delete), as `fin/service` does. Upload validates size and extension against D8 config, calls the file service with `entity_type = "cms:media"` (the file service already ensures the workspace bucket and cleans up the S3 object if its own insert fails), then creates the `cms_media` row and links it back by setting the file's `entity_id` to the media ID through the file service `Update`; if the media row insert fails, delete the uploaded file through the file service. Delete returns the "media in use" error when referenced, otherwise deletes metadata then the file.
    - Generate mocks with `go:generate` + mockgen like existing services.
  - Done when: unit tests cover happy path, disallowed type, oversize, compensation on insert failure, delete-when-referenced.
  - Commit: `feat(api): ✨ add cms media service`

- [x] **T2.2 Media controller, routes, integration tests**
  - Depends: T2.1
  - Files: `internal/cms/controller/media.go`, `internal/cms/routes.go`, `internal/cms/test/media_test.go`.
  - Do: implement the media admin endpoints from "API surface" with `az.Require(domain.ObjMedia, ...)`. Upload is multipart (`file`, optional `title`, `altText`, `caption`); mirror `core/controller/file.go` and `fin` `UploadAttachment`. Add Swaggo annotations (`@Tags CMS Media`, `@Security BearerAuth`, error responses).
  - Done when: integration tests (use `testutil.ApiTestSuite`) cover upload, search, get, update, delete, `409` when in use, `403` for a member without role, `401` unauthenticated, cross-workspace isolation.
  - Commit: `feat(api): ✨ add cms media endpoints`

## Phase 3: Publishing core + Posts (API)

- [x] **T3.1 Publishing and slug helpers**
  - Depends: T1.2
  - Files: `internal/cms/domain/{publish.go,slug.go}` plus tests.
  - Do: pure functions, no DB. `Status` constants (`DRAFT`, `PUBLISHED`) and effective-status constants (`DRAFT`, `SCHEDULED`, `PUBLISHED`); `EffectiveStatus(status, publishAt, now)`; `Publish`, `Schedule`, `Unpublish` transition functions implementing the transition table above (they return the new `status` and `publish_at`, or a domain error); `Slugify(title)` (lowercase, ASCII-safe, hyphen-separated, fallback for empty result) and a unique-suffix helper.
  - Done when: table-driven tests cover every transition (valid and invalid), the `publish_at == now` boundary, empty/unicode/duplicate-hyphen slugs.
  - Commit: `feat(api): ✨ add cms publishing and slug helpers`

- [x] **T3.2 Post model and repository**
  - Depends: T1.1, T3.1
  - Files: `internal/cms/{model,repository}/post*.go` plus tests.
  - Do:
    - DTOs: create/update (`title`, `slug`, `excerpt`, `body`, `coverMediaId`), schedule request (`publishAt`), search request (`query` over title/slug, `status` filter applied on **effective** status, `cursor`, `limit`), admin response (all fields, `status`, `effectiveStatus`, `coverImage`, `auditMeta`).
    - Repository: CRUD, search, slug-exists check, a reusable `visible(now)` Ent predicate implementing the visibility rule, and public list/get-by-slug methods that take the workspace ID explicitly and add a `workspace_id` predicate themselves (see "Tenant isolation"), ordered `publish_at DESC, id DESC` with the composite cursor. Map unique-slug violations to the "slug taken" error.
  - Done when: repository tests cover search filters, effective-status filtering, the visibility predicate (draft, scheduled future, `publish_at` in the past, `publish_at == now`, deleted), admin ID cursor pagination, public composite cursor pagination (including items with equal `publish_at`), slug uniqueness (including reuse after soft delete).
  - Commit: `feat(api): ✨ add cms post repository`

- [x] **T3.3 Post service**
  - Depends: T3.2, T2.1
  - Files: `internal/cms/service/post*.go` plus tests and mocks.
  - Do: create (auto slug via helpers, resolve conflicts with suffix), get, search, update, delete, `Publish`, `Schedule`, `Unpublish`. Inject a clock (`now func() time.Time` option, same style as `With...` options in `fin`) so tests control time. Validate that `coverMediaId` belongs to the same workspace and is not deleted. Cross-module access only through contracts, never other modules' repositories.
  - Done when: unit tests cover each operation, invalid transitions, schedule in the past, bad cover media, slug conflict handling.
  - Commit: `feat(api): ✨ add cms post service`

- [x] **T3.4 Post controller, routes, integration tests**
  - Depends: T3.3, T2.2
  - Files: `internal/cms/controller/post.go`, `internal/cms/routes.go`, `internal/cms/test/post_test.go`.
  - Do: admin endpoints from "API surface" with `az.Require(domain.ObjPost, ...)`; publish/schedule/unpublish require `update`. Swaggo annotations (`@Tags CMS Posts`).
  - Done when: integration tests cover CRUD, search, publish/schedule/unpublish flows, validation errors (`422`), slug conflict (`409`), permission matrix (viewer read-only, editor can write, no role gets `403`), unauthenticated `401`, cross-workspace isolation.
  - Commit: `feat(api): ✨ add cms post endpoints`

## Phase 4: Pages (API)

- [x] **T4.1 Page model, repository, service**
  - Depends: T3.3
  - Do: same as T3.2 + T3.3 for pages (no excerpt, no cover). Reuse `domain` helpers; do not copy-paste logic that can be shared inside `internal/cms`. Unit tests next to each file.
  - Done when: tests mirror the post test coverage.
  - Commit: `feat(api): ✨ add cms page service`

- [x] **T4.2 Page controller, routes, integration tests**
  - Depends: T4.1, T3.4
  - Do: same as T3.4 for pages with `domain.ObjPage` (`@Tags CMS Pages`), tests in `internal/cms/test/page_test.go`.
  - Done when: integration tests mirror the post integration coverage.
  - Commit: `feat(api): ✨ add cms page endpoints`

## Phase 5: Public API

- [x] **T5.1 Auth middleware bypass for `/public/`**
  - Depends: T1.2
  - Files: `internal/core/middleware/auth.go` and its test.
  - Do: let `GET` and `HEAD` requests whose path starts with `/public/` skip authentication, like the existing bypasses. Update the function doc comment. Other methods on `/public/` still require auth.
  - Done when: middleware tests cover `GET /public/x` without token (passes), `POST /public/x` without token (`401`), and that existing bypasses still behave the same.
  - Commit: `feat(auth): ✨ allow unauthenticated GET on /public routes`

- [x] **T5.2 Public posts and pages**
  - Depends: T5.1, T4.2
  - Files: `internal/cms/model/public*.go`, `internal/cms/controller/public.go`, `internal/cms/test/public_test.go`; service/repository methods as needed.
  - Do: implement the four content endpoints from "API surface" using the `visible(now)` predicate. Public DTOs per the public API rules. No `az.Require`: each handler parses `:wid`, sets it in the context with `base.WithWorkspaceID`, and passes it explicitly to the repository (see "Tenant isolation"). Never call anything that mutates. Swaggo annotations (`@Tags CMS Public`, no `@Security`).
  - Done when: integration tests, all **unauthenticated**, cover: published visible; draft `404`; scheduled-in-future `404`; scheduled-in-past `200` (set `publish_at` directly through the Ent client in the test, since the API refuses past schedules); soft-deleted `404`; another workspace's slug `404` (same slug exists in two workspaces and each only sees its own); unknown workspace; an invalid or zero `:wid`; pagination and ordering; response JSON contains no status/audit fields; cover image URL is the stable public media URL.
  - Commit: `feat(api): ✨ add cms public content endpoints`

- [x] **T5.3 Public media redirect**
  - Depends: T5.2, T2.2
  - Do: `GET /public/workspaces/:wid/cms/media/:id` (set the workspace in the context first; the core file service looks files up through the context filter) returns `302` to a presigned URL when the media exists in that workspace, is not deleted, and `is_public = true`; otherwise `404`.
  - Done when: integration tests cover public, non-public, deleted, wrong workspace.
  - Commit: `feat(api): ✨ add cms public media endpoint`

## Phase 6: OpenAPI

- [x] **T6.1 Regenerate spec and types**
  - Depends: T5.3
  - Do: `bunx turbo run oa:gen`, then `bunx openapi-typescript apps/api/docs/swagger.json -o packages/openapi/src/api.ts`. Commit `apps/api/docs/*` and `packages/openapi/src/api.ts`. Confirm CMS admin and public operations appear in the spec.
  - Done when: `bun run typecheck` passes; spec contains all endpoints from "API surface".
  - Commit: `chore(api): 🔧 regenerate openapi for cms`

## Phase 7: Web

General rules for every web task: types come from `@omed/openapi` via `openapi-fetch` (no hand-written response types); style with the Ant Design theme and design tokens from BRANDING.md (no hardcoded hex); all user-facing strings in both `i18n/en.ts` and `i18n/id.ts`; unit tests (Vitest) next to the file under test; follow `apps/web/.agents/skills/react-router`; mirror `modules/fin` structure and patterns.

- [x] **T7.1 Module scaffolding**
  - Depends: T6.1
  - Files: `apps/web/app/modules/cms/{module.ts,route.ts,queries.ts,hooks/}`, i18n files.
  - Do:
    - `module.ts` via `defineModule`: `id: "cms"`, label/description i18n keys, an Ant Design icon, `order: 40` (fin is 30), menu entries for posts, pages, media.
    - `route.ts`: `prefix("cms", [...])`; the index route redirects to `posts`. Route files may initially be minimal placeholders that later tasks fill in. Do not edit any central registry (there is none; discovery is by glob).
    - `queries.ts`: query keys and hooks mirroring `modules/fin/queries.ts`.
    - `hooks/use-cms-permissions.ts` mirroring `use-fin-permissions.ts` (`cms:viewer|editor|admin`), with tests.
    - i18n: `cms.label`, `cms.description`, `cms.menu.*`, and role labels (`role.cms.viewer`, `role.cms.editor`, `role.cms.admin`; see `lib/roles.ts`).
  - Done when: the launcher shows the CMS module and the sidebar menu renders; permission hook tests pass.
  - Commit: `feat(web): ✨ add cms module scaffolding`

- [x] **T7.2 Media library and picker**
  - Depends: T7.1
  - Files: `modules/cms/routes/media.tsx`, `modules/cms/components/{media-picker.tsx,...}` plus tests.
  - Do: grid/list of media with search and cursor pagination; upload via Ant Design `Upload` (client-side check of size/type matches D8 limits, server remains authoritative); edit drawer for title, alt text, caption, `isPublic`; delete with confirm and a clear message on `409` (in use); "copy public URL" action shown only when `isPublic`; a reusable `MediaPicker` modal (select one media) used later for cover images. Gate actions with `use-cms-permissions`.
  - Done when: component tests cover upload validation, picker selection, permission gating, `409` message (mirror `transaction-attachments.test.tsx`).
  - Commit: `feat(web): ✨ add cms media library`

- [x] **T7.3 Publishing components**
  - Depends: T7.1
  - Files: `modules/cms/components/{status-tag.tsx,publish-panel.tsx}` plus tests.
  - Do: `StatusTag` renders `effectiveStatus` (Draft, Scheduled, Published) with theme tokens. `PublishPanel` takes the stored status/`publishAt` and callbacks, offers Publish now, Schedule (Ant Design `DatePicker` with time, disallow past), Unpublish; shows only transitions valid for the current state; disabled without permission.
  - Done when: tests cover each state's available actions and past-date blocking.
  - Commit: `feat(web): ✨ add cms publish panel`

- [x] **T7.4 Posts UI**
  - Depends: T7.2, T7.3
  - Files: `modules/cms/routes/{posts.tsx,post-new.tsx,post.tsx}`, `modules/cms/components/markdown-editor.tsx` plus tests; add `react-markdown` to `apps/web/package.json` (D10).
  - Do: posts table (search, effective-status filter, cursor pagination, status tags, row actions); create and edit forms (title, slug with auto-fill hint, excerpt, Markdown editor with Write/Preview tabs, cover image via `MediaPicker`); `PublishPanel` on the edit screen; delete with confirm. Register routes in `route.ts`: `posts`, `posts/new`, `posts/:id`. Handle API errors (slug conflict, validation) inline.
  - Done when: route/component tests cover create, edit, publish, schedule, delete flows with mocked API and permission gating; preview does not render raw HTML.
  - Commit: `feat(web): ✨ add cms posts screens`

- [x] **T7.5 Pages UI**
  - Depends: T7.4
  - Files: `modules/cms/routes/{pages.tsx,page-new.tsx,page.tsx}`.
  - Do: same as T7.4 without excerpt and cover; reuse `markdown-editor`, `status-tag`, `publish-panel`. Routes: `pages`, `pages/new`, `pages/:id`.
  - Done when: tests mirror T7.4 coverage.
  - Commit: `feat(web): ✨ add cms pages screens`

## Phase 8: E2E

- [x] **T8.1 Page objects and support**
  - Depends: T7.5
  - Files: `e2e/pages/cms/{posts,pages,media}.page.ts`, `e2e/pages/index.ts`, `e2e/support/test-client.ts` if needed.
  - Do: Page Object Model per AGENTS.md section 5 (locators and `expect*` in page objects only). Register a `cms` group in `createPages()`. Role assignment already exists in `support/test-client.ts` (`addMember({ workspaceId, userId, role })`, roles are plain strings such as `cms:viewer`); see `features/fin/permissions.feature` for usage. Add test-client helpers only for creating CMS content through the API if the UI path is too slow for setup.
  - Done when: `bun run typecheck` passes; no selectors outside page objects.
  - Commit: `test(e2e): ✅ add cms page objects`

- [x] **T8.2 Features and steps**
  - Depends: T8.1
  - Files: `e2e/features/cms/{posts,pages,media,publishing,permissions,public-api}.feature`, `e2e/steps/cms/*.steps.ts`.
  - Do: tag scenarios `@authenticated`. Cover: create/edit/delete post and page; publish, unpublish, schedule (future date) with correct status tag; media upload, edit alt text, delete blocked while used as cover; viewer is read-only and editor can edit; public API checks through the test client without a session (published visible, draft hidden, future-scheduled hidden). The scheduled-becomes-visible transition is covered by Go integration tests (T5.2), not e2e, to avoid time-dependent flakiness. Put steps in `steps/common/` only if another module also uses them.
  - Done when: `bunx playwright install chromium` done once, `bun run e2e` passes locally.
  - Commit: `test(e2e): ✅ cover cms module`

## Phase 9: Docs

- [x] **T9.1 README**
  - Depends: T8.2
  - Do: add a CMS section to README.md in the same style as the Finance routes list (web routes and capabilities, admin and public API summary, new env vars if not already listed). Do not change AGENTS.md or BRANDING.md.
  - Done when: README matches the implemented routes and env vars.
  - Commit: `docs(docs): 📝 document cms module`

---

## Definition of done (whole module)

- All tasks above are checked.
- `cd apps/api && go test ./...`, `bun run lint`, `bun run typecheck`, `bun run test`, `bun run e2e` all pass.
- `apps/api/docs/*` and `packages/openapi/src/api.ts` are in sync with the controllers.
- No controller touches Ent or repositories; no module reaches another module's repository.
- No hardcoded hex colors, no hand-written API response types in `apps/web`.
- Public API leaks nothing that is draft, future-scheduled, deleted, non-public media, or from another workspace.
- All three cms entity types are covered by the workspace filter in `internal/core/mutator.go`.

## Blockers

_Agents: record anything that blocked you here (task ID, what you found, what you need)._

