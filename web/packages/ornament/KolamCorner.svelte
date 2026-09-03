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

  /** @type {'tl'|'tr'|'bl'|'br'} */
  export let corner = 'tl';
  export let title = undefined;
</script>

<Kolam
  class="k-kolam-corner{corner === 'tl' ? '' : ` k-kolam-corner--${corner}`}"
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...$$restProps}
/>
