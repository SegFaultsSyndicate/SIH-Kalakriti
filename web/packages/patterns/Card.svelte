<!--
  Kalakriti <Card /> — Batch 6

    <Card>flat, the default</Card>
    <Card variant="hairline">...</Card>
    <Card variant="printed">...</Card>   (Batch 3's k-card--printed)

  No card in this system uses a drop shadow — these three variants replace
  that entirely.
-->
<script>
  /** @type {'flat'|'hairline'|'printed'} */
  export let variant = 'flat';
  export let element = 'div';

  $: cls = variant === 'printed' ? 'k-card--printed' : `k-card k-card--${variant}`;
</script>

<svelte:element this={element} class={cls} {...$$restProps}>
  <slot />
</svelte:element>
