# Kalakriti Web — Monorepo Structure

## Overview

Three separate SvelteKit + Svelte 5 apps sharing a design system, built on `adapter-static` with no Node runtime in production.

```
/web
  /apps
    /artisan      — Mobile PWA, offline-first. Artisan capture & sales.
    /buyer        — Responsive marketplace. Discovery & purchase.
    /admin        — Desktop dashboard. Ministry officials & cluster officers.
  /packages
    /tokens       — Design system: CSS custom properties, typography, spacing
    /ui           — Primitive components: Button, Field, Dialog, Sheet, etc.
    /icons        — SVG icon set + Icon.svelte component
    /api          — OpenAPI client, auth, error handling
    /i18n         — i18n system, voice, accessibility (text scale, contrast)
    /offline      — Dexie schema, outbox sync, conflict handling
    /* rest: illustrations, patterns, ornament, motion, identity, print
  /scripts        — Asset generation & validation (from asset pack)
  /docs           — Asset catalogue, pitch summary, contact sheets
  /e2e            — Playwright cross-app tests
```

## Getting Started

Install dependencies:
```bash
pnpm install
```

Dev servers (pick one):
```bash
pnpm dev:artisan    # Mobile PWA at :5173
pnpm dev:buyer      # Marketplace at :5174
pnpm dev:admin      # Dashboard at :5175
```

Build all:
```bash
pnpm build
```

## Design System

All three apps use `/packages/tokens` for:
- Colour palette (khadi, ink, terracotta, indigo, haldi, etc.)
- Typography (Anek Latin + Anek Devanagari, Mukta for body)
- Spacing scale (4px base)
- Motion durations
- Dark mode, high-contrast mode

No Tailwind. No component library. Vanilla CSS with `@layer`.

## API Contract

The Go BFF at `:8080/api/v1` exposes OpenAPI 3.1. Generate the TypeScript client:
```bash
pnpm api:gen
```

Never hand-write a request type. The generated `schema.d.ts` is the source of truth.

## Artisan Bundle Budget

**Target: <150KB gzipped** for initial JS (Batch 1 target).

Check with:
```bash
pnpm build && du -sh apps/artisan/dist
```

The artisan PWA must stay small for 2G / low-end Android. Violations fail CI.

## Asset Packages

All 167 SVG assets across 8 packages are co-located with the frontend:
- Icons: 89 UI icons, micro-icon system
- Illustrations: ~30 scenes for empty states, onboarding
- Patterns: Ajrakh, khadi, jaali
- Identity: Logomark, wordmark, emblem (with State Emblem legal notice)
- Print: A4 provenance tags for batch export

Run the validator before any commit:
```bash
pnpm validate
```

Batch 8 of the asset pack introduced:
- Comprehensive SVG validation (viewBox, no bare hex, SMIL, etc.)
- Fixed SVGO config to preserve XML comments (Batch 0 mandate)
- Icon stroke-weight CSS selectors (dense/default/bold)
- Component props: size, strokeWidth, title, class

## Three Batches, Three Tracks

**Foundational (Batches 1-6):** Monorepo, tokens, UI, API, auth, offline — all sequential.

After that:
- **Frontend A (Batches 7-10):** Artisan app — capture, listings, pricing, orders
- **Frontend B (Batches 11-13):** Buyer + admin — discovery, purchase, insights
- **Batch 14:** Accessibility audit, performance, PWA polish, e2e, demo hardening

## Key Rules

- One component per file, PascalCase, Svelte 5 runes ($state, $derived, $effect).
- No `export let`, no `$:` reactive statements, no stores where a rune fits.
- Semantic HTML first; ARIA only where semantics run out.
- Every interactive element keyboard-operable with a visible focus ring.
- Every user-visible string via i18n. Never hardcode English.
- No dark patterns, no urgency, no purple gradients.
- WCAG 2.1 AA minimum. Artisan app at 360px, all apps at 200% zoom.

## Contact & Accessibility

Each app has an `/accessibility` route (GIGW convention) documenting conformance and a contact path.

---

See `/ASSETS_README.md` for the design system and asset-pack documentation.
