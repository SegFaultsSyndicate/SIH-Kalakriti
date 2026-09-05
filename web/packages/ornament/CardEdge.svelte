<!--
  Kalakriti <CardEdge /> — Batch 3

  Wraps content in a block-printed frame instead of a drop shadow.

    <CardEdge>...card content...</CardEdge>
    <CardEdge element="article" class="my-card">...</CardEdge>

  The frame is card-edge.svg applied as a 9-slice border-image (see
  .k-card--printed in ornament.css). This is the one Batch 3 asset that
  cannot use currentColor: a border-image loads in an isolated document
  where currentColor resolves to black and the --k-* tokens are invisible,
  so the colour is baked per theme in the stylesheet. The consequence is
  that the card edge follows the theme, not the element's text colour.

  If a dynamic edge colour is ever needed, the upgrade is
  -webkit-mask-box-image on a ::before over background-color: currentColor,
  with the current border-image kept as the Firefox fallback.

  Requires: import '@kalakriti/ornament/ornament.css'
-->
<script>
  /**
   * @typedef {object} Props
   * @property {string} [element] Tag to render. Use 'article'/'li' where the
   *   card is semantic.
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props & Record<string, unknown>} */
  let { element = 'div', children, ...rest } = $props();
</script>

<svelte:element this={element} class="k-card--printed" {...rest}>
  {@render children?.()}
</svelte:element>
