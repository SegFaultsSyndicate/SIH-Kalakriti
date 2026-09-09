# Kalakriti — web

Three SvelteKit applications, one shared design system, no Node runtime in
production. In production, a hardened NGINX reverse proxy (`deploy/nginx/nginx.conf`,
`Dockerfile.web`) serves the static assets on port 80 and proxies `/api/` to the Go BFF.

---

## Why three apps and not three route groups

The artisan application is used on a 16–32GB Android phone over intermittent
2G/3G. It is offline-first: its service worker precaches the whole shell, and
its initial JavaScript payload is a budgeted number that CI enforces.

A single SvelteKit build with route groups would put the ministry dashboard's
tables and the marketplace's discovery code in the same module graph, the same
shared chunks and the same precache manifest. Every admin dependency would
become bytes on an artisan's phone, and every admin deploy would invalidate the
artisan's precache. Three builds make that impossible by construction rather
than by discipline. The cost is three configs; the shared code lives in
`packages/` and is imported as source.

| App            | Audience                                         | Device                         | Service worker                           |
| -------------- | ------------------------------------------------ | ------------------------------ | ---------------------------------------- |
| `apps/artisan` | Artisans                                         | Mobile, low-end Android, 2G/3G | Full offline, `registerType: 'prompt'`   |
| `apps/buyer`   | Buyers, bulk buyers                              | Responsive, real connection    | Shell only, `registerType: 'autoUpdate'` |
| `apps/admin`   | Ministry officials, cluster development officers | Desktop                        | None, deliberately                       |

The artisan worker never auto-updates: a silent reload while an artisan is
part-way through photographing a piece discards capture state that has not
reached IndexedDB yet. Admin has no worker at all: it runs on managed desktops
on an office connection, where a stale cache of policy data is a hazard and
offline support buys nothing.

---

## Layout

```
web/
  apps/
    artisan/            mobile PWA, offline-first, voice-first
    buyer/              responsive marketplace
    admin/              desktop dashboard
  packages/
    tokens/             CSS custom properties: palette, scale, reset, base
    ui/                 shared primitive components
    icons/              ~90 SVG icons + <Icon>
    api/                OpenAPI-generated client + transport
    offline/            Dexie schema, outbox, connectivity
    i18n/               catalogues, locale store, money/date formatting
    illustrations/  patterns/  ornament/  motion/  identity/  print/
                        asset packs from the earlier asset batches
  e2e/                  Playwright, one project per app
  scripts/
    vite/               shared vite plugins (size reporter, aliases, chunking)
    check-brand-tokens.mjs
    gen-maskable-icons.mjs
  docs/                 asset catalogue and contact sheets
```

---

## Getting started

```bash
pnpm install
```

```bash
pnpm dev:artisan
```

Dev servers: artisan `:5173`, buyer `:5174`, admin `:5175`.
Preview servers (production build): `:4173`, `:4174`, `:4175`.

| Script                                         | What it does                                         |
| ---------------------------------------------- | ---------------------------------------------------- |
| `pnpm dev:artisan` / `dev:buyer` / `dev:admin` | one dev server                                       |
| `pnpm build`                                   | build all three to `apps/*/build`                    |
| `pnpm check`                                   | brand-token drift check, then `svelte-check` per app |
| `pnpm lint`                                    | ESLint across apps and first-party packages          |
| `pnpm test`                                    | Vitest per package (excludes e2e; see `pnpm e2e`)    |
| `pnpm e2e`                                     | Playwright only (starts its own preview servers)     |
| `pnpm size`                                    | build all three with the budget **enforced**         |
| `pnpm format`                                  | Prettier                                             |
| `pnpm assets:icons`                            | re-rasterise the maskable PWA icons                  |

Preview ports are `strictPort: true`. If `pnpm e2e` reports
`ERR_CONNECTION_RESET` on 4173, a preview server from an earlier run is still
holding the port; kill it and re-run. Without the strict port, vite would
quietly bind the next one up and Playwright would test the stale server.

The service worker is off in dev, because a worker caches the very file you are
editing. To exercise install and offline behaviour in dev:

```bash
PWA_DEV=1 pnpm dev:artisan
```

---

## The artisan bundle stays small

This is the rule the artisan app is designed around, and the only one with a
number attached.

**Budget: 120 KB gzip of initial JavaScript.** Initial means the entry chunk
plus everything it statically imports, transitively — what the browser must
download and run before the first screen is usable. Lazily imported route
chunks do not count, which is what makes the number actionable.

Measured at the end of Batch 1, with Vite 5.4.21 / SvelteKit 2.70.3 /
Svelte 5.57.0:

| App     | Initial JS (gzip)           | All JS (gzip) | CSS (gzip) | Build dir |
| ------- | --------------------------- | ------------- | ---------- | --------- |
| artisan | **70.6 KB** (59% of budget) | 73.2 KB       | 3.8 KB     | 1.1 MB    |
| buyer   | 33.0 KB                     | 35.6 KB       | 2.8 KB     | 698 KB    |
| admin   | 30.7 KB                     | 32.7 KB       | 2.8 KB     | 624 KB    |

Every build prints this table. `SIZE_BUDGET_ENFORCE=1 pnpm build` (that is what
`pnpm size` runs, and what CI runs) turns going over the budget into a build
failure. Each app also writes `apps/<app>/size-report.json` with the per-chunk
numbers.

The artisan figure is higher than the other two because it carries Dexie and
the offline layer, which the marketplace and the dashboard do not.

### Manual chunking — measured, not assumed

`build.rollupOptions.output.manualChunks` splits `node_modules` into `svelte`,
`dexie` and `vendor`. Whether that helps is an empirical question, so it was
measured both ways on the artisan app:

|                                  | Initial JS | All JS  |
| -------------------------------- | ---------- | ------- |
| with manual chunks               | 70.6 KB    | 73.2 KB |
| without (rollup's own splitting) | 72.2 KB    | 77.1 KB |

Manual chunking wins by 1.6 KB on the critical path and 3.9 KB overall, and
keeps the Svelte runtime in a chunk that a dependency change does not
invalidate. Re-check the comparison at any time with:

```bash
NO_MANUAL_CHUNKS=1 pnpm build
```

### Keeping it small

- **Import icons directly in the artisan app.** `import { Icon } from
'@kalakriti/icons'` pulls the entire ~90-icon set, because the name lookup
  needs every component available to dispatch on. Use
  `import Search from '@kalakriti/icons/src/search.svg'` instead.
- Message catalogues are dynamically imported, one chunk per language. A Hindi
  phone never downloads another language's strings.
- Anything new on the critical path has to earn its bytes against the table
  above.

---

## Static output, NGINX routing, and SPA fallback

Every app uses `adapter-static` with `fallback: 'index.html'`,
`ssr = false` and `prerender = false` in the root `+layout.ts`. In production,
all three builds are containerized via `Dockerfile.web` and served by an Alpine
NGINX reverse proxy (`deploy/nginx/nginx.conf`) exposed on **Port 80**:

- **Subdomain Routing**:
  - `kalakriti.in` (default): Buyer Marketplace (`/var/www/buyer`)
  - `artisan.kalakriti.in`: Artisan PWA (`/var/www/artisan`)
  - `admin.kalakriti.in`: Admin Dashboard (`/var/www/admin`)
- **Path Fallbacks** (for single-host / localhost testing):
  - `http://localhost/` → Buyer
  - `http://localhost/artisan/` → Artisan PWA
  - `http://localhost/admin/` → Admin Dashboard
- **API Proxy**: `/api/` requests are reverse-proxied to `bff:8000` with SSE streaming buffers disabled (`proxy_buffering off;`).
- **PWA Service Worker**: Served with `Cache-Control: no-cache, no-store, must-revalidate` so updates deploy cleanly without worker trapping.
- **Static Assets**: Precompressed `.br` and `.gz` static assets served with 1-year immutable caching (`_app/immutable/`).

Two settings that are load-bearing and easy to lose:

- **`paths: { relative: false }`.** With relative paths the single fallback
  page served at `/listings/new` looks for `./_app/` under `/listings/` and
  404s.
- **`workbox.additionalManifestEntries: [{ url: '/', … }]`.** `adapter-static`
  writes the fallback page _after_ the PWA plugin has generated its precache
  manifest, so the shell is not in the glob. Without the explicit entry, the
  navigation route is bound to a URL workbox never cached, install fails, and
  the browser discards the registration — the app then has no offline
  navigation at all, while still appearing to register a worker. It is keyed on
  `/` rather than `/index.html` because that is the URL a static file server
  actually serves the shell at.

---

## PWA

**Artisan** — full offline. Precaches the shell, icons, CSS, JS, the paper
grain and the active message catalogue (30 entries, ~217 KB). Runtime caching:

| What             | Strategy     | Detail                                    |
| ---------------- | ------------ | ----------------------------------------- |
| `GET /api/v1/*`  | NetworkFirst | 3s network timeout, 24h, 200 entries      |
| images and video | CacheFirst   | 30 days, 60 entries, purge on quota error |
| fonts            | CacheFirst   | 1 year, 24 entries                        |

Manifest: `display: standalone`, `lang: hi-IN`, `dir: ltr`, colours imported
from `@kalakriti/tokens/brand`, PNG icons at 192/512 in both `any` and
`maskable`, and shortcuts for "New listing" (`/listings/new`) and "My orders"
(`/orders`).

**Buyer** — `autoUpdate`, shell only (23 entries, ~105 KB). No media
precaching: the catalogue is large and changes constantly.

**Admin** — no service worker.

Verified by `e2e/tests/artisan/shell.spec.ts`, which asserts registration at
the root scope, the manifest link and its installability fields, and that
`/offline` still renders with the network cut.

---

## Accessibility baseline (established here, built on later)

- Skip link is the first focusable element in all three apps, shared from
  `@kalakriti/ui` so no layout can ship without one.
- No fixed px font sizes anywhere; the root font-size is left at the browser
  default. `data-text-scale` on `<html>` multiplies it (125/150/200%).
- `prefers-reduced-motion` zeroes every duration token at the root.
- Focus rings are a token, never removed.
- Pinch-zoom is not disabled.
- Connectivity state is shown with a shape _and_ a word, never colour alone.
- 44px minimum touch targets in the artisan app.

---

## Design system notes

- `@kalakriti/tokens` is the only source of colour, type, space and motion
  values. Import order: `@kalakriti/tokens`, then `/reset`, then `/base`.
- Layer order is declared once in `tokens.css`:
  `reset, tokens, base, pattern, component, utility, app`.
- Separation is by hairline rule and whitespace. There is no card-shadow token;
  the three elevation tokens are for menus, sheets and modals only.
- The page ground is khadi with a paper grain served from `static/`, not a flat
  grey.
- `packages/tokens/src/brand.js` duplicates three hex values for the PWA
  manifest, which is JSON and cannot read a custom property.
  `scripts/check-brand-tokens.mjs` (part of `pnpm check`) fails the build if
  they drift from `palette.css`.

---

## Batch 2 additions — design system foundation

The token package is documented in [packages/tokens/README.md](packages/tokens/README.md),
which also carries the **design law checklist** a reviewer runs against any new
screen. Open [packages/tokens/src/type-specimens.html](packages/tokens/src/type-specimens.html)
in a browser for the type scale, four scripts, and every colour pair with its
measured ratio in all three themes.

**Font budget, separate from the JS budget.** The artisan app self-hosts Anek
Latin only: 43.7 KB gzip for the `latin` subset, which is what a Hindi or
English page actually fetches. The Indic scripts use the platform face, because
every complete Devanagari face (Anek 251.6 KB, Noto variable 118.4 KB, Mukta
197.2 KB) breaks a 120 KB budget on its own, on a device that already has the
glyphs, over 2G. Buyer and admin opt into `@kalakriti/tokens/fonts-indic`;
artisan must not. Fonts are served by the runtime CacheFirst rule rather than
precached, so the service worker install does not carry 90.7 KB of font.

**Two verification scripts run in `pnpm check`:**

- `scripts/check-contrast.mjs` recomputes all 36 semantic colour pairs across
  the three themes from the real hex values, and fails both on a WCAG miss and
  on a comment in palette.css that no longer matches. It caught one: dark-theme
  `--k-accent-warning-text` was 1.86:1 on its own chip.
- `scripts/validate-all.mjs` (from the asset track) checks all 167 SVGs for
  viewBox, currentColor, grid conformance and size budget.

## Known debt, carried forward

- **`vite-plugin-svelte-svg` is replaced by `@poppanator/sveltekit-svg` 5.0.1.**
  The original's published peer range is `vite < 5.0.0` and `svelte < 5.0.0`;
  this workspace is Vite 5.4.21 and Svelte 5, so it cannot be installed. The
  replacement is the same author's successor with peers `vite >=5 || >=6` and
  `svelte >=5`. Registered in all three apps' vite.config.ts. Its generated
  component is `<svg {...props}>{@html contents}</svg>` with no slot, which is
  why Icon and Illustration name themselves with `aria-label` rather than a
  `<title>` child -- see apps/artisan/src/lib/icon-a11y.test.ts.

---

## Resolved versions

Recorded at install time rather than pinned from memory.

|                                |                                                      |
| ------------------------------ | ---------------------------------------------------- |
| Svelte                         | 5.57.0                                               |
| SvelteKit                      | 2.70.3                                               |
| `@sveltejs/adapter-static`     | 3.0.10                                               |
| `@sveltejs/vite-plugin-svelte` | 4.0.4                                                |
| Vite                           | 5.4.21                                               |
| `@vite-pwa/sveltekit`          | 0.6.8 (vite-plugin-pwa 0.21.2, workbox-window 7.4.1) |
| Dexie                          | 4.4.5                                                |
| TypeScript                     | 5.9.3                                                |
| Vitest                         | 2.1.9                                                |
| Playwright                     | 1.62.1                                               |
| svelte-check                   | 4.7.6                                                |
| ESLint                         | 9.39.5                                               |
| Node                           | >= 20.11 (developed on 24.18.0)                      |
| pnpm                           | 10.26.1                                              |
