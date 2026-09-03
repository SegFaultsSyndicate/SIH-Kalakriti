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

  /** 0-1, or undefined for indeterminate. Must come from real state. */
  export let progress = undefined;
  export let title = 'Loading';
</script>

<Charkha
  class="k-icon k-spinner {progress === undefined ? 'k-spinner--indeterminate' : 'k-spinner--determinate'}"
  style={progress === undefined ? undefined : `--k-spin-progress:${Math.max(0, Math.min(1, progress))}`}
  role="img"
  aria-label={title}
  focusable="false"
  {...$$restProps}
/>
