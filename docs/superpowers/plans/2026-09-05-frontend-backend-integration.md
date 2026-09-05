# Frontend–Backend Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect every implemented frontend journey to the Go BFF and verify the backend/frontend contracts end to end.

**Architecture:** Browser code calls only typed operations from `@kalakriti/api`; the BFF translates REST to gRPC service clients. OpenAPI is the contract source, generated TypeScript is committed output, and offline behavior is restricted to explicit safe mutations.

**Tech Stack:** Svelte 5, SvelteKit, TypeScript, Vitest, Go 1.23, chi/gin-compatible HTTP adapters, gRPC, protobuf, OpenAPI, PostgreSQL, Redis, Kafka, MinIO.

**Spec:** `docs/superpowers/specs/2026-09-05-frontend-backend-integration-design.md`

## Global Constraints

- The browser must never call gRPC or backend service ports directly.
- `services/bff/openapi.json` is the browser contract.
- `web/packages/api/src/generated/schema.d.ts` is generated output and must not be hand-edited.
- Every mutating request carries a stable `Idempotency-Key`.
- Do not add silent mock success paths to production runtime.
- Preserve existing user changes and unrelated behavior.
- Write a failing test before each production behavior change.
- External hardware/provider checks remain explicitly unclaimed.

### Task 1: Synchronize the BFF contract and generated browser client

**Files:**
- Modify: `services/bff/openapi.json`
- Modify: `web/packages/api/package.json`
- Modify: `web/packages/api/src/operations.ts`
- Modify: `web/packages/api/src/generated/schema.d.ts` via the existing generator
- Create or modify: contract parity tests near `services/bff/internal/bff/handler`
- Test: `web/packages/api/src/operations.test.ts`

**Interfaces:**
- Produces typed operations for every route used by the apps.
- Preserves `CallOptions`, generated `paths`, and `API_BASE`.

- [ ] Write tests asserting operation URLs, request methods, idempotency headers, and generated response typing for auth refresh, profile, listings, search, orders, follows, statements, insights, clusters, moderation, and provenance.
- [ ] Run the focused API tests and confirm they fail for missing/stale operations.
- [ ] Update the OpenAPI document only where it differs from registered BFF routes; do not invent routes without handlers.
- [ ] Run the existing OpenAPI type generator and commit generated output.
- [ ] Implement only the missing operation wrappers and query serializers.
- [ ] Add a BFF route-parity test that enumerates registered `/api/v1` routes and checks the documented contract.
- [ ] Run API and BFF focused tests.

### Task 2: Complete session refresh and request retry

**Files:**
- Modify: `web/packages/api/src/auth.ts`
- Modify: `web/packages/api/src/session.svelte.ts`
- Modify: `web/packages/api/src/retry.ts`
- Modify: `web/packages/api/src/auth-flow.ts`
- Modify: each app root layout/startup hook
- Test: focused API/session tests

**Interfaces:**
- `restoreAccessToken(): string | null`
- `refreshSession(): Promise<boolean>`
- `call(path, options): Promise<unknown>`

- [ ] Add failing tests for persisted session restore, a single refresh after 401, retry with the original idempotency key, and redirect after refresh failure.
- [ ] Run those tests and verify the expected failures.
- [ ] Implement refresh through `/auth/refresh`, preserving the refresh token and clearing both tokens on failure.
- [ ] Ensure app startup calls restore and configures locale/unauthorized handling.
- [ ] Run the focused tests and all API package tests.

### Task 3: Connect artisan runtime flows

**Files:**
- Modify: `web/apps/artisan/src/routes/login/+page.svelte`
- Modify: `web/apps/artisan/src/routes/register/**/*.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/**/*.svelte`
- Modify: `web/apps/artisan/src/routes/listings/**/*.svelte`
- Modify: `web/apps/artisan/src/routes/orders/**/*.svelte`
- Modify: `web/apps/artisan/src/routes/earnings/+page.svelte`
- Test: artisan route/lib tests

**Interfaces:**
- Uses typed operations from `@kalakriti/api`.
- Uses existing offline draft/outbox APIs for retryable mutations.

- [ ] Add failing tests for registration submission, media confirmation, listing submission, approval, price advisory, and order response.
- [ ] Run focused tests to verify failures.
- [ ] Replace local successful placeholders with operation calls and explicit loading/error states.
- [ ] Preserve draft state when requests fail and queue only supported mutations.
- [ ] Add idempotency keys at mutation boundaries.
- [ ] Run artisan tests and `svelte-check`.

### Task 4: Connect buyer runtime flows

**Files:**
- Modify: `web/apps/buyer/src/routes/search/+page.svelte`
- Modify: `web/apps/buyer/src/routes/listing/[slug]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/artisan/[slug]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/bulk-order/+page.svelte`
- Modify: `web/apps/buyer/src/routes/orders/**/*.svelte`
- Modify: `web/apps/buyer/src/routes/feed/+page.svelte`
- Modify: `web/apps/buyer/src/routes/verify/[code]/+page.svelte`
- Test: buyer route/lib tests

**Interfaces:**
- Uses typed search, listing, storefront, follow, bulk-order, order-event, and verification operations.

- [ ] Add failing tests for search response rendering, follow/unfollow, bulk-order submission, SSE reconnect, and verification response handling.
- [ ] Run focused tests to verify failures.
- [ ] Replace fixture-only data paths with BFF calls and preserve query/filter state.
- [ ] Wire SSE through the shared order-events helper and deduplicate replayed events.
- [ ] Add visible auth/network/validation states.
- [ ] Run buyer tests and `svelte-check`.

### Task 5: Connect admin runtime flows

**Files:**
- Modify: `web/apps/admin/src/routes/clusters/+page.svelte`
- Modify: `web/apps/admin/src/routes/crafts/+page.svelte`
- Modify: `web/apps/admin/src/routes/moderation/+page.svelte`
- Modify: `web/apps/admin/src/routes/insights/+page.svelte`
- Test: admin route/lib tests

**Interfaces:**
- Uses cluster, SHG, moderation, craft-index, and insight operations.
- Enforces MINISTRY/CLUSTER_OFFICER role checks through shared session state.

- [ ] Add failing tests for cluster creation/member onboarding, moderation actions, craft refresh, and insight loading.
- [ ] Run focused tests to verify failures.
- [ ] Replace local mutations with BFF calls and idempotency keys.
- [ ] Add typed error rendering and role-denied states.
- [ ] Run admin tests and `svelte-check`.

### Task 6: Verify and fix directly coupled backend runtime seams

**Files:**
- Modify only directly failing files under `services/bff`, `services/core-svc`, `services/search-svc`, `services/collab-svc`, `services/channel-svc`, `services/insight-svc`, `pkg`, and `proto`.
- Test: existing Go and Python tests; integration-tagged tests when Docker is available.

**Interfaces:**
- Preserve protobuf service names and generated clients.
- Preserve BFF handler interfaces while fixing concrete client wiring.

- [ ] Run `go test ./...` for each existing Go module and record failures.
- [ ] Run the ML service test suite in mock mode and record failures.
- [ ] Fix only errors that block a frontend-used route or violate the backend prompt contract.
- [ ] Add regression tests before each fix.
- [ ] Run migrations/code generation checks where the required tools are available.

### Task 7: End-to-end integration verification and documentation

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `web/README.md` or the existing integration documentation
- Create/modify: deterministic E2E specs under `web/e2e/tests`
- Create/modify: backend integration tests under existing service test locations

**Interfaces:**
- CI runs contract freshness, frontend tests/checks, Go tests, ML tests, and deterministic E2E where services are available.

- [ ] Add failing journey tests for artisan publish, buyer order, admin management, provenance verification, and session refresh.
- [ ] Run them to confirm the missing integration behavior.
- [ ] Implement only the fixtures/adapters needed to exercise the real BFF boundary.
- [ ] Run the complete repository-verifiable suite with explicit heap/resource limits.
- [ ] Document external-only checks and their exact required environments.
- [ ] Mark all integration tasks complete only after the verification output is captured.
