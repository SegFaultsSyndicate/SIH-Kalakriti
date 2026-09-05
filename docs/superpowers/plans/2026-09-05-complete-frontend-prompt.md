# Complete Frontend Prompt Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close every frontend-prompt requirement that can be implemented and verified in this repository, while documenting hardware-only checks with explicit evidence boundaries.

**Architecture:** Keep the three existing SvelteKit applications and shared packages. Repair shared type/build wiring first, then strengthen offline/i18n/accessibility primitives, then add deterministic demo fixtures and authenticated Playwright journeys, and finally enforce CI budgets and coverage. Do not invent API contracts: use the generated OpenAPI types and existing mock/BFF seams.

**Tech Stack:** SvelteKit 2, Svelte 5 runes, TypeScript, pnpm, Dexie, Vitest, Playwright, axe-core, Lighthouse CI, vanilla CSS tokens, Go BFF.

**Spec:** `Pasted text #1.txt` (frontend prompt pack, Batches 1–14)

## Global Constraints

- Preserve the three-app split; admin code must not enter artisan bundles.
- Use generated OpenAPI types for request/response payloads.
- Never base64-encode queued media; store `Blob` values in IndexedDB.
- Never auto-publish listings or silently discard failed work.
- Use logical CSS properties and localized message keys for user-visible strings.
- Treat `INSUFFICIENT_EVIDENCE` as a normal provenance state.
- Do not claim NVDA/VoiceOver, real Android airplane-mode, printed QR, or Go/Svelte pixel parity without external evidence.
- Run the smallest targeted test after each task and the full repository checks before completion.

---

### Task 1: Restore shared type-check and build correctness

**Files:**
- Modify: `web/packages/i18n/src/locale.svelte.ts`
- Modify: `web/packages/ui/src/SpeakButton.svelte`
- Modify: `web/packages/ui/package.json`
- Modify: `web/apps/*/tsconfig.json` only if required by the package-resolution fix
- Test: `web/packages/i18n/src/locales.test.ts`
- Test: `web/packages/voice/src/speak.test.ts` (create if absent)

**Interfaces:**
- Consumes: existing `@kalakriti/voice` package exports and Vite `ImportMetaEnv`.
- Produces: all three app `svelte-check` commands pass without missing-package or duplicate-env errors.

- [ ] **Step 1: Add a failing package-resolution test/check** by running `pnpm --dir web check` and recording the current `ImportMetaEnv.DEV` and `@kalakriti/voice` failures.
- [ ] **Step 2: Remove the duplicate optional `DEV` declaration** and rely on the Vite type declaration already supplied by the app toolchain; keep dev-only missing-key warnings guarded by the existing environment value.
- [ ] **Step 3: Export the voice package through its declared package entrypoint** and add the workspace dependency to `@kalakriti/ui` if it is missing.
- [ ] **Step 4: Run `pnpm --dir web check` and verify zero errors.**

### Task 2: Complete locale catalogue and locale/document behavior

**Files:**
- Modify: `web/packages/i18n/src/locales.ts`
- Modify: `web/packages/i18n/src/locale.svelte.ts`
- Create: `web/packages/i18n/src/messages/fallback.ts`
- Modify: `web/packages/i18n/src/locales.test.ts`
- Modify: `web/packages/i18n/src/format.test.ts`
- Modify: `web/apps/*/src/routes/+layout.svelte`
- Modify: `web/FRONTEND.md`

**Interfaces:**
- Produces: `SUPPORTED_LOCALES` containing all 22 scheduled locale codes, fallback resolution `requested -> hi -> en`, persisted locale preferences, and synchronized `<html lang>`/`dir`.

- [ ] **Step 1: Write tests asserting all 22 locale codes resolve and missing keys fall back through Hindi to English.**
- [ ] **Step 2: Add the 22 catalogue entries using English fallback objects for untranslated languages and a coverage map exposed for the accessibility statement.**
- [ ] **Step 3: Update the locale store to persist selection in Dexie prefs and update document language/direction without reload.**
- [ ] **Step 4: Add Indic-aware `Intl.NumberFormat`, currency, date, and relative-time tests, including `en-IN` grouping.**
- [ ] **Step 5: Run the i18n package tests.**

### Task 3: Finish voice and screen-reading infrastructure

**Files:**
- Modify: `web/packages/voice/src/speak.ts`
- Modify: `web/packages/voice/src/listen.ts`
- Modify: `web/packages/voice/src/screen-reader.ts`
- Modify: `web/packages/voice/src/ReadScreen.svelte`
- Modify: `web/packages/voice/src/commands.ts`
- Modify: `web/packages/voice/src/*.test.ts`
- Modify: `web/apps/artisan/src/routes/+layout.svelte`
- Modify: `web/apps/artisan/src/lib/route-guard.ts` only if route command guards require it

**Interfaces:**
- Produces: interruptible `speak`, ASR listener with partial transcript support where available, semantic read-screen control, and locale command routing.

- [ ] **Step 1: Add tests for utterance cancellation, queue ordering, command normalization, and semantic heading/label/value extraction.**
- [ ] **Step 2: Implement the queue so a new utterance cancels the previous utterance and static message keys use pre-generated audio when configured.**
- [ ] **Step 3: Implement `ReadScreen` against semantic landmarks and form labels rather than raw DOM text order.**
- [ ] **Step 4: Wire persistent read-screen and voice navigation controls into the artisan shell.**
- [ ] **Step 5: Run voice tests and the artisan shell tests.**

### Task 4: Close accessibility shell and route coverage

**Files:**
- Modify: `web/packages/ui/src/AccessibilityControl.svelte`
- Modify: `web/packages/ui/src/SkipLink.svelte`
- Modify: `web/packages/ui/src/VisuallyHidden.svelte`
- Create: `web/packages/ui/src/RouteAnnouncer.svelte`
- Create: `web/packages/ui/src/focus-main-heading.ts`
- Modify: `web/apps/*/src/routes/+layout.svelte`
- Modify: `web/apps/*/src/routes/accessibility/+page.svelte`
- Modify: `web/e2e/tests/*/a11y-sweep.spec.ts`
- Modify: `web/ACCESSIBILITY.md`

**Interfaces:**
- Produces: skip links, route title announcements, main-heading focus after navigation, all-app accessibility controls, and authenticated route sweep coverage using deterministic fixtures.

- [ ] **Step 1: Add unit tests for focus target selection and route-announcer title changes.**
- [ ] **Step 2: Wire the shared components into all three layouts and ensure every page has one `main` landmark and an `h1`.**
- [ ] **Step 3: Add accessibility statement content with conformance target, known external-only checks, and contact route.**
- [ ] **Step 4: Extend axe tests to all public routes and fixture-authenticated routes.**
- [ ] **Step 5: Run targeted axe and component tests.**

### Task 5: Harden offline storage, sync, conflict, and quota behavior

**Files:**
- Modify: `web/packages/offline/src/db.ts`
- Modify: `web/packages/offline/src/outbox.ts`
- Modify: `web/packages/offline/src/drain.ts`
- Modify: `web/packages/offline/src/sync-engine.svelte.ts`
- Modify: `web/packages/offline/src/storage.ts`
- Modify: `web/packages/offline/src/optimistic.ts`
- Modify: `web/packages/offline/src/*.test.ts`
- Modify: `web/packages/offline/src/ConflictBanner.svelte`

**Interfaces:**
- Produces: versioned Dexie tables for drafts/media/outbox/cache/prefs, FIFO dependency-aware retries, visible `needsAttention`, conflict resolution actions, and safe quota eviction.

- [ ] **Step 1: Add fake-indexeddb tests for reload persistence, dependency ordering, idempotency preservation, backoff, terminal failure, and “queued media is never evicted.”**
- [ ] **Step 2: Implement explicit `dependsOn` checks and exponential retry scheduling without dropping records.**
- [ ] **Step 3: Make the sync store expose `synced|syncing|pending|offline|attention`, counts, timestamps, and failures.**
- [ ] **Step 4: Wire online, foreground, timer, manual sync, and BackgroundSync-when-available triggers.**
- [ ] **Step 5: Wire conflict banners and optimistic rollback to the toast system.**
- [ ] **Step 6: Run all offline package tests.**

### Task 6: Complete artisan capture/review acceptance behavior

**Files:**
- Modify: `web/apps/artisan/src/routes/listing/new/capture/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/video/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/processing/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/review/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/story/+page.svelte`
- Modify: `web/apps/artisan/src/lib/listing-draft.ts`
- Modify: `web/apps/artisan/src/lib/ml-mock.ts`
- Test: `web/apps/artisan/src/lib/listing-draft.test.ts`

**Interfaces:**
- Produces: multi-shot reorder/primary selection, actionable quality issues, honest offline processing state, claim-to-attribute highlighting, editable low-confidence attributes, and explicit approval.

- [ ] **Step 1: Add tests for draft resume, primary-photo persistence, claim highlighting, and approval gating.**
- [ ] **Step 2: Add keyboard reorder controls and primary image selection while retaining native camera capture.**
- [ ] **Step 3: Replace fake processing timers with the existing polling/SSE state seam and show “will process when connected” offline.**
- [ ] **Step 4: Render `claims[]`, `needs_artisan_input`, and per-attribute confidence with localized guidance.**
- [ ] **Step 5: Run artisan flow tests and verify no path publishes before approval.**

### Task 7: Complete artisan catalog, pricing, provenance, and orders

**Files:**
- Modify: `web/apps/artisan/src/routes/listings/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listings/[id]/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listings/[id]/provenance/+page.svelte`
- Modify: `web/apps/artisan/src/routes/orders/+page.svelte`
- Modify: `web/apps/artisan/src/routes/orders/[orderId]/+page.svelte`
- Modify: `web/apps/artisan/src/routes/orders/[orderId]/lots/[lotId]/offer/+page.svelte`
- Modify: `web/apps/artisan/src/routes/earnings/+page.svelte`
- Modify: `web/apps/artisan/src/routes/notifications/+page.svelte`
- Modify: `web/packages/ui/src/PriceAdvisory.svelte`
- Create: `web/packages/ui/src/HandloomVerdict.svelte`
- Test: matching route/component tests

**Interfaces:**
- Produces: visible author-vs-model provenance, price-floor-safe advisory, all technique verdict states, explicit offline offer responses, SSE order narratives, income PDF/share actions, and deduplicated notifications.

- [ ] **Step 1: Add focused tests for price-floor clamping, verdict states, and artisan-authored override markers.**
- [ ] **Step 2: Implement component behavior and localized copy without “dynamic pricing.”**
- [ ] **Step 3: Wire SSE reconnect/backfill and offer queue states to the offline engine.**
- [ ] **Step 4: Run artisan package/app tests.**

### Task 8: Complete buyer discovery, listing, provenance, and bulk allocation

**Files:**
- Modify: `web/apps/buyer/src/routes/+page.svelte`
- Modify: `web/apps/buyer/src/routes/search/+page.svelte`
- Modify: `web/apps/buyer/src/routes/listing/[slug]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/bulk-order/+page.svelte`
- Modify: `web/apps/buyer/src/routes/orders/[id]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/verify/[code]/+page.svelte`
- Modify: `web/apps/buyer/src/lib/*` as needed for SSE allocation state
- Test: `web/e2e/tests/buyer/*.spec.ts`

**Interfaces:**
- Produces: removable parsed-filter chips, cross-lingual match explanation, made-to-order treatment, provenance no-JS structure/CSS handoff, and accessible live/table allocation view with dropout narration.

- [ ] **Step 1: Add component tests for search chips, sibling-craft zero results, and made-to-order copy.**
- [ ] **Step 2: Wire allocation SSE reconnect/backfill with deduplication and a table/live-region equivalent.**
- [ ] **Step 3: Ensure JSON-LD and public provenance markup use the shared backend contract.**
- [ ] **Step 4: Run buyer tests and existing QR/provenance checks.**

### Task 9: Complete admin dashboard acceptance criteria

**Files:**
- Modify: `web/apps/admin/src/routes/insights/+page.svelte`
- Modify: `web/apps/admin/src/routes/clusters/+page.svelte`
- Modify: `web/apps/admin/src/routes/moderation/+page.svelte`
- Modify: `web/apps/admin/src/routes/crafts/+page.svelte`
- Modify: `web/apps/admin/src/lib/BarChart.svelte`
- Modify: `web/apps/admin/src/lib/csv.ts`
- Test: admin route/component tests

**Interfaces:**
- Produces: accessible chart tables, visible suppression states, pre-commit CSV validation, SHG share total validation, human-only moderation, ontology alias/merge/index-refresh actions, and command-palette coverage.

- [ ] **Step 1: Add tests for suppression rendering, CSV validation-before-submit, and SHG percentages summing to 100.**
- [ ] **Step 2: Implement table equivalents and keyboard-accessible chart controls.**
- [ ] **Step 3: Implement CSV preview/error report and block commit while invalid rows remain.**
- [ ] **Step 4: Run admin tests and type-check.**

### Task 10: Add deterministic demo fixtures and missing E2E journeys

**Files:**
- Create: `web/e2e/fixtures/demo-state.ts`
- Modify: `web/e2e/playwright.config.ts`
- Modify: `web/e2e/tests/buyer/cross-lingual-search.spec.ts`
- Modify: `web/e2e/tests/buyer/bulk-order-reallocation.spec.ts`
- Modify: `web/e2e/tests/buyer/provenance-qr.spec.ts`
- Create: `web/e2e/tests/artisan/offline-publish.spec.ts`
- Modify: `web/DEMO.md`

**Interfaces:**
- Produces: deterministic browser-local demo state and four stable journeys covering offline publish, cross-lingual search, bulk dropout/reallocation, and provenance verification.

- [ ] **Step 1: Define fixture data matching generated API shapes: nine artisans, 500 units, one dropout/reallocation, Hindi-authored listing, and valid provenance code.**
- [ ] **Step 2: Add Playwright route handlers or seeded local storage at test setup; do not change production API contracts.**
- [ ] **Step 3: Implement each journey with explicit assertions for the prompt acceptance criteria.**
- [ ] **Step 4: Run the four tests against preview builds.**

### Task 11: Enforce performance, accessibility, and visual-regression checks

**Files:**
- Modify: `web/.github/workflows/ci.yml`
- Modify: `web/lighthouserc.json`
- Modify: `web/scripts/check-size.mjs` or the existing size script
- Modify: `web/e2e/tests/artisan/visual.spec.ts`
- Modify: `web/e2e/tests/buyer/visual.spec.ts`
- Modify: `web/e2e/tests/admin/visual.spec.ts`
- Modify: `web/ACCESSIBILITY.md`

**Interfaces:**
- Produces: CI-enforced artisan/buyer gzip budgets, axe sweeps, Lighthouse thresholds, and visual baseline coverage for all three themes and key routes.

- [ ] **Step 1: Make size checks fail on budget regression with explicit per-app limits.**
- [ ] **Step 2: Remove temporary Lighthouse `continue-on-error` only after the configured local/CI command passes.**
- [ ] **Step 3: Extend visual tests to buyer home and admin dashboard and generate platform-consistent baselines in the configured Playwright environment.**
- [ ] **Step 4: Run the focused CI-equivalent commands locally.**

### Task 12: Final documentation and verification

**Files:**
- Modify: `web/FRONTEND.md`
- Modify: `web/DEMO.md`
- Modify: `web/ACCESSIBILITY.md`
- Modify: `web/ASSETS_README.md`
- Modify: `web/README.md`

- [ ] **Step 1: Replace stale gap statements with evidence-backed status and retain explicit external-only checklists.**
- [ ] **Step 2: Run `pnpm --dir web check`, `pnpm --dir web lint`, `pnpm --dir web test`, `pnpm --dir web build`, `pnpm --dir web size`, and targeted Playwright suites.**
- [ ] **Step 3: Record every failure with its exact command and classify it as fixed, pre-existing, environment-blocked, or external-only.**
- [ ] **Step 4: Do not mark the work complete until all repository-verifiable checks pass or are explicitly surfaced as blockers.**

