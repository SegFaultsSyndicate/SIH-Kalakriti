<!--
  Kalakriti <Card /> — Batch 6

    <Card>flat, the default</Card>
    <Card variant="hairline">...</Card>
    <Card variant="printed">...</Card>   (Batch 3's k-card--printed)

  No card in this system uses a drop shadow — these three variants replace
  that entirely.
-->
<script>
  /**
   * @type {{
   *   variant?: 'flat'|'hairline'|'printed'|'media',
   *   element?: string,
   *   children?: import('svelte').Snippet,
   *   [key: string]: unknown
   * }}
   */
  let { variant = 'flat', element = 'div', children, ...rest } = $props();

  const cls = $derived(
    variant === 'printed' ? 'k-card--printed' : `k-card k-card--${variant}`,
  );
</script>

<svelte:element this={element} class={cls} {...rest}>
  {@render children?.()}
</svelte:element>
