<!--
  Kalakriti <Spinner /> — Batch 4

    <Spinner />                    indeterminate — the whole charkha rotates
    <Spinner progress={0.6} />     determinate — thread winds onto the spindle

  The charkha (spinning wheel) is the primary loading indicator: its literal
  mechanical function maps onto "in progress" without borrowing a religious
  or historical symbol, per motifs.md. Rotation and the reduced-motion pulse
  are CSS (icons.css), not SMIL.
-->
<script>
  import Charkha from './src/charkha-spinner.svg';

  /**
   * @typedef {object} Props
   * @property {number} [progress] 0-1, or omitted for indeterminate. Must
   *   come from real state -- a fake timer here lies about the backend.
   * @property {string} [title]
   */

  /** @type {Props & Record<string, unknown>} */
  let { progress, title = 'Loading', ...rest } = $props();

  const determinate = $derived(progress !== undefined);
</script>

<Charkha
  class="k-icon k-spinner {determinate ? 'k-spinner--determinate' : 'k-spinner--indeterminate'}"
  style={determinate
    ? `--k-spin-progress:${Math.max(0, Math.min(1, /** @type {number} */ (progress)))}`
    : undefined}
  role="img"
  aria-label={title}
  focusable="false"
  {...rest}
/>
