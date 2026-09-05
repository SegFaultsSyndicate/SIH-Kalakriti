<!--
  packages/ui/src/AudioPlayback.svelte

    <AudioPlayback src={ttsClipUrl} label="Listen to this screen" />

  Reads a TTS clip aloud. A real <audio> element drives playback and time;
  this component only supplies the visible progress and the two controls
  the brief needs (play/pause, replay) rather than the browser's own
  control strip, which does not match the design system and is not
  reliably keyboard-consistent across browsers.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    src: string;
    /** What is being read aloud, e.g. "Listing description". Names the control for a screen reader. */
    label: string;
    class?: string;
  }

  let { src, label, class: className }: Props = $props();

  const t = $derived(locale.t);

  let audio: HTMLAudioElement;
  let playing = $state(false);
  let currentTime = $state(0);
  let duration = $state(0);

  /** A live-streamed or malformed clip can report Infinity or NaN, neither of
   *  which a <progress> element or a "m:ss" string can render. */
  function finite(value: number): number {
    return Number.isFinite(value) ? value : 0;
  }

  function format(seconds: number): string {
    const total = Math.max(0, Math.round(finite(seconds)));
    const m = Math.floor(total / 60);
    const s = total % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
  }

  function toggle(): void {
    if (playing) audio.pause();
    else void audio.play();
  }

  function replay(): void {
    audio.currentTime = 0;
    void audio.play();
  }
</script>

<div class="k-audio {className || ''}">
  <audio
    bind:this={audio}
    {src}
    preload="metadata"
    onplay={() => (playing = true)}
    onpause={() => (playing = false)}
    onended={() => (playing = false)}
    onloadedmetadata={() => (duration = audio.duration)}
    ontimeupdate={() => (currentTime = audio.currentTime)}
  ></audio>

  <button
    type="button"
    class="k-audio__toggle"
    onclick={toggle}
    aria-label={`${playing ? t('ui.audio.pause') : t('ui.audio.play')}: ${label}`}
  >
    <Icon name={playing ? 'pause' : 'play'} />
  </button>

  <div class="k-audio__body">
    <progress class="k-audio__progress" value={finite(currentTime)} max={finite(duration) || 1}
    ></progress>
    <span class="k-audio__time">
      {t('ui.audio.progress', { elapsed: format(currentTime), total: format(duration) })}
    </span>
  </div>

  <button type="button" class="k-audio__replay" onclick={replay} aria-label={t('ui.audio.replay')}>
    <Icon name="refresh" />
  </button>
</div>
