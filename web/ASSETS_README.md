# Kalakriti asset pack — delivered so far

This tracks progress against `Kalakriti_Assets_Prompt_Pack.md` (8 batches).

## packages/tokens/  (Batch 1 — complete)
- `src/palette.css` — OKLCH-authored colour scales (khadi, ink, terracotta,
  indigo, haldi, madder, neem, stone) + semantic light/dark/high-contrast
  theme mappings, every text-on-surface pair carrying a measured WCAG
  contrast ratio.
- `src/motifs.md` — grammar for the six licensed traditions (Warli, jaali,
  block print, kolam, charkha/lotus/mandala, handloom), what each is
  licensed for, and what counts as misuse.
- `src/stroke-geometry-spec.md` — icon/illustration stroke weights, corner
  radius convention, kolam dot-grid pitch, optical correction rules.
- `palette.html` — contact sheet: every scale + every semantic pairing with
  its measured ratio, across all three themes.

## packages/identity/  (Batch 2 — complete)
- `src/logomark-{solid,outline,reversed}.svg` — warp/weft crossing mark
  (one bar passes over, one is interrupted at the crossing).
- `src/wordmark-{horizontal,stacked}.svg` — "Kalakriti" / "कलाकृति", real
  converted letterform outlines (Fraunces + HarfBuzz-shaped Noto Serif
  Devanagari), not `<text>`.
- `src/emblem-lockup.svg` — mark + wordmark + ministry caption + a
  placeholder slot for the State Emblem of India (explicitly not drawn —
  see the file's own comment on the 2005 Act).
- `src/heritage-crest-{fine,coarse}.svg` — concentric jaali watermark
  device, two densities.
- `src/app-icon-maskable{,-mono}.svg`, `src/apple-touch-icon.svg`,
  `src/favicon.svg` — plus rendered PNGs in `raster/` at 192/512/1024,
  180, and 16/32px, including a circular-crop proof.
- `src/seal.svg` / `src/seal-print.svg` — provenance seal, simplified for
  legibility at 15mm print. The 15mm simulation lives in the identity
  contact sheet, not in `raster/`.
- `contact-sheet-batch2.html` — every asset above, plus the 16/24px
  logomark, the circular-crop test, and the 15mm print simulation, all
  rendered at real target size rather than scaled down from a preview.

## packages/ornament/  (Batch 3 — complete)

Structural ornament: things that divide, edge and frame. Nothing in this
package is a sticker — if one of these ends up floating in a corner as
decoration, it is being used wrong.

- `src/jaali-{hex,octstar,interlace}-{16,24,32}.svg` — nine section-divider
  tiles: three jaali geometries (hexagonal, octagonal-star,
  interlaced-square) at fine / medium / bold density, each seamlessly
  tileable along X.
- `src/blockprint-running-20.svg`, `src/blockprint-band-32.svg` — Ajrakh /
  Bagru border strips, fill-led with resist-dye negative space: a simple
  running border for card edges, a layered band for page headers and the
  provenance page.
- `src/kolam-corner.svg` — sikku-kolam L-corner, one continuous closed line
  looping around a five-dot L on the 24px grid, touching no dot. Four
  rotations come from CSS transforms, not four files.
- `src/rule-knot.svg`, `src/rule-double-woven-8.svg` — hairline rule motifs.
  The plain hairline is CSS only (`.k-rule`), so it has no image and cannot
  fail to load.
- `src/weft-texture-12.svg` — plain-weave page texture, 545B raw / 208B
  minified, for 3–6% opacity.
- `src/card-edge.svg` — 9-slice printed card frame, replacing drop shadows.
- `src/patterns.svg` — every tileable asset as an SVG `<pattern>`, ids
  namespaced `k-*`. Inline once per document, then
  `fill="url(#k-jaali-hex-24)"`.
- `ornament.css` — the CSS utilities, and `Divider/Rule/KolamCorner/CardEdge`
  Svelte components plus `index.js`.
- `contact-sheet-batch3.html` — full-width tiling at every density on all
  three themes, an explicit four-tile seam test per tile, the weft at
  3 / 6 / 12%, and the rule knot at 16/20/24/48px.

### The one thing not to undo

The dividers are applied as a CSS **mask** over `background-color:
currentColor`, never as a `background-image`. An SVG referenced from
`background-image` or `border-image` — data URI included — loads in an
isolated document where `currentColor` resolves to black and the `--k-*`
tokens are invisible. A divider delivered that way looks correct in a
contact sheet and is wrong in the product. `ornament.css` says this at the
top; the card edge is the single documented exception, and it bakes a hex
per theme because a 9-slice border has no mask equivalent with full browser
support.

### Regenerating

```
node scripts/gen-batch3.mjs
```

Every tile, the pattern sprite, `ornament.css` and the contact sheet are
generated from that one script, so the seam guarantee is a property of the
construction rather than of an illustrator's patience. It runs four hard
checks and exits non-zero on any failure: seam parity (what crosses `x=0`
must equal what crosses `x=W`), the 32px divider height cap, per-asset size
budgets, and the Batch 0 constraints (viewBox present, no root width/height,
no bare hex, no unreferenced ids, max 2 decimal places).

### SVGO

`svgo.config.mjs` in the pack root is the minimum config that is safe for
these assets — the default preset removes the viewBox, can fold
`currentColor` away, and strips the ids that `patterns.svg` exists to expose.
Verified on the whole batch: viewBox and `currentColor` survive on all 16
assets, no bare hex appears, and `patterns.svg` keeps all thirteen pattern
ids. Batch 8 owns the full pipeline; this file is what makes Batch 3 safe to
optimise today.

## packages/icons/  (Batch 4 — complete)

The UI icon set: 61 core icons, 14 domain icons, 12 craft-category icons,
the charkha spinner, and the upload-zone graphic — 89 SVGs, all on the
24x24 grid, 1.5 stroke, round cap/join (icons are UI chrome, not a
geometric motif, so they don't take the miter rule Batch 3's dividers do).

- `src/*.svg` — one file per icon. Generated from a primitive DSL (line /
  poly / circ / arc / dot) in `scripts/gen-batch4.mjs` rather than
  hand-typed path data, so the set stays geometrically consistent across
  ~90 icons and is cheap to adjust.
- `Icon.svelte` — `<Icon name="search" />`, decorative by default, `title`
  switches to `role="img"` with `aria-labelledby` wired through.
- `Spinner.svelte` — the charkha, indeterminate (CSS rotation) or
  determinate (`progress={0..1}`, thread winding onto the spindle via
  `stroke-dashoffset`, no timer). Both variants have a
  `prefers-reduced-motion` fallback (opacity pulse, no rotation).
- `UploadZone.svelte` — one asset (`upload-zone.svg`), five states
  (idle/drag/uploading/success/error) switched by a `data-state` attribute
  in `icons.css`, not five separate files.
- `icons.js` / `icons.d.ts` / `manifest.js` — the name→component map behind
  `<Icon>`, the `IconName` union type for editor autocomplete, and the
  docs/validation manifest.
- `icons-sprite.svg` — optional `<symbol>`-based sprite for the buyer app.
- `icons.css` — sizing, the 1.25/1.5/2 stroke-weight variants (one path,
  stroke-width swap — never a redrawn icon), spinner keyframes, upload-zone
  state visibility.
- `contact-sheet-batch4.html` — a 16px dense grid of the whole set (the
  view that actually surfaces stroke-weight inconsistency), every icon at
  16/20/24/32px with its name, the spinner in all three motion states, and
  the upload zone in all five states — all on light/dark/high-contrast.

### Two ways to import an icon

`<Icon name="search" />` is the convenience path but pulls in all ~90
components, since the name lookup has to dispatch on any of them. A direct
`import Search from '@kalakriti/icons/src/search.svg'` tree-shakes to just
that icon. Batch 8 proves the actual bundle number; this package only keeps
the two paths from being the same entry point (`icons.js` is not
re-exported as part of the tree-shaking path).

### WhatsApp

`whatsapp.svg` is deliberately not the WhatsApp brand mark, which is
trademarked — it's a generic handset-in-bubble glyph standing in for the
channel. Said again in the file's own comment.

### Regenerating

```
node scripts/gen-batch4.mjs
```

Checks run on every icon: `viewBox="0 0 24 24"` exactly, `stroke-width="1.5"`
literal (not the token-driven approach Batch 3's patterns use — weight here
is a per-render CSS override, not a per-tile constant), round cap/join, no
bare hex, no unreferenced id, no SMIL `<animate>`, every path endpoint inside
the 20x20 safe area, and the 2KB per-icon budget. SVGO round-trip verified
separately (`svgo.config.mjs`): viewBox and `currentColor` survive on all 89.

## packages/illustrations/  (Batch 5 — complete)

30 SVGs: 8 empty states, 4 onboarding scenes, 4 feature illustrations, 12
craft-process step icons (3 crafts × 4 steps), a tileable hero backdrop, and
the FFT comparison graphic. All Warli-derived economy of line — geometric,
flat, two colours maximum (`currentColor` plus one `--k-illustration-accent`
used sparingly for the one detail each scene wants to draw the eye to).

- `src/empty-*.svg` — no-listings (empty strung loom), no-orders (folded
  cloth + ledger), no-search-results (jaali window), no-notifications (a
  still charkha), offline (a thread with one end curling loose), error (a
  dropped shuttle, tangled thread), no-followers (a Warli figure alone at a
  loom), empty-cart (an empty interlaced-cane basket). 320×240, legible at
  the 200px-wide real size the acceptance criterion asks for.
- `src/onboard-*.svg` — photograph your work, we write the description
  (warp threads becoming text lines), buyers find you (a kolam-derived node
  network), you are paid directly (one thread, two Warli figures, no
  intermediary node in between — the whole slide's argument in one image).
- `src/feature-*.svg` — collective fulfilment (many looms, one cloth),
  provenance (seal with a thread trail to its loom), handloom verification
  (irregular hand-woven grid vs. regular machine grid, the same comparison
  the FFT graphic makes numerically), fair pricing (a balance scale).
- `src/process-{blockprint,weaving,pottery}-{1..4}-*.svg` — four steps per
  craft, 120×120. The step number is a **dot count** in the corner, not a
  numeral — a digit needs its glyph converted to a path (as Batch 2's
  wordmark did), disproportionate for twelve small icons, and a dot count
  reads independent of language.
- `src/hero-backdrop-tile.svg` — a 60×60 tileable jaali+weft pattern, not a
  fixed hero-width canvas: the backdrop repeats regardless, so a one-off
  1440px asset would blow the illustration size budget for no visual gain.
- `src/fft-comparison.svg` — two fabric swatches (irregular hand-woven,
  regular machine-made) each above its own frequency-spectrum bar chart —
  broad and uneven vs. one sharp accent peak. Bar heights are placeholder
  data at a fixed 10-bar layout the ML track's real FFT output can drop into.
- `illustrations.css` — the accent-colour tokens per theme, the hero-backdrop
  mask, and the process-strip layout.
- `Illustration.svelte`, `ProcessSequence.svelte`, `HeroBackdrop.svelte` +
  `index.js` / `manifest.js`.
- `contact-sheet-batch5.html` — every illustration at its real rendered
  size, the FFT graphic, all three process sequences, and the hero tile
  visibly seamed (raised opacity) — light/dark/high-contrast.

### The hero backdrop repeats the Batch 3 mask rule

`illustrations.css`'s `.k-hero-backdrop::before` is a CSS mask over
`background-color: currentColor`, never `background-image` — the same rule
`ornament.css` documents at length in Batch 3. An early draft of this file
used `background-image` and rendered solid black regardless of theme; caught
by re-reading Batch 3's own reasoning before shipping. Don't undo it.

### Regenerating

```
node scripts/gen-batch5.mjs
```

Checks: each scene's `viewBox` matches its declared canvas size, no
hardcoded root width/height, no bare hex outside a `var(..., #hex)`
fallback, no unreferenced id, at most two distinct colours per file, and the
6KB illustration budget. SVGO round-trip verified separately
(`svgo.config.mjs`): viewBox and `currentColor` survive on all 30.

## packages/patterns/  (Batch 6 — complete)

Six seamless pattern tiles, paper grain (SVG + PNG fallback), three section
background treatments, three card surfaces, skeleton shimmer, the
provenance page backdrop, and the QR frame.

- `src/pattern-ajrakh.svg`, `pattern-bagru.svg`, `pattern-dabu.svg`,
  `pattern-kolam-dotgrid.svg`, `pattern-warp-weft.svg`,
  `pattern-jaali-fill.svg` — all under 2KB, each with a recommended opacity
  range in its own comment. The jaali and warp-weft tiles here are a
  **different job** from Batch 3's: Batch 3 built a horizontal jaali
  *divider* band and a divider-scale weft *line* texture; these are 2D
  *fill* patterns for panels and page backgrounds, so they're new files,
  not re-exports.
- `src/paper-grain.svg` — feTurbulence, tuned low. `src/paper-grain.png` —
  a real fallback, generated with a from-scratch PNG encoder using only
  Node's built-in `zlib` (no new dependency for one small raster).
- `src/qr-frame.svg` — jaali surround for the provenance QR. Reserves 14%
  of the frame on each side as quiet zone (comfortably over the ISO/IEC
  18004 4-module minimum at this system's typical QR module count) — **not
  verified against a real phone scan**, per the prompt pack's own warning
  to test a printed tag before week three.
- `patterns.css` — pattern-tile masks, section backgrounds
  (`khadi-plain`/`khadi-weft`/`indigo-panel`, indigo-panel measured at
  11.50:1), card surfaces (`flat`/`hairline`; `printed` reuses Batch 3's
  `k-card--printed` rather than duplicating it), skeleton shimmer with a
  reduced-motion static fallback, and the QR-frame slot.
- `provenance.css` — standalone stylesheet for the Go-templated public
  verification page (no Svelte, no build step — just this file plus
  `palette.css` and the referenced SVGs served as static assets).
- `Section.svelte`, `Card.svelte`, `Skeleton.svelte`, `QRFrame.svelte` +
  `index.js`.
- `contact-sheet-batch6.html` — 4-tile seam strips for all six patterns,
  opacity studies against real body text, the paper grain (SVG vs. PNG),
  all three section backgrounds, all three card variants, the skeleton
  shimmer (animated + reduced-motion), and the QR frame with its quiet
  zone dimensioned — light/dark/high-contrast.

### The mask rule, a third time

`patterns.css`'s tile masks and `provenance.css`'s backdrop both use the
same CSS mask over `background-color: currentColor` that Batch 3's
`ornament.css` and Batch 5's `illustrations.css` use — never
`background-image`. Said once more here because it's the mistake most
likely to get "simplified" back in by someone who hasn't read the other
two files' header comments.

### Regenerating

```
node scripts/gen-batch6.mjs
```

Checks: viewBox present, no hardcoded root width/height, no bare hex
outside a `var(..., #hex)` fallback, every id referenced, per-tile size
budget, and 2D seam parity (both axes, not just X — Batch 3's dividers only
needed X-axis parity; these tile in two dimensions) on every 2D pattern.
`paper-grain.png` is checked against its own 3KB budget. SVGO round-trip
verified separately (`svgo.config.mjs`): viewBox and `currentColor` survive
on all 8 SVG assets.

## packages/motion/  (Batch 7 — complete)

Motion and state assets. The charkha spinner (Batch 4) and skeleton
shimmer (Batch 6) already satisfy two of this batch's deliverables and are
referenced here, not rebuilt.

- `src/pipeline-{enhance,attributes,describe,translate}.svg` — the four ML
  pipeline stage glyphs. Per-stage state (pending/active/complete/failed)
  is drawn entirely by `motion.css` around each glyph via a `data-state`
  attribute — the SVGs never change, so there's one file per stage rather
  than a 4×4 combinatorial sprite.
- `src/sync-pending.svg`, `sync-syncing.svg`, `sync-synced.svg` — three new
  sync-state marks; `offline` reuses Batch 4's `icons/src/offline.svg`
  rather than a fourth new file. `sync-syncing` is a thread that draws
  itself in (`stroke-dashoffset`, looping); `sync-synced` is a completed
  stitch, static.
- `src/failure-thread.svg` — a broken thread, two ends out of alignment.
  `sync-synced.svg` above doubles as the general "success" moment — one
  stitch mark, two names, rather than two near-identical files to keep in
  sync with each other.
- `src/seal-draw.svg` — a stroke-only reconstruction of `identity/seal.svg`
  purpose-built for the one-time sealing animation: the real seal is
  fill-based, which `stroke-dashoffset` cannot animate, so this outline
  draws on (400ms, the UI-feedback cap) and `SealAnimation.svelte`
  crossfades to the real filled seal once it completes.
- `motion.css` — every `@keyframes` block and `transition` in this file
  animates only `transform`, `opacity`, `stroke-dashoffset`, or
  `stroke-dasharray` — checked mechanically on every generator run, not
  just by eye.
- `PipelineProgress.svelte`, `SyncState.svelte`, `Moment.svelte`,
  `SealAnimation.svelte`, `VoiceWaveform.svelte` + `index.js`.
- `MOTION-SPEC.md` — durations, easings, the reduced-motion rule for every
  animation in the *whole system* (not just this package), and why
  `stroke-dashoffset` is on the safe-properties allowlist but isn't
  actually free the way `transform`/`opacity` are (it repaints).
- `contact-sheet-batch7.html` — **live**, not a static render: every
  animation actually plays, with buttons to advance the pipeline, replay
  the success/failure moments, play the seal, and start a real
  `AnalyserNode`-driven demo tone for the waveform, plus a page-wide toggle
  that forces the `prefers-reduced-motion` CSS path for everything at once.

### Two real bugs the live sheet caught that a static render wouldn't have

- The `motion.css` self-check (extracting `@keyframes` bodies to verify
  only safe properties animate) initially used a regex that broke on
  single-line `@keyframes` rules (`k-fade-in`, `k-fade-out`) and produced
  false failures — replaced with a brace-counting extractor.
- `.k-waveform__bar` had no explicit `height`, so every bar collapsed to
  0px regardless of its `scaleY` transform — invisible in a description,
  only caught by actually starting the demo tone and looking. Fixed with
  `height: 100%` on the bar, scaled down by the transform as designed.

### Voice waveform, specifically

`VoiceWaveform.svelte` takes a real Web Audio `AnalyserNode` — never a
synthetic loop. Its reduced-motion path is not just "turn the animation
off": a silent static bar during live recording would tell the user
nothing, so it keeps sampling the real analyser (throttled to 4Hz) and
renders the level as a number instead. See `MOTION-SPEC.md`'s reduced-motion
section for why this one component doesn't follow the same "freeze at
a static frame" pattern as everything else.

### Regenerating

```
node scripts/gen-batch7.mjs
```

Checks: viewBox, no hardcoded root width/height, no bare hex outside a
`var(..., #hex)` fallback, no id, no SMIL, 2KB per-asset budget,
`motion.css`'s animated-property allowlist (both `@keyframes` and
`transition`), and a grep of `PipelineProgress.svelte` for
`setTimeout`/`setInterval` (comments stripped first, after an early false
positive from the component's own explanatory header). SVGO round-trip
verified separately: viewBox and `currentColor` survive on all 9 SVGs.

## Batch 8 — component system, optimisation, delivery

The closing batch. No new visual package — it validates, proves, and
packages the 167 SVGs the first seven batches already built.

**Whole-tree validation** — `scripts/validate-all.mjs` walks every
`packages/*/src` directory (not just newly-generated files, the whole
shipped tree) and checks viewBox, no hardcoded root width/height, no bare
hex outside a `var(..., #hex)` fallback, no unreferenced `id`, no SMIL, no
literal `--` inside an XML comment (the exact way Batch 6's SVGO failure
slipped past every earlier check), plus per-category rules: icons locked to
`viewBox="0 0 24 24"` and `stroke-width="1.5"` with round cap/join,
illustrations capped at 2 colours, and a byte budget per category (icon
2KB / pattern 2KB / ornament 2KB / motion 2KB / illustration 6KB / identity
8KB). A short, explicit `EXCEPTIONS` map documents the handful of files
that legitimately break a rule — platform-chrome icons (`favicon.svg`,
`app-icon-maskable*.svg`, `apple-touch-icon.svg`) that render outside any
CSS context and so correctly use bare hex; `ornament/src/patterns.svg`, a
sprite-defs container with no standalone render and ids consumed from
other files' CSS; and four full-lockup identity compositions
(`wordmark-horizontal`, `wordmark-stacked`, `heritage-crest-fine`,
`emblem-lockup`) that are real multi-glyph artwork over the single-mark
8KB budget, already SVGO-optimised to 2-decimal precision — rather than
silently raising the budget for the whole package. Running it also caught
and fixed real bugs in the older Batch 2 identity files: unoptimised path
precision (SVGO brought `emblem-lockup.svg` from 18.4KB to 12.2KB) and an
`aria-labelledby` check that only matched a single id, not the
space-separated list identity's title+desc pattern actually uses.

```
node scripts/validate-all.mjs
```

**Two more bugs found auditing Batch 8 against the original brief, both
fixed:**

- Running that first SVGO pass on `packages/identity/src/` was also the
  first time `svgo.config.mjs` was ever applied to those 14 hand-authored
  files — and the config had never disabled `removeComments`, so the pass
  silently deleted every comment in the package, including the State
  Emblem of India legal notice on `emblem-lockup.svg` (Batch 0's mandatory
  tradition/use comment, and Batch 2's explicit legal requirement). Fixed
  in two places: `removeComments: false` added to `svgo.config.mjs` so it
  can't happen again, and all 14 comments reconstructed by hand — the
  legal notice included, verbatim to what Batch 2's brief required — since
  the deleted originals weren't recoverable. Verified the config fix holds
  by re-running SVGO against the restored files: all 14 comments survive.
- `icons.css`'s stroke-weight swap (`k-icon--dense`/`k-icon--bold`) never
  actually applied: its read-back selector was `.k-icon svg
  [stroke="currentColor"]`, which requires `.k-icon` to be an *ancestor*
  of a nested `<svg>` — but `Icon.svelte` puts the `k-icon` class directly
  on the `<svg>` root itself (vite-plugin-svelte-svg's output has no
  wrapper element), so the selector never matched anything and every icon
  silently rendered at its literal `stroke-width="1.5"` regardless of the
  modifier class. Confirmed with a live DOM test
  (`getComputedStyle(path).strokeWidth` stayed `1.5px` under
  `k-icon--bold`) before and after. Fixed by adding a same-element variant
  of the selector, `.k-icon [stroke="currentColor"]`; re-tested and dense
  (1.25px) / default (1.5px) / bold (2px) all now compute correctly.
  Also: neither `Icon.svelte` nor `Illustration.svelte` had the literal
  `size`/`strokeWidth` props Batch 8's own brief asks for ("consistent
  props: size, strokeWidth, title, class") — both existed only as
  CSS classes (`k-icon--bold`) and custom properties, not component props.
  Added `size` (+ `strokeWidth` on `Icon`, and `class` passthrough on
  `Illustration`, which was missing it entirely) as thin wrappers over the
  same CSS custom properties — the underlying mechanism didn't change, just
  how a caller reaches it.

**Tree-shaking proof** — `scripts/treeshake/` bundles two esbuild entries
(`--loader:.svg=text`, minified): one importing `icons/src/search.svg`
directly, one importing the full `ICON_COMPONENTS` barrel from
`icons/icons.js`. Direct import: 584B. Barrel: 69,253B for all 89 icons —
expected, since the barrel is a live object reference to every icon and no
bundler can shake it. Recorded with the honest scope caveat in
`scripts/treeshake/README.md`: this measures the ES module graph through
esbuild's text loader, not a Svelte-compiled build — `vite-plugin-svelte-svg`
compiles each `.svg` to a `.svelte` component before Rollup ever sees it,
and that step isn't exercised here.

**Print-ready provenance tag** — `packages/print/src/tag-a4-colour.svg` and
`tag-a4-mono.svg`: an A4 sheet, 8 tags (90x54mm, 2x4 grid) with corner crop
marks, seal + QR slot + blank Artisan/Craft fields per tag. This is the one
package that deliberately breaks the currentColor/no-hardcoded-size rules
every other package enforces — see `packages/print/README.md` for why a
print pipeline needs literal mm and explicit hex. QR quiet-zone geometry
and its not-yet-scan-verified caveat are reused from `qr-frame.svg`
verbatim, restated on the tag itself.

```
node scripts/gen-batch8-print.mjs
```

**Go-handoff proof** — `provenance-demo.html` at the repo root renders
`patterns/provenance.css` completely standalone, no build step, every
asset reached by the same root-relative `/assets/...` paths the stylesheet
itself hardcodes. This verifies the CSS renders correctly outside the
Svelte app; it does not verify pixel parity against a Svelte-rendered
equivalent, since no such render exists in this project to diff against —
stated as a partial verification, not claimed as full.

**Documentation site** — `docs/index.html` + `docs/manifest.json`
(generated by `scripts/gen-docs.mjs`) is a static, build-free, searchable
catalogue of all 167 assets: name, package, byte size, a copy-to-clipboard
import line, and a "blurb" reused directly from each asset's own top XML
comment rather than re-authored — every generator already writes a
tradition/use comment per this project's own Batch 0 rule, so the doc site
just surfaces it.

**Full index** — `contact-sheet-index.html` links the seven existing
per-batch contact sheets (Batches 1-7) plus the Batch 8 deliverables above.
It is an index, not an eighth rendered sheet — Batch 8 has no new visual
package to catalogue. `docs/pitch-summary.html` is the one-page visual
summary.

## Still to come
Nothing — all 8 batches are complete.
