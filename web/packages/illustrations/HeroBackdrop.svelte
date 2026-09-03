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
  export let element = 'div';
</script>

<svelte:element this={element} class="k-hero-backdrop" {...$$restProps}>
  <slot />
</svelte:element>
