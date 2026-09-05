<!--
  packages/voice/src/ReadScreen.svelte

    Mounted once, persistently, in the artisan app's root layout:
    <ReadScreen />

  The global "read this screen" control. Reads buildReadingOrder()'s output
  (headings, then labels and values) one utterance at a time via speak(),
  awaiting each before starting the next so a slow phone doesn't overlap two
  utterances. If no voice matches the active locale, this says so instead of
  silently doing nothing when pressed -- see speak.ts's voiceAvailable().
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { speak, stopSpeaking, voiceAvailable } from './speak';
  import { buildReadingOrder } from './screen-reader';

  interface Props {
    containerSelector?: string;
  }

  let { containerSelector = '#main-content' }: Props = $props();

  const t = $derived(locale.t);

  let reading = $state(false);
  let unavailable = $state(false);
  let stopRequested = false;

  async function toggle(): Promise<void> {
    if (reading) {
      stopRequested = true;
      stopSpeaking();
      reading = false;
      return;
    }

    unavailable = false;
    if (!voiceAvailable(locale.meta.tag)) {
      unavailable = true;
      return;
    }

    const root = document.querySelector(containerSelector);
    if (!root) return;

    reading = true;
    stopRequested = false;
    for (const part of buildReadingOrder(root)) {
      if (stopRequested) break;
      try {
        await speak(part, { tag: locale.meta.tag });
      } catch {
        break;
      }
    }
    reading = false;
  }
</script>

<button type="button" class="k-read-screen" onclick={toggle} aria-pressed={reading}>
  {reading ? t('voice.readScreen.stop') : t('voice.readScreen')}
</button>

{#if unavailable}
  <p class="k-read-screen__unavailable" role="status">
    {t('voice.unavailable', { language: locale.meta.endonym })}
  </p>
{/if}

<style>
  .k-read-screen {
    position: fixed;
    inset-block-end: var(--k-space-4);
    inset-inline-end: var(--k-space-4);
    z-index: var(--k-z-sticky);
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .k-read-screen[aria-pressed='true'] {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
  }

  .k-read-screen__unavailable {
    position: fixed;
    inset-block-end: calc(var(--k-space-4) + var(--k-touch-min) + var(--k-space-2));
    inset-inline-end: var(--k-space-4);
    max-inline-size: 16rem;
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
