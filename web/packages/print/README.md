# Kalakriti print — Batch 8

The provenance product tag as physical print artwork: `src/tag-a4-colour.svg`
and `src/tag-a4-mono.svg`, each an A4 sheet (210mm x 297mm) of 8 tags
(90mm x 54mm, 2x4 grid) with corner crop marks for trimming. Regenerate with
`node scripts/gen-batch8-print.mjs` from `web/` (the script lives at
`web/scripts/gen-batch8-print.mjs`, not a repo-root `scripts/` — that
directory holds backend dev/seed scripts, not asset tooling).

## Why this package breaks two rules every other package enforces

Every other asset in this system is consumed by a browser: `currentColor`
and `var(--k-token, #fallback)` resolve because there is a CSS cascade to
resolve them against, and the root `<svg>` has no hardcoded width/height
because CSS sizes it. A print sheet has neither. It is opened directly by a
printer's RIP, or by a person in Illustrator/Inkscape, with no CSS engine
in the loop at all. So, only here:

- The root `<svg>` carries literal `width="210mm" height="297mm"` — the
  physical sheet size a printer needs, matched 1:1 to `viewBox="0 0 210 297"`
  so 1 SVG user unit = 1mm throughout.
- Colours are explicit brand hex, not `var()`: `#241E1A` (ink-900),
  `#FCFAF6` (khadi-50), `#963D14` (terracotta-700) — the same fallback
  values already hardcoded in `identity/src/favicon.svg` and the app-icon
  files, for the same reason (rendered outside any CSS context).
- The seal (`identity/src/seal-print.svg`) and QR-frame
  (`patterns/src/qr-frame.svg`) geometry are inlined and reused as-is; both
  use `fill="currentColor"` / `stroke="currentColor"` internally, so the
  wrapping `<g>` sets both `color` (what `currentColor` actually resolves
  against) and a matching `fill`/`stroke` as a static-viewer fallback.

`validate-all.mjs` does not scan `packages/print/` — its rules assume a
CSS-consuming document, which this deliberately isn't.

## Fields

`Artisan` and `Craft` are blank ruled lines — filled by hand or by
variable-data printing, not by this generator. The QR slot is empty
geometry only; the backend fills the actual QR bitmap at print time (see
[provenance.css](../patterns/provenance.css) for the paired digital
verification page).

**Not verified against a real printed-and-scanned tag.** The QR quiet-zone
math is inherited from `qr-frame.svg`'s own caveat and repeated on the tag
itself: test a printed tag before relying on it — this environment cannot
print or scan.

## Crop marks

Standard registration marks: two 3mm hairlines per corner, offset 1.5mm
outside the trim edge, not touching the artwork — printer/trimmer reads
these to cut at the tag boundary without a visible mark surviving on the
finished tag.
