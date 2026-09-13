# Frontend prompt-pack audit

Audit of `web/` against the 14-batch frontend prompt pack (BATCH 0 design law,
stack rules, per-batch acceptance criteria). Run 13 September 2026.

Verification after every fix below: `svelte-check` across all three apps
(0 errors), `check-contrast.mjs` (39 pairs pass), `@kalakriti/api` unit tests
(30 pass).

---

## Fixed

| # | Rule broken | Where | Fix |
|---|---|---|---|
| 1 | BATCH 4: "Do not store tokens in localStorage" | `packages/api/src/auth.ts` | Deleted the localStorage mirror for both tokens. Memory + IndexedDB only, restored once per app start. The file's own header already said "Never localStorage" — the mirror was added later, against it. All three layouts already `await restoreAccessToken()` before the first request, so no call site changes. |
| 2 | BATCH 0: "NEVER use Svelte 4 syntax" | `packages/patterns/Card.svelte` | The only Svelte 4 file left: `export let` ×2 and a `$:`. Ported to `$props()` / `$derived()` / `{@render children?.()}`. Imported in 8 places, so ported rather than deleted. |
| 3 | BANNED: purple-to-pink gradients; gradient as a primary surface | `artisan/lib/DigitalLiteracyTutorial.svelte` | Four step-card gradients removed, including `#581c87 → #6b21a8` — purple-900 to purple-800, the first item on the banned list. Card is now a flat `--k-surface-inverse`. |
| 4 | BANNED: glassmorphism / backdrop-blur cards | Same file, `buyer/lib/ListingCard.svelte`, `buyer/lib/SellerShowcaseBanner.svelte` | `backdrop-filter: blur()` removed from card and chip surfaces, replaced with opaque token backgrounds. Modal scrims keep theirs — a blurred overlay behind a dialog is the standard treatment, not the banned pattern. |
| 5 | REQUIRED: separation by line and space, not shadow | Same files | Decorative `box-shadow`s on those surfaces removed. |
| 6 | BANNED: gradient as a primary surface | `artisan/routes/profile`, `artisan/routes/+page`, `artisan/lib/GemExportPreview` | Three more gradients flattened to semantic tokens. The sahayak banner now separates with a hairline plus an accent edge rule instead of a wash. |
| 7 | BATCH 4 ABSOLUTE RULE: never invent a field absent from the spec | `buyer/lib/ListingCard.svelte:36,56` | `(listing as any).artisan_image_url` — zero hits in `services/bff/openapi.json`, so the `|| devAvatar` fallback was *always* taken: a dead branch pretending to be real. Removed. `(listing.price as any)?.amount` removed too; `amount_paise` is the spec field and was already first. |
| 8 | BATCH 0: TypeScript strict, no `any` | `artisan/routes/profile/+page.svelte` | Two `catch (err: any)` → `unknown`, narrowed through `ApiError`. |
| 9 | BANNED: emoji as icons or bullets, anywhere | 9 files | 🏛️ 📍 🎖️ 💡 📖 ⚡ 🎪 📅 💳 🏦 🧵 🧣 🏺 🛍️ 🛡️ and two ★ bullets replaced with `packages/icons` components or dropped. ✓ ✕ ➔ left: dingbats in prose, not icons. |
| 10 | BATCH 2/3: a component may not reference a raw colour | 33 files, `<style>` blocks | 898 of 901 hex literals mapped to semantic aliases. Kept literal: the tricolour saffron `#FF9933` and green `#138808`, and WhatsApp's `#25D366` — brand and flag colours are reproduced exactly or not at all. |
| 11 | BATCH 2/3: same, for raw palette steps | 30 files | Raw `--k-<family>-<step>` references in component styles: 561 to 195, after G1 added the aliases that were missing. |
| 12 | BANNED: gradient as a primary surface (follow-up) | `artisan/lib/DigitalLiteracyTutorial.svelte` | Flattening the four step gradients left the card flat and dull. Each step now carries one accent from the craft palette — terracotta, indigo, haldi, neem — on the icon disc, the badge and a 4px top rule, with a keyed 0.22s enter and a `prefers-reduced-motion` opt-out. Flat fills, no gradient. |

The colour work is a codemod, `scripts/hex-to-token.mjs`, kept in the repo so
the mapping is reviewable and re-runnable rather than a one-off hand edit. It
touches `<style>` blocks only — never markup, strings or i18n catalogues — maps
each literal to its nearest palette step by CIE76 distance in Lab, and picks the
alias by CSS property role, so a dark panel background cannot become a text
token and invert under dark mode.

---

## Gaps: now closed

**G1 — the semantic layer had no room for half its own uses. Fixed.**
13 aliases added to `palette.css` in all three themes, every text and border
pairing measured and added to `check-contrast.mjs` (66 pairs now, was 39):
`--k-text-tertiary`, `--k-surface-neutral`, `--k-border-subtle/-muted/
-on-inverse/-accent/-danger/-warning`, `--k-accent-danger-bg/-success-bg/
-danger-muted/-danger-strong/-success-muted`. The codemod's text role also
now recognises accent aliases (`--k-accent-danger` *is* what danger text is
painted with), which was a script bug, not a missing token. Raw palette-step
references in component styles: 515 to 195.

One case deliberately left raw: `color: #C65D3B` and friends land on
`--k-terracotta-600`, which is 4.45:1 on `--k-surface-base` — below AA. There
is no `--k-accent-primary-muted` because minting one would launder a failing
colour into a token. Those ~16 declarations are pre-existing contrast
failures and should be darkened, not tokenised.

**G2 — the provenance page is no longer styled twice. Fixed.**
`verificationPageCSS` was a hand-maintained Go string with the palette's hex
values copied into it. It is now generated:
`web/packages/patterns/verify-page.css` holds the rules in tokens,
`scripts/gen-verification-css.mjs` resolves them against `palette.css` and
writes `services/bff/internal/bff/handler/verification_css.go`, and
`pnpm check` runs it with `--check` so a stale copy fails the build. Only the
tokens the page reaches are emitted (3.2 KB, whole-palette would be 5.5 KB).
The page stays inlined in one request on purpose — it is reached by a
stranger scanning a tag, often on a bad connection.

`packages/patterns/provenance.css` is not a second copy of that page: it is
the ornamental certificate treatment from the Batch 6 print pack. Its header
now says so, and says that adopting it on the live page is a design decision
rather than a cleanup.

**G3 — bundle budgets fail the build by default. Fixed.**
`enforce` flipped from `SIZE_BUDGET_ENFORCE === '1'` to `!== '0'` in buyer and
artisan; admin had a reporter but no budget at all and now has one (80 KB).
Measured: artisan 49.0 KB of 120, buyer 48.2 of 200, admin 35.1 of 80.
`pnpm size` is now the same thing as `pnpm build`, and the README/FRONTEND.md
figures were stale by 20 KB and are corrected.

**G4 — the two `state_referenced_locally` warnings. Fixed.**
`artisan/lib/OrderTimeline.svelte` and `buyer/routes/orders/[id]/+page.svelte`
both captured `orderId` once, so a route-param change left the SSE stream
subscribed to the previous order. Both now open the watcher inside an
`$effect` that re-subscribes and stops the old one on cleanup; the manual
`onDestroy` teardown is gone with it.

---

## Verified present (spot-checked, not exhaustive)

- No SSR anywhere: all three `+layout.ts` set `ssr = false`, `prerender = false`,
  and there is not one `+page.server.ts` or `+server.ts` in the repo.
- No Inter / Poppins / Montserrat.
- 22 scheduled languages plus English present in `packages/i18n/src/messages/`,
  with a fallback catalogue.
- `Save-Data` / `effectiveType` honoured (`packages/offline/src/connection.ts`,
  `packages/ui/src/Image.svelte`).
- Small-bucket suppression renders as suppressed, never as zero
  (`admin/routes/insights/+page.svelte`).
- Command palette, `DEMO_MODE`, axe sweeps per app in `e2e/`, type-specimen
  page, and the `/stories` design-system review page all exist.
