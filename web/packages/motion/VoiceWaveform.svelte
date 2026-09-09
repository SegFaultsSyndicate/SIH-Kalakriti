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
  /**
   * @typedef {object} Props
   * @property {AnalyserNode} [analyser] Live Web Audio analyser. The bars are
   *   driven by real amplitude data -- never a decorative loop, because a
   *   waveform that moves while the microphone is dead is a lie the artisan
   *   only discovers after recording nothing.
   * @property {number} [bars]
   * @property {(percent: number) => string} label Builds the accessible
   *   name from the current level, e.g. `(p) => t('voice.level', { p })`.
   *   From the i18n layer -- this used to be a hardcoded English string.
   */

  /** @type {Props & Record<string, unknown>} */
  let { analyser, bars = 24, label, ...rest } = $props();

  let defaultLevels = $derived(new Array(bars).fill(0.05));
  let liveLevels = $state(/** @type {number[] | null} */ (null));
  let levels = $derived(liveLevels ?? defaultLevels);
  let levelPercent = $state(0);
  let reduced = $state(false);

  function sample() {
    if (!analyser) return;
    const data = new Uint8Array(analyser.frequencyBinCount);
    analyser.getByteFrequencyData(data);
    const step = Math.floor(data.length / bars) || 1;
    liveLevels = Array.from({ length: bars }, (_, i) => Math.max(0.06, data[i * step] / 255));
    levelPercent = Math.round(
      (data.reduce((a, b) => a + b, 0) / data.length / 255) * 100,
    );
  }

  // Track the media query live: a user who turns reduced motion on mid-
  // recording gets the static meter without reloading.
  $effect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    reduced = mq.matches;
    const onChange = (/** @type {MediaQueryListEvent} */ e) => {
      reduced = e.matches;
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  });

  /*
   * One sampling loop, torn down whenever its inputs change. The Svelte 4
   * version started a loop from a reactive statement and never cancelled it,
   * so every change of `analyser` or `reduced` left another
   * requestAnimationFrame chain running for the life of the page.
   */
  $effect(() => {
    if (!analyser) return;

    if (reduced) {
      // 4Hz: enough to prove the meter is live, not a redraw storm.
      const id = setInterval(sample, 250);
      return () => clearInterval(id);
    }

    let frame = requestAnimationFrame(function loop() {
      sample();
      frame = requestAnimationFrame(loop);
    });
    return () => cancelAnimationFrame(frame);
  });
</script>

{#if reduced}
  <div class="k-waveform k-waveform--reduced" role="img" aria-label={label(levelPercent)} {...rest}>
    <div class="k-waveform__bar" style="--k-bar-level:{Math.max(0.06, levelPercent / 100)}"></div>
    <span class="k-waveform__level-text">{levelPercent}%</span>
  </div>
{:else}
  <div class="k-waveform" role="img" aria-label={label(levelPercent)} {...rest}>
    {#each levels as level, i (i)}
      <div class="k-waveform__bar" style="--k-bar-level:{level}"></div>
    {/each}
  </div>
{/if}
