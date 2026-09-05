<!--
  Kalakriti <KolamCorner /> — Batch 3

    <KolamCorner />              top-left (the drawn orientation)
    <KolamCorner corner="tr" />  mirrored
    <KolamCorner corner="br" />  rotated 180
    <KolamCorner corner="bl" />  flipped

  One asset, four rotations, per the batch brief. The transforms live in
  ornament.css so the SVG file stays a single source of truth.

  Decorative by default. Pass a title only if the corner is genuinely
  carrying meaning somewhere, which it should not be.
-->
<script>
  import Kolam from './src/kolam-corner.svg';

  /**
   * @typedef {object} Props
   * @property {'tl'|'tr'|'bl'|'br'} [corner]
   * @property {string} [title] Accessible name. Omit for decorative use,
   *   which is what a corner ornament almost always is.
   */

  /** @type {Props & Record<string, unknown>} */
  let { corner = 'tl', title, ...rest } = $props();
</script>

<Kolam
  class={corner === 'tl' ? 'k-kolam-corner' : `k-kolam-corner k-kolam-corner--${corner}`}
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...rest}
/>
