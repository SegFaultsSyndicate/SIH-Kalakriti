<!--
  Kalakriti <Skeleton /> — Batch 6, `shape` presets added in the frontend
  track's Batch 3 (packages/ui).

    <Skeleton width="220px" height="90px" />
    <Skeleton shape="text" />           one line of body text
    <Skeleton shape="media" />          16:9 block, e.g. a photo placeholder
    <Skeleton shape="card" />           a card-sized block, k-radius-lg

  `shape` sets a sensible default width/height/radius for the three things
  a loading screen in this system actually waits on; pass width/height
  directly to override either without abandoning the shape's radius.

  Weft-grid shimmer, not the generic grey sweep. Reduced-motion handling is
  automatic (patterns.css media query) — nothing to pass here for it.
-->
<script>
  const SHAPE_DEFAULTS = {
    text: { width: '100%', height: '1em', radius: 'var(--k-radius-sm, 2px)' },
    media: { width: '100%', height: '0', radius: 'var(--k-radius-md, 4px)' },
    card: { width: '100%', height: '12rem', radius: 'var(--k-radius-lg, 8px)' },
  };

  /**
   * @typedef {object} Props
   * @property {'text'|'media'|'card'} [shape]
   * @property {string} [width]
   * @property {string} [height]
   */

  /** @type {Props & Record<string, unknown>} */
  let { shape, width, height, ...rest } = $props();

  const defaults = $derived(shape ? SHAPE_DEFAULTS[shape] : undefined);
  const resolvedWidth = $derived(width ?? defaults?.width ?? '100%');
  const resolvedHeight = $derived(height ?? defaults?.height ?? '1rem');
  const aspect = $derived(shape === 'media' ? 'aspect-ratio:16/9;' : '');
  const radius = $derived(defaults?.radius);
</script>

<div
  class="k-skeleton"
  style="width:{resolvedWidth};height:{resolvedHeight};{aspect}{radius
    ? `border-radius:${radius};`
    : ''}"
  aria-hidden="true"
  {...rest}
></div>
