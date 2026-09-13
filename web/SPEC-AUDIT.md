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
| 11 | BATCH 2/3: same, for raw palette steps | 30 files | 46 of 391 raw `--k-<family>-<step>` references swapped for the semantic alias playing their role. The other 345 are blocked — see gap G1. |

The colour work is a codemod, `scripts/hex-to-token.mjs`, kept in the repo so
the mapping is reviewable and re-runnable rather than a one-off hand edit. It
touches `<style>` blocks only — never markup, strings or i18n catalogues — maps
each literal to its nearest palette step by CIE76 distance in Lab, and picks the
alias by CSS property role, so a dark panel background cannot become a text
token and invert under dark mode.

---

## Gaps: specified but not implemented

**G1 — the semantic token layer has no room for half its own uses.**
345 raw palette-step references cannot be converted because no alias plays
their role: there is `--k-accent-danger` (text) but no danger *background*, no
hover step, no decorative border role. A component wanting a danger fill has
nowhere to go but the raw scale, so BATCH 2's "never reference a raw palette
step" is unfollowable as the palette currently stands. Worst: `--k-madder-600`
×60, `--k-neem-600` ×40, `--k-stone-100` ×38, `--k-madder-800` ×34. Fix is
adding the missing aliases to `palette.css` with measured ratios — a palette
decision with contrast consequences, so not guessed at here.

**G2 — the public provenance page is styled twice, in two languages.**
BATCH 12 item 3 asked for one standalone stylesheet built from the tokens, plus
the exact HTML structure, handed to the backend track so both render
identically. Reality: `services/bff/internal/bff/handler/verification.go` has
its own inline `verificationPageCSS` with hex values copied literally and a
`.k-verify` class tree, while `web/packages/patterns/provenance.css` (119 lines,
token-based, `.k-provenance`) is referenced by nothing. Two stylesheets for one
page, drifting independently. The "we cannot verify this tag" state does exist,
on the Go side only.

**G3 — bundle budgets do not fail the build by default.**
BATCH 14 asks for budgets enforced in CI, failing on regression. The reporter
exists in all three vite configs but only enforces when `SIZE_BUDGET_ENFORCE=1`,
which only `pnpm size` sets. A regression passes `pnpm build` silently.

**G4 — two `state_referenced_locally` warnings.**
`artisan/lib/OrderTimeline.svelte:35` and `buyer/routes/orders/[id]/+page.svelte:66`
both capture `orderId`'s initial value only. Pre-existing, unrelated to this
audit; a route-param change would not re-subscribe the SSE stream.

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
