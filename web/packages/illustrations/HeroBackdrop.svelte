<!--
  Kalakriti <HeroBackdrop /> — Batch 5

    <HeroBackdrop>
      <img src="/hero.jpg" alt="..." />
      <h1>...</h1>
    </HeroBackdrop>

  Wraps hero content with the low-contrast jaali+weft tile behind it (never
  on top — see illustrations.css's k-hero-backdrop rule, a currentColor mask
  over the tile, not a background-image).
-->
<script>
  /**
   * @typedef {object} Props
   * @property {string} [element]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props & Record<string, unknown>} */
  let { element = 'div', children, ...rest } = $props();
</script>

<svelte:element this={element} class="k-hero-backdrop" {...rest}>
  {@render children?.()}
</svelte:element>
