<!--
  Kalakriti <VoiceWaveform /> — Batch 7

    const audioCtx = new AudioContext();
    const analyser = audioCtx.createAnalyser();
    analyser.fftSize = 64;
    micStream.connect(analyser);

    <VoiceWaveform {analyser} />

  Driven by a real Web Audio AnalyserNode — never a synthetic loop. Reads
  getByteFrequencyData on each animation frame and maps it to bar heights
  via CSS custom properties (transform: scaleY, not width/height, so the
  60fps update never touches layout).

  reduced-motion: per the brief, a STATIC bar chart is not enough on its
  own — it would tell a user nothing about whether recording is live. The
  reduced-motion branch below still samples the analyser (at a throttled
  4Hz instead of every frame) and shows the current level as a number, so
  the "recording is live" signal survives even with animation off.
-->
<script>
  import { onDestroy } from 'svelte';

  /** @type {AnalyserNode | undefined} */
  export let analyser = undefined;
  export let bars = 24;

  let levels = new Array(bars).fill(0.05);
  let levelPercent = 0;
  let rafId;
  let intervalId;
  let reduced = false;

  if (typeof window !== 'undefined' && window.matchMedia) {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    reduced = mq.matches;
    mq.addEventListener?.('change', (e) => { reduced = e.matches; });
  }

  function sample() {
    if (!analyser) return 0;
    const data = new Uint8Array(analyser.frequencyBinCount);
    analyser.getByteFrequencyData(data);
    const step = Math.floor(data.length / bars) || 1;
    levels = Array.from({ length: bars }, (_, i) => {
      const v = data[i * step] / 255;
      return Math.max(0.06, v);
    });
    levelPercent = Math.round((data.reduce((a, b) => a + b, 0) / data.length / 255) * 100);
    return levelPercent;
  }

  $: if (analyser && !reduced) {
    const loop = () => { sample(); rafId = requestAnimationFrame(loop); };
    rafId = requestAnimationFrame(loop);
  }
  $: if (analyser && reduced) {
    intervalId = setInterval(sample, 250); // 4Hz — enough to prove it's live, not a redraw storm
  }

  onDestroy(() => {
    if (rafId) cancelAnimationFrame(rafId);
    if (intervalId) clearInterval(intervalId);
  });
</script>

{#if reduced}
  <div class="k-waveform k-waveform--reduced" role="img" aria-label="Recording level {levelPercent} percent">
    <div class="k-waveform__bar" style="--k-bar-level:{Math.max(0.06, levelPercent / 100)}"></div>
    <span class="k-waveform__level-text">{levelPercent}%</span>
  </div>
{:else}
  <div class="k-waveform" role="img" aria-label="Recording level {levelPercent} percent">
    {#each levels as level}
      <div class="k-waveform__bar" style="--k-bar-level:{level}"></div>
    {/each}
  </div>
{/if}
