<!--
  Kalakriti <Skeleton /> — Batch 6

    <Skeleton width="220px" height="90px" />

  Weft-grid shimmer, not the generic grey sweep. Reduced-motion handling is
  automatic (patterns.css media query) — nothing to pass here for it.
-->
<script>
  export let width = '100%';
  export let height = '1rem';
</script>

<div class="k-skeleton" style="width:{width};height:{height}" aria-hidden="true" {...$$restProps}></div>
