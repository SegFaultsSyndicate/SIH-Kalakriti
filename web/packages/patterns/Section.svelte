<!--
  Kalakriti <Section /> — Batch 6

    <Section variant="khadi-plain">...</Section>
    <Section variant="khadi-weft">...</Section>
    <Section variant="indigo-panel">
      <a href="/orders">An order link</a>   (gets the terracotta-300 accent automatically)
    </Section>

  One of the three alternating page-section treatments, so a long page (the
  buyer home) doesn't read as one flat scroll. indigo-panel remaps text
  colour itself — do not add k-text-primary inside it, the section already
  carries the correct (measured) pairing.
-->
<script>
  /**
   * @typedef {object} Props
   * @property {'khadi-plain'|'khadi-weft'|'indigo-panel'} [variant]
   * @property {string} [element]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props & Record<string, unknown>} */
  let { variant = 'khadi-plain', element = 'section', children, ...rest } = $props();
</script>

<svelte:element
  this={element}
  class="k-section--{variant} {variant === 'khadi-weft' ? 'k-pattern' : ''}"
  {...rest}
>
  {@render children?.()}
</svelte:element>
