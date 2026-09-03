# Kalakriti motion specification

Batch 7. Applies to every animation in the system, not only this package —
the charkha spinner (Batch 4) and skeleton shimmer (Batch 6) follow the same
rules and are listed here for completeness.

## Durations and easings

| Animation | Duration | Easing | Loop? |
|---|---|---|---|
| Charkha spinner, indeterminate | 1.1s / rotation | linear | infinite |
| Charkha spinner, reduced-motion pulse | 1.6s | ease-in-out | infinite |
| Skeleton shimmer sweep | 1.6s | ease-in-out | infinite |
| Pipeline station, active pulse | 1.4s | ease-in-out | infinite |
| Pipeline thread, stage completes | 0.4s | ease-out | once |
| Sync state, "syncing" thread draw | 0.9s | ease-out | infinite |
| Success / failure moment | 0.35–0.4s | ease-out | once |
| Seal draw-on | 0.4s | ease-out | once |
| Seal crossfade (draw → real seal) | 0.2s | ease-out | once |
| Voice waveform bar response | 0.06s | linear | continuous, driven by real analyser data |

**400ms is the hard cap for UI feedback** — any animation that confirms or
responds to a user action (a success mark, a failed action, the seal
draw-on). This is a Do-Not from the batch brief, not a guideline: nothing
in this system exceeds it. Where a moment needs to *feel* longer (the seal),
the 400ms draw is followed by a separate, also-short crossfade — the total
perceived sequence is longer than 400ms, but no single feedback animation
is.

Loops (spinners, shimmer, the syncing thread) are exempt from the cap by
nature — they're not confirming an action, they're indicating an ongoing
one, and they keep running until the state changes.

## What animates, and what never does

Every `@keyframes` block and `transition` declaration in this system's CSS
animates only:

- `transform` (translate, scale, rotate)
- `opacity`
- `stroke-dashoffset` / `stroke-dasharray`

Nothing animates `width`, `height`, `top`, `left`, `margin`, or any other
property that triggers layout. `scripts/gen-batch7.mjs` checks this
mechanically against `motion.css` on every run — a PR that adds a
layout-triggering `@keyframes` property fails the build, not just review.

One caveat worth naming: `stroke-dashoffset` is on the allowlist above
because it's the only mechanism for a "line drawing itself" effect (the
seal, the syncing thread), but it is **not** a compositor-only property the
way `transform`/`opacity` are — the browser repaints the path on every
frame. That's fine at this system's scale (short paths, brief durations),
but it is not free, and it's the one exception to "everything here is
cheap" worth knowing before reaching for it somewhere bigger.

## Reduced motion

Every animation in this system has a `prefers-reduced-motion: reduce`
variant. The pattern is consistent:

- **Loops** (spinner, shimmer, pipeline pulse, syncing thread) become a
  static frame, usually at a fixed intermediate opacity so the element still
  reads as "not yet resolved" without motion.
- **One-time moments** (success/failure, seal draw-on) skip straight to
  their end state — the checkmark or broken thread appears, fully formed,
  with no draw-on.
- **The voice waveform is the one case that can't just go static.** A silent
  static bar during live recording tells the user nothing — worse than
  useless, since they'd have no way to tell if the microphone is even
  picking anything up. Its reduced-motion mode keeps sampling the real
  analyser (throttled to 4Hz instead of every frame) and shows the level as
  a number, so the "recording is live" signal survives even with the moving
  bars turned off.

## No animation blocks interaction

Every animated element remains interactive through its own animation — a
button doesn't become unclickable while its success moment plays, a pipeline
station's tooltip works mid-pulse. None of this system's animations run on
the main thread in a way that could block input; they're all
transform/opacity/stroke-dashoffset, which the browser can composite without
racing the event loop.

## Real state, never a fake timer

The ML pipeline progress indicator's per-stage state comes from a `stages`
prop the caller sets from actual backend state (poll, SSE, websocket).
`PipelineProgress.svelte` contains no `setTimeout`/`setInterval` that
advances a stage on its own — `scripts/gen-batch7.mjs` greps the component
source for both on every run and fails the build if either appears. A demo
that wants to *simulate* progress does so from outside the component
(changing the prop on a timer in the demo page, not inside the component),
which keeps the fake-timer risk out of the shipped code path entirely.
