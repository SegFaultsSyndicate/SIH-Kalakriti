<!--
  Kalakriti <Divider /> — Batch 3

  Renders one of the tileable ornament bands as a full-width horizontal rule.

    <Divider />                                     jaali-hex, medium
    <Divider variant="jaali-octstar" density="bold" />
    <Divider variant="blockprint-band" />
    <Divider variant="jaali-interlace" opacity={0.3} />

  The tile is applied as a CSS mask over background-color: currentColor, so
  the ornament takes the surrounding text colour. See the header comment in
  ornament.css for why it is a mask and not a background-image — that choice
  is load-bearing, not stylistic.

  Because the mask lives in ornament.css there is no inline <svg> and
  therefore no <pattern> id, so any number of dividers can coexist on one
  page without id collisions.

  Requires: import '@kalakriti/ornament/ornament.css'
-->
<script>
  /** @type {'jaali-hex'|'jaali-octstar'|'jaali-interlace'|'blockprint-running'|'blockprint-band'} */
  export let variant = 'jaali-hex';
  /** @type {'fine'|'medium'|'bold'} — jaali variants only. */
  export let density = 'medium';
  /** Override the default 0.18. Dividers are structure; keep this low. */
  export let opacity = undefined;

  const HEIGHT = {
    'jaali-hex': { fine: 16, medium: 24, bold: 32 },
    'jaali-octstar': { fine: 16, medium: 24, bold: 32 },
    'jaali-interlace': { fine: 16, medium: 24, bold: 32 },
    'blockprint-running': { fine: 20, medium: 20, bold: 20 },
    'blockprint-band': { fine: 32, medium: 32, bold: 32 },
  };

  $: height = HEIGHT[variant][density];
  $: modifier = `k-divider--${variant}-${height}`;
</script>

<div
  class="k-divider {modifier}"
  style={opacity === undefined ? undefined : `opacity:${opacity}`}
  role="presentation"
  aria-hidden="true"
  {...$$restProps}
></div>
