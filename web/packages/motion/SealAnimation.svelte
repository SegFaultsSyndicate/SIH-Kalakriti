<!--
  Kalakriti <SealAnimation /> — Batch 7

    <SealAnimation play={justSealed} on:done={() => justSealed = false} />

  Plays once when `play` becomes true: seal-draw.svg's stroke-dasharray
  draws on over 400ms (the UI-feedback cap — see MOTION-SPEC.md), then
  crossfades to the real filled seal (identity/seal.svg) and stays there.
  Set `play` back to false (or just leave it) once the `done` event fires;
  the component does not reset itself, so a re-seal needs the caller to
  toggle `play` false-then-true again.

  This is the one meaningful, rare moment in the product that gets an
  actual flourish — do not reuse this component or its timing for routine
  UI feedback (that's <Moment>, capped at 400ms with no crossfade tail).
-->
<script>
  import SealDraw from './src/seal-draw.svg';
  import RealSeal from '../identity/src/seal.svg';

  /**
   * @typedef {object} Props
   * @property {boolean} [play] Flip to true once, when provenance is
   *   actually sealed. The draw runs a single time; setting it back to
   *   false does not rewind it, because the sealed state is permanent.
   * @property {() => void} [ondone] Called when the seal has settled.
   * @property {string} label Accessible name for the settled seal, from the
   *   i18n layer -- this used to be a hardcoded English string.
   */

  /** @type {Props & Record<string, unknown>} */
  let { play = false, ondone, label, ...rest } = $props();

  /** @type {'idle'|'drawing'|'settled'} */
  let phase = $state('idle');

  // A genuine side effect: a timer, plus its teardown. The 620ms is 400ms of
  // draw plus the ~200ms crossfade in motion.css (k-seal-crossfade-*); the
  // two must move together.
  $effect(() => {
    if (!play || phase !== 'idle') return;
    phase = 'drawing';
    const timer = setTimeout(() => {
      phase = 'settled';
      ondone?.();
    }, 620);
    return () => clearTimeout(timer);
  });
</script>

<div class="k-seal-animation-host" {...rest}>
  {#if phase === 'drawing'}
    <SealDraw class="k-seal-draw" aria-hidden="true" focusable="false" />
  {:else if phase === 'settled'}
    <RealSeal
      class="k-seal-crossfade-in"
      role="img"
      aria-label={label}
      focusable="false"
    />
  {/if}
</div>
