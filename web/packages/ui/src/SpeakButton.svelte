<!--
  packages/ui/src/SpeakButton.svelte

    <SpeakButton text={t('welcome.body')} label={t('welcome.listen')} />
    <SpeakButton text={meta.endonym} tag={meta.tag} label={t('language.speak')} iconOnly />

  The "offer audio, do not auto-play" control: every spoken prompt in the
  onboarding flow (Batch 7) is behind one of these rather than a mounted
  speak() call, per the do-not on auto-playing audio without a gesture.

  Speech is sent to the browser with the requested locale even when its voice
  list does not advertise that locale; some engines can still resolve it.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { speak, stopSpeaking } from '@kalakriti/voice';

  interface Props {
    /** The string spoken aloud. Never the raw i18n key -- pass t(key). */
    text: string;
    /** BCP 47 tag. Defaults to the active locale's -- pass a specific tile's own tag on /language, where nothing is active yet. */
    tag?: string;
    /** Accessible name, and the visible label unless iconOnly. */
    label: string;
    iconOnly?: boolean;
    class?: string;
  }

  let { text, tag, label, iconOnly = false, class: className }: Props = $props();

  const t = $derived(locale.t);
  const effectiveTag = $derived(tag ?? locale.meta.tag);

  let speaking = $state(false);
  let unavailable = $state(false);

  async function toggle(): Promise<void> {
    if (speaking) {
      stopSpeaking();
      speaking = false;
      return;
    }
    unavailable = false;
    speaking = true;
    try {
      await speak(text, { tag: effectiveTag });
    } catch (error) {
      if (!(error instanceof Error) || !['canceled', 'interrupted'].includes(error.message)) {
        unavailable = true;
      }
    } finally {
      speaking = false;
    }
  }
</script>

<button
  type="button"
  class="k-speak {className || ''}"
  class:k-speak--icon-only={iconOnly}
  aria-pressed={speaking}
  aria-label={iconOnly ? label : undefined}
  onclick={toggle}
>
  <Icon name={speaking ? 'volume' : 'speaker'} class="k-speak__icon" />
  {#if !iconOnly}
    <span>{label}</span>
  {/if}
</button>

{#if unavailable}
  <p class="k-speak__unavailable" role="status">
    {t('voice.unavailable', { language: label })}
  </p>
{/if}

<style>
  .k-speak {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    min-block-size: var(--k-touch-min);
    min-inline-size: var(--k-touch-min);
    padding-inline: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-md);
    cursor: pointer;
  }

  .k-speak--icon-only {
    padding: 0;
    justify-content: center;
  }

  .k-speak[aria-pressed='true'] {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
  }

  .k-speak__unavailable {
    margin-block-start: var(--k-space-2);
    max-inline-size: 24rem;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
