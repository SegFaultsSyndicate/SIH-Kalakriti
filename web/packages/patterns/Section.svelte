<!--
  Kalakriti <Section /> — Batch 6

    <Section variant="khadi-plain">...</Section>
    <Section variant="khadi-weft">...</Section>
    <Section variant="indigo-panel">
      <a href="/orders">An order link</a>   <!-- gets the terracotta-300 accent automatically -->
    </Section>

  One of the three alternating page-section treatments, so a long page (the
  buyer home) doesn't read as one flat scroll. indigo-panel remaps text
  colour itself — do not add k-text-primary inside it, the section already
  carries the correct (measured) pairing.
-->
<script>
  /** @type {'khadi-plain'|'khadi-weft'|'indigo-panel'} */
  export let variant = 'khadi-plain';
  export let element = 'section';
</script>

<svelte:element
  this={element}
  class="k-section--{variant} {variant === 'khadi-weft' ? 'k-pattern' : ''}"
  {...$$restProps}
>
  <slot />
</svelte:element>
