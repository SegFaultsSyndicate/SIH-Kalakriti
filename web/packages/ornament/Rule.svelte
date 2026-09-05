<!--
  Kalakriti <Rule /> — Batch 3

    <Rule />                  plain hairline (pure CSS, cannot fail to load)
    <Rule variant="motif" />  hairline interrupted by the warp/weft knot
    <Rule variant="woven" />  double rule with woven-thread infill

  Every variant degrades to a plain 1px line: "plain" has no image at all,
  "motif" draws its line with pseudo-elements and only the knot is an image,
  and "woven" keeps a border-top fallback behind its mask (ornament.css).

  Requires: import '@kalakriti/ornament/ornament.css'
-->
<script>
  import Knot from './src/rule-knot.svg';

  /**
   * @typedef {object} Props
   * @property {'plain'|'motif'|'woven'} [variant]
   */

  /** @type {Props & Record<string, unknown>} */
  let { variant = 'plain', ...rest } = $props();
</script>

{#if variant === 'motif'}
  <div class="k-rule-motif" role="presentation" aria-hidden="true" {...rest}>
    <Knot />
  </div>
{:else if variant === 'woven'}
  <hr class="k-rule k-rule--woven" aria-hidden="true" {...rest} />
{:else}
  <hr class="k-rule" aria-hidden="true" {...rest} />
{/if}
