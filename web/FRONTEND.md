# Kalakriti — frontend reference

Everything a new contributor or a demo presenter needs about the three web
apps: what exists, where it lives, what it looks like, and how to run it.
This is a snapshot doc, not a spec — see the prompt pack for the original
batch-by-batch brief this was built against. Gaps found while writing this
are marked **GAP** and are real, not hypothetical.

---

## The three apps

| App | Audience | Device | Dev port | Preview port | Production / NGINX route (Port 80) | Service worker |
|---|---|---|---|---|---|---|
| `apps/artisan` | Artisans | Low-end Android, 2G/3G | 5173 | 4173 | `/artisan/` or `artisan.kalakriti.in` | Full offline, `registerType: 'prompt'` |
| `apps/buyer` | Buyers, bulk buyers | Any device, real connection | 5174 | 4174 | `/` or `kalakriti.in` | Shell only, `registerType: 'autoUpdate'` |
| `apps/admin` | Ministry officials, cluster development officers | Desktop | 5175 | 4175 | `/admin/` or `admin.kalakriti.in` | None (deliberately) |

Three separate SvelteKit builds, not route groups — an admin dependency must
never become bytes on an artisan's phone, and an admin deploy must never
invalidate the artisan's precache. Shared code lives in `packages/` and is
imported as source, not published.

Stack: SvelteKit 2 + Svelte 5 (runes only), TypeScript strict, `adapter-static`
(no SSR, no Node runtime in production — an Alpine NGINX container built via
`Dockerfile.web` serves static assets on Port 80 and proxies `/api/` to the Go BFF
at `:8000`), Dexie for offline storage, vanilla CSS with custom properties
(no Tailwind, no component library).

---

## Running it locally

### Option 1: Local Vite Dev Servers (Hot Reload)
```bash
cd web
pnpm install
pnpm dev:artisan     # :5173
pnpm dev:buyer       # :5174
pnpm dev:admin       # :5175
```

### Option 2: Full Stack via Docker Compose (NGINX + BFF + Infra)
```bash
# From repository root
make demo-up
# Or directly:
docker compose -f docker-compose.full.yml up -d
```
All 3 web apps are served at `http://localhost` (port 80), and the Go BFF is at `http://localhost:8000`.

Each app needs the Go BFF reachable at whatever `@kalakriti/api`'s transport
points at (`packages/api/src/transport.ts`) — without it, every screen still
boots (offline-first design), but nothing beyond local/cached state resolves.

The service worker is off in dev by default (a worker would cache the file
you're editing). To test install/offline behavior in dev:

```bash
PWA_DEV=1 pnpm dev:artisan
```

Other useful scripts (run from `web/`):

| Script | What it does |
|---|---|
| `pnpm build` | builds all three apps to `apps/*/build` |
| `pnpm build:artisan` / `:buyer` / `:admin` | build one app |
| `pnpm preview:artisan` / `:buyer` / `:admin` | serve a production build locally |
| `pnpm check` | brand-token drift check + `svelte-check` per app |
| `pnpm lint` | ESLint across apps and packages |
| `pnpm test` | Vitest per package |
| `pnpm e2e` | Playwright suite (starts its own preview servers) |
| `pnpm size` | build all three with the JS budget **enforced** — fails on regression |
| `pnpm format` | Prettier |
| `pnpm assets:icons` | re-rasterize maskable PWA icons |

Preview ports are `strictPort: true` — if `pnpm e2e` throws
`ERR_CONNECTION_RESET` on 4173, a stale preview server from an earlier run is
still holding the port; kill it and re-run.

---

## Sitemap

### Artisan (`apps/artisan`)

Onboarding:
`/language` → `/welcome` → `/login` → `/verify` →
`/register/name` → `/register/craft` → `/register/district` →
`/register/cluster` → `/register/pehchan`

Home and shell:
`/` (home), `/offline` (useful offline fallback, not a dead end),
`/accessibility`, `/stories` (component gallery / design-review page)

Capture-to-listing flow (`/listing/new/...`):
`/capture` → `/video` → `/processing` → `/pricing` → `/story` →
`/review` → `/terms`

Listings:
`/listings`, `/listings/[id]`, `/listings/[id]/provenance`

Orders:
`/orders`, `/orders/[orderId]`,
`/orders/[orderId]/lots/[lotId]`, `/orders/[orderId]/lots/[lotId]/offer`

Other: `/earnings`, `/notifications`, `/profile`

**GAP**: the PWA manifest's "New listing" shortcut points at
`/listings/new`, which doesn't exist — the real route is
`/listing/new/capture` (singular "listing"). The shortcut is dead; fix in
`apps/artisan`'s manifest config (wherever `shortcuts` is set alongside the
`@vite-pwa/sveltekit` config).

### Buyer (`apps/buyer`)

`/` (home — hero, shop-by-craft, editorial sections, artisan-of-the-month,
trust strip), `/search`, `/feed` (vertical process-provenance clips),
`/craft/[slug]`, `/artisan/[slug]` (storefront), `/listing/[slug]` (product
page), `/bulk-order`, `/orders`, `/orders/[id]`, `/accessibility`

**GAP**: no buyer-side provenance/QR verification route. The spec (Batch 12)
called for a public provenance page; `DEMO.md`'s script currently sends the
presenter back to the artisan app's `/listings/[id]/provenance` to show
this. Worth adding a buyer-facing equivalent, or a public
`/verify/[hash]`-style page.

### Admin (`apps/admin`)

`/` (dashboard), `/login`, `/login/verify`, `/clusters` (roster, capacity,
CSV bulk onboarding), `/crafts` (ontology management), `/insights`
(ministry analytics — GMV, income uplift, dying-craft watch, suppressed
small-bucket rendering), `/moderation` (review queue, human-decision-only),
`/accessibility`

Global command palette (Ctrl/Cmd+K) for navigation and search.

---

## Design system

**Colors** — `packages/tokens/src/palette.css`, natural-dye palette:
khadi (base), ink, terracotta, indigo, haldi, madder, neem, plus neutral
stone greys. Semantic aliases only (`--k-surface`, `--k-text`, `--k-accent`,
`--k-focus-ring`, etc.) — components never touch a raw palette step.
Light, dark, and high-contrast themes, all driven by swapping the same
semantic aliases. 36 colour pairs verified against real WCAG ratios by
`scripts/check-contrast.mjs`, part of `pnpm check`.

Manifest / brand colors (from `packages/tokens/src/brand.js`, checked against
`palette.css` for drift): background `#FCFAF6` (khadi-50), theme
`#963D14` (terracotta-700), dark background `#0F0A07` (ink-950).

**Typography** — `@fontsource-variable/anek-{latin,devanagari,bangla,tamil}`.
Anek Latin (variable) for Latin script; Indic scripts render with the
platform system face rather than a self-hosted font, because a full Indic
face (118–252 KB gzip depending on family) alone blows the artisan app's
120 KB JS budget on a 2G connection where the device already has the
glyphs installed. The artisan app self-hosts **Latin only** (43.7 KB gzip);
buyer and admin additionally opt into `@kalakriti/tokens/fonts-indic` since
they don't carry the same budget pressure. Fonts are runtime-cached
(CacheFirst, 1 year) rather than precached, so install doesn't have to
download them up front. Type specimens with every script and every color
pair's measured ratio: `packages/tokens/src/type-specimens.html`.

**Icons** — `packages/icons`, ~89 SVG components, `currentColor`-based,
uniform viewBox, validated by `scripts/validate-all.mjs`. In the artisan
app, icons are imported directly per-file
(`import Search from '@kalakriti/icons/src/search.svg'`) rather than via
the barrel `Icon` component, because the barrel pulls in the full set —
this is a deliberate bundle-size tradeoff, not an oversight.

**Illustrations / patterns / ornament** — `packages/illustrations` (~24:
empty states, onboarding art, craft-process sequences for
blockprint/pottery/weaving), `packages/patterns` and `packages/ornament`
(jaali/paper-grain/QR-frame motifs used structurally as dividers and
borders, per the design law — never as decorative stickers).

**Motion** — `packages/motion`; all durations zero out under
`prefers-reduced-motion`.

**Components** — `packages/ui`, ~33 primitives: Button, Dialog, Sheet,
Stepper, OtpInput, VoiceInput, SpeakButton, AccessibilityControl, Money,
CraftTerm, Toast, EmptyState, SectionHeader, and others — all keyboard-
complete, all built on tokens, none importing a raw palette value.

---

## PWA

**Artisan**: full offline. Precaches shell, icons, CSS, JS, paper-grain
background, and the active language's message catalogue only (~217 KB for
30 message-catalogue entries). Runtime caching: API GETs NetworkFirst
(3s timeout, 24h, 200 entries), media CacheFirst (30 days, 60 entries,
purges on quota pressure), fonts CacheFirst (1 year). Update prompt is
non-forcing — `registerType: 'prompt'`, never auto-reloads mid-capture.
Install prompt withheld until the artisan has published at least one
listing (`hasPublishedOnce()` in `apps/artisan/src/lib/listings.ts`) —
never shown on first load.

**Buyer**: `autoUpdate`, shell-only precache (~105 KB, 23 entries) — no
media precaching since the catalog is large and constantly changing.

**Admin**: no service worker — runs on managed desktops where offline
support buys nothing and a stale cache of policy data is a hazard.

iOS notes (documented, not fixable client-side): no Background Sync API
(outbox drains on next foreground instead), no `beforeinstallprompt`
(install path is manual Share → Add to Home Screen), aggressive storage
eviction after ~7 days unopened.

---

## Bundle budget

**120 KB gzip** initial JS for the artisan app — the entry chunk plus
everything it statically imports before first paint. Lazy route chunks
don't count. `pnpm size` (`SIZE_BUDGET_ENFORCE=1 pnpm build`) turns a
regression into a build failure; this also runs in CI. Buyer's budget is
200 KB gzip. Last measured: artisan 70.6 KB (59% of budget), buyer 33.0 KB,
admin 30.7 KB — see `web/README.md` for the full table and how manual
chunking was verified to help.

---

## Accessibility

WCAG 2.1 AA target. Skip link first-focusable in every app. No fixed-px
font sizes (`data-text-scale` on `<html>` drives 125/150/200% scaling).
`prefers-reduced-motion` zeroes all motion tokens. Focus rings are a token,
never removed. 44px minimum touch targets in the artisan app. Connectivity
state is always shown with shape *and* word, never colour alone.

axe-core sweeps run in CI across every route in all three apps (three
themes each) — see `e2e/tests/*/a11y-sweep.spec.ts`.

**GAP**: each app's `/accessibility` route is currently a 13-line stub
(identical across all three), most likely just wrapping
`<AccessibilityStatement>` from `packages/ui` without app-specific known-gap
content. Worth reviewing against what `ACCESSIBILITY.md` already documents
as unverified, so the public-facing statement doesn't overclaim.

Full manual-verification checklist (keyboard traversal, screen-reader pass,
200% zoom, real-device offline test, Lighthouse CI status): `web/ACCESSIBILITY.md`.

---

## Demo

Full 8-minute click-path script, timings, and a troubleshooting table:
`web/DEMO.md`. `PUBLIC_DEMO_MODE=1` is a cosmetic-only flag (fixed
empty-state illustrations, deterministic timings) — it does **not** seed
any data; there is no seed-data backend for artisans/listings/orders yet.

---

## Known gaps summary (for planning, not alarm)

1. Artisan PWA install shortcut points at a dead route (`/listings/new`).
2. No buyer-side provenance/QR verification page.
3. `/accessibility` pages are stubs, not fleshed-out statements.
4. 2 of 4 required Playwright e2e journeys unwritten (buyer cross-lingual
   search, bulk-order dropout/reallocation, provenance QR) — tracked in
   `ACCESSIBILITY.md`.
5. No visual-regression baseline screenshots committed yet (needs a CI run
   with real browser binaries).
6. `HANDOFF.md` at the repo root is stale — references a worktree path
   that no longer exists now that all batches are merged to `main`.
