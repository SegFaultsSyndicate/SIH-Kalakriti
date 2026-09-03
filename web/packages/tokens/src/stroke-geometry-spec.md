# Kalakriti stroke and geometry specification

## Icon stroke weights

| Token | Weight | Use |
|---|---|---|
| `--k-icon-stroke-dense` | 1.25 | Dense UI: table rows, compact lists, inline-with-text icons |
| `--k-icon-stroke-default` | 1.5 | Default — nav, buttons, most of the app |
| `--k-icon-stroke-bold` | 2 | The artisan-facing app (larger touch targets, often used outdoors,
often by first-time smartphone users — bolder strokes hold up at
a glance and at low screen brightness) |

All three are drawn on the same 24×24 grid with the same path geometry —
weight is a stroke-width swap, not a redrawn icon. A component that renders
an icon at `bold` must not need a different SVG file.

## Illustration stroke weight

Illustration stroke weight scales with canvas size rather than using a fixed
px value, since illustrations are used at wildly different sizes (a 48px
empty-state icon vs. a 480px hero). Formula: `stroke-width = canvas-width /
120`, clamped to a minimum of 1.5px so it never disappears at small render
sizes. At the two reference sizes this pack uses:

- 240×240 illustration canvas → 2px stroke
- 480×480 illustration canvas → 4px stroke

Use `vector-effect="non-scaling-stroke"` only on illustrations that are
scaled via `transform` after render (e.g., a hover zoom) — not on
illustrations sized purely via their `viewBox`, which already scale
correctly without it.

## Divider and pattern stroke weight

*Added in Batch 3.* Neither of the two rules above covers a tileable ornament:
a divider is not on the 24px icon grid, and `canvas-width / 120` gives a
sub-pixel stroke for a 16px-tall tile. Tileable assets take their own rule:

    stroke-width = tile-height / 20, snapped to the 0.25px grid, minimum 0.75

At the three shipped divider densities:

| Density | Tile height | Stroke |
|---|---|---|
| fine | 16px | 0.75 |
| medium | 24px | 1.25 |
| bold | 32px | 1.5 |

Two consequences follow from this and are load-bearing:

- **Density is not scale.** A tile is redrawn at each density, not scaled. A
  16px tile cannot hold the element count a 32px one can once the stroke has
  to stay above 0.75px, so the Batch 3 dividers escalate: fine is the base
  cell, medium adds a centre aperture, bold adds an inner echo of the cell.
  Scaling one tile to three sizes produces three tiles that are all slightly
  wrong.
- **Anything sitting exactly on the tile's left or right edge renders as a
  half stroke** and is completed by the neighbouring tile's matching half.
  That is the mechanism the seam guarantee rests on, so tile geometry is
  authored inside `[0, W]` and edge-touching geometry is mirrored on both
  edges. `scripts/gen-batch3.mjs` asserts this: what crosses `x=0` must equal
  what crosses `x=W`, and the build fails otherwise.

Fill-led ornament (block print) has no stroke rule — it is stamped shapes with
resist-dye negative space cut by `fill-rule="evenodd"`, and its weight is
carried by the shapes, not by a stroke.

## Corner radius convention

Geometric motifs (jaali, block print, handloom grid) use hard corners
(`stroke-linejoin="miter"`) — these traditions are built from straight
lines and sharp intersections, and rounding them softens the pattern into
something that reads as generic rather than architectural.

Illustrative strokes (Warli figures, charkha, lotus) use
`stroke-linecap="round"` and `stroke-linejoin="round"` — these are drawn,
gestural forms, and round joins are what keep a Warli triangle's apex from
looking like a technical drawing.

UI chrome (cards, buttons, inputs — not motifs) uses a separate
`--k-radius-*` scale, not derived from any motif:

| Token | Value | Use |
|---|---|---|
| `--k-radius-sm` | 2px | Chips, tags |
| `--k-radius-md` | 4px | Buttons, inputs |
| `--k-radius-lg` | 8px | Cards |
| `--k-radius-none` | 0 | Block-print-edged card variant (Batch 6) — the
motif border itself supplies the edge treatment, so the underlying box
stays square |

## Kolam dot-grid unit

Kolam-derived continuous-line patterns snap to a dot grid with **24px
pitch** at 1× (matching the icon grid, so a kolam-derived loading
indicator can sit at icon scale without re-gridding), and **48px pitch**
when used as a larger decorative pattern (patterns batch). The line
itself is drawn at the illustration stroke formula above, never thinner
than 1.5px, so it stays continuous and doesn't fragment into dashes at
small render sizes.

## Optical correction rules

Mathematical centering is not optical centering. Apply these corrections
by eye, then lock the corrected coordinates — don't leave a "centered"
transform in the file that looks off:

- **Circle overshoot.** A circle centered by bounding box on the same
  baseline as a square reads as slightly smaller and higher. Circular
  elements (charkha wheel, kolam dots, the seal) are drawn 2–3% larger in
  diameter than the mathematically equivalent square, and shifted 0.5–1%
  of the canvas size *downward*, to read as equal in weight.
- **Triangle centring.** A triangle's centroid sits below the visual
  center of its bounding box (the apex "weighs" less than the base). Warli
  torso/pelvis triangles are shifted upward by roughly 5% of the
  triangle's height from their mathematical centroid position so the
  figure doesn't read as bottom-heavy.
- **Stroke-weight illusion at intersections.** Where two strokes cross
  (jaali lattice, handloom grid), the crossing point can look thicker than
  either stroke alone. On dense lattices, taper the stroke by roughly 10%
  approaching an intersection rather than holding a constant width through
  it.

  *Batch 3 status: deliberately not applied to the jaali dividers.* Tapering
  needs each stroke expressed as a filled outline rather than a stroked path,
  which multiplies the path data by roughly four and would push every tile
  past its size budget — for a correction that is invisible at the 0.18
  opacity these dividers ship at. It stays the rule for any dense lattice
  drawn at full opacity, and the weft texture sidesteps it entirely by
  drawing only the over-thread at each crossing, so no two threads overlap.
- **24px grid alignment.** Icons align to the pixel grid at 24px, but
  "aligned" means optically balanced within the 20×20 safe area, not every
  anchor point on an integer coordinate — an icon that is pixel-snapped
  everywhere except at one deliberately off-grid point (to fix an optical
  wobble) is correct; pixel-perfect everywhere is not the goal.

## Legibility floor

Every icon must remain legible at 16px. A sixteen-pixel icon can hold
about three strokes' worth of distinguishable detail — if a detail
disappears at 16px in a real render (not a zoomed preview), simplify the
icon rather than accepting the loss. This is tested per-icon on the
contact sheet's real-size row, not assumed from the 24px source.
