# Wire Remaining Kalakriti Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the delivered identity, illustration, ornament, print, documentation, and provenance assets reachable from real product flows and add truthful automated verification for the remaining hardware/browser-dependent checks.

**Architecture:** Reuse the existing asset packages and Svelte wrappers rather than duplicating SVGs. Add a shared identity/asset showcase surface for assets that are not appropriate as persistent chrome, wire identity and process assets into existing app shells and artisan flows, and make the provenance page consume the same visual assets as the standalone Go stylesheet. Add deterministic structural checks for QR quiet-zone geometry and provenance asset references; document real-device scan and pixel-parity checks as manual acceptance steps.

**Tech Stack:** SvelteKit, Svelte 5, CSS custom properties, existing `@kalakriti/*` workspace packages, Vitest, Playwright, Node generator/validation scripts.

**Spec:** User request to wire the previously unused assets into runtime flows and address the documented verification gaps.

## Global Constraints

- Preserve the existing asset-pack rules: no gradients, no unrelated raw colors, and use the existing token/package interfaces.
- Do not draw or embed the protected State Emblem of India; keep the existing placeholder slot.
- Do not fake QR scan or pixel-parity results; automate what is deterministic and clearly label manual checks.
- Keep user-visible copy in `@kalakriti/i18n`.
- Run only existing package checks, targeted tests, and the existing asset validation scripts.

---

### Task 1: Wire identity and asset showcase surfaces

**Files:**
- Create: `web/packages/identity/IdentityShowcase.svelte`
- Create: `web/apps/buyer/src/routes/assets/+page.svelte`
- Create: `web/apps/buyer/src/routes/assets/+page.css`
- Modify: `web/packages/identity/index.js`
- Modify: `web/apps/buyer/src/routes/+layout.svelte`
- Modify: `web/packages/i18n/src/messages/en.ts`
- Modify: `web/packages/i18n/src/messages/hi.ts`
- Test: `web/e2e/tests/buyer/visual.spec.ts`

**Interfaces:**
- `IdentityShowcase` accepts `mode?: 'header' | 'gallery'`, `title?: string`, and `class?: string`; it renders the logomark, horizontal wordmark, heritage crest, and seal using package imports.
- The buyer `/assets` route is a real searchable asset catalogue entry point, not a build-only contact sheet; it links to the existing generated documentation and batch contact sheets.

- [ ] **Step 1: Add failing route coverage** for `/assets`, checking the wordmark, crest, seal, and a link to the asset documentation.
- [ ] **Step 2: Run the focused Playwright test and confirm it fails** because the route and component are absent.
- [ ] **Step 3: Implement the identity showcase** using existing SVG components/assets, theme-aware token styles, and accessible labels.
- [ ] **Step 4: Add the route and navigation entry** without replacing the existing application header.
- [ ] **Step 5: Run the focused Playwright test and the asset validator** and confirm both pass.
- [ ] **Step 6: Commit** with `feat: expose identity asset showcase`.

### Task 2: Wire unused illustrations and ornament variants

**Files:**
- Modify: `web/packages/ui/src/EmptyState.svelte`
- Modify: `web/apps/artisan/src/routes/welcome/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listing/new/processing/+page.svelte`
- Modify: `web/apps/buyer/src/routes/+page.svelte`
- Modify: `web/apps/buyer/src/routes/craft/[slug]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/listing/[slug]/+page.svelte`
- Modify: `web/apps/artisan/src/routes/stories/+page.svelte`
- Modify: `web/packages/ui/src/ui.css`
- Modify: `web/apps/buyer/src/app.css`
- Modify: `web/apps/artisan/src/app.css`
- Modify: `web/packages/i18n/src/messages/en.ts`
- Modify: `web/packages/i18n/src/messages/hi.ts`
- Test: `web/e2e/tests/buyer/visual.spec.ts`
- Test: `web/e2e/tests/artisan/stories.spec.ts`

**Interfaces:**
- Existing `Illustration`, `ProcessSequence`, `HeroBackdrop`, `Divider`, `KolamCorner`, and `CardEdge` components remain the public interfaces; callers pass existing asset names and craft values.

- [ ] **Step 1: Add assertions** that the buyer home renders `HeroBackdrop`, the craft route renders `ProcessSequence`, and the artisan welcome/processing screens render the corresponding onboarding/process assets.
- [ ] **Step 2: Run focused tests to capture the current missing-asset behavior.**
- [ ] **Step 3: Add the components to the existing screens** while preserving real data and keeping decorative SVGs hidden from assistive technology.
- [ ] **Step 4: Apply the unused ornament variants structurally** as section rules/card edges, never floating decoration.
- [ ] **Step 5: Run focused browser tests plus package validation and verify no layout-breaking console errors.**
- [ ] **Step 6: Commit** with `feat: use remaining illustration and ornament assets`.

### Task 3: Wire print and provenance assets end-to-end

**Files:**
- Modify: `web/apps/artisan/src/routes/listings/[id]/provenance/+page.svelte`
- Modify: `web/apps/artisan/src/routes/listings/[id]/provenance/+page.css`
- Modify: `web/apps/buyer/src/routes/verify/[code]/+page.svelte`
- Modify: `web/apps/buyer/src/routes/verify/[code]/+page.css`
- Modify: `web/packages/patterns/provenance.css`
- Modify: `web/packages/print/README.md`
- Modify: `web/apps/artisan/src/lib/provenance.ts`
- Test: `web/e2e/tests/buyer/provenance-qr.spec.ts`
- Test: `web/e2e/tests/artisan/shell.spec.ts`

**Interfaces:**
- Both Svelte and standalone provenance surfaces use `patterns/provenance.css`, `identity/seal.svg`, and `patterns/qr-frame.svg` through stable public asset paths.
- The print package remains a physical A4 deliverable with blank variable-data fields and a backend QR insertion slot.

- [ ] **Step 1: Add failing browser assertions** for seal, QR frame, certificate backdrop, and matching provenance content on the public verification route.
- [ ] **Step 2: Run the focused provenance tests and confirm the missing visual hooks.**
- [ ] **Step 3: Refactor provenance styles so Svelte and Go/static surfaces share the same token and asset declarations; add the identity lockup and certificate treatment to the public verification page.**
- [ ] **Step 4: Add an explicit print/download link** from the artisan provenance flow to the generated A4 tag asset, with an accessible label and no attempt to inject fake QR data client-side.
- [ ] **Step 5: Run the QR E2E test and the print generator.**
- [ ] **Step 6: Commit** with `feat: complete provenance asset wiring`.

### Task 4: Add deterministic verification and handoff documentation

**Files:**
- Create: `web/scripts/check-provenance-assets.mjs`
- Modify: `web/package.json`
- Modify: `web/ASSETS_README.md`
- Modify: `web/packages/print/README.md`
- Modify: `web/provenance-demo.html`
- Test: `web/scripts/check-provenance-assets.test.mjs` (use the repository’s existing Node test runner conventions)

**Interfaces:**
- `node scripts/check-provenance-assets.mjs` exits non-zero if required provenance assets are missing, QR quiet-zone geometry is reduced below the declared margin, or Svelte/static asset paths diverge.

- [ ] **Step 1: Write failing checks** for required asset paths, quiet-zone margin, shared provenance references, and the print files.
- [ ] **Step 2: Run the script and confirm it fails before implementation.**
- [ ] **Step 3: Implement the deterministic checker** with explicit error messages and no broad catch/silent fallback.
- [ ] **Step 4: Update the documentation** to distinguish automated checks from manual acceptance: print a tag, scan it on a phone, and compare rendered Svelte/static screenshots at the same viewport.
- [ ] **Step 5: Run the checker, existing `validate-all.mjs`, generator scripts, targeted Vitest/Playwright tests, and the frontend type/build command already defined by the workspace.**
- [ ] **Step 6: Commit** with `test: verify provenance asset handoff`.

### Final verification

- [ ] Run `node web/scripts/validate-all.mjs` from the repository root.
- [ ] Run `node web/scripts/check-provenance-assets.mjs`.
- [ ] Run the targeted buyer/artisan Playwright tests.
- [ ] Run the affected package tests/type checks using the existing `web` scripts.
- [ ] Confirm the git diff contains no generated secrets, unrelated changes, or modified read-only attachments.

