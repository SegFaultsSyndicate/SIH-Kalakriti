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
  import { createEventDispatcher, onDestroy } from 'svelte';
  import SealDraw from './src/seal-draw.svg';
  import RealSeal from '../identity/src/seal.svg';

  export let play = false;

  const dispatch = createEventDispatcher();
  let phase = 'idle'; // idle -> drawing -> settled
  let timer;

  $: if (play && phase === 'idle') {
    phase = 'drawing';
    timer = setTimeout(() => {
      phase = 'settled';
      dispatch('done');
    }, 620); // 400ms draw + ~200ms crossfade, see motion.css k-seal-crossfade-*
  }

  onDestroy(() => clearTimeout(timer));
</script>

<div class="k-seal-animation-host" {...$$restProps}>
  {#if phase === 'drawing'}
    <SealDraw class="k-seal-draw" aria-hidden="true" focusable="false" />
  {:else if phase === 'settled'}
    <RealSeal class="k-seal-crossfade-in" role="img" aria-label="Provenance sealed" focusable="false" />
  {/if}
</div>
