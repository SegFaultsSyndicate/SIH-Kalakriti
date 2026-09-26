<!-- apps/artisan/src/routes/register/name/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Input, Button } from '@kalakriti/ui';
  import { listen, listenSupported } from '@kalakriti/voice';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft } from '$lib/registration';

  const t = $derived(locale.t);

  let name = $state('');
  let listening = $state(false);

  $effect(() => {
    void getDraft().then((draft) => {
      name = draft.name ?? '';
    });
  });

  function onchange(): void {
    void patchDraft({ name });
  }

  async function useVoice(): Promise<void> {
    listening = true;
    try {
      const { result } = listen({ tag: locale.meta.tag });
      name = (await result).trim();
      onchange();
    } catch {
      // No mic permission, or recognition failed -- typing still works.
    } finally {
      listening = false;
    }
  }

  async function next(): Promise<void> {
    if (name.trim() === '') return;
    await goto('/register/craft');
  }
</script>

<svelte:head>
  <title>{t('register.name.heading')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep index={0} heading={t('register.name.heading')} backHref="/welcome">
  {#snippet children()}
    <Input
      bind:value={name}
      onchange={onchange}
      placeholder={t('register.name.label')}
      aria-label={t('register.name.label')}
      onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') { e.preventDefault(); void next(); } }}
    />
    {#if listenSupported()}
      <button type="button" class="voice-alt" onclick={useVoice} disabled={listening}>
        {listening ? t('ui.voice.recording') : t('register.name.voice')}
      </button>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={name.trim() === ''} onclick={next} tooltip={tooltip('tooltip.next')}>{t('action.next')} →</Button>
  {/snippet}
</RegisterStep>

<style>
  .voice-alt {
    align-self: start;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: none;
    color: var(--k-text-primary);
    cursor: pointer;
  }
</style>
