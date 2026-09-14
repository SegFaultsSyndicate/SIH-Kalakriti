<!-- apps/artisan/src/routes/register/pehchan/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { FieldGroup, Input, Button } from '@kalakriti/ui';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft } from '$lib/registration';

  const t = $derived(locale.t);

  let pehchanId = $state('');

  $effect(() => {
    void getDraft().then((draft) => {
      pehchanId = draft.pehchanId ?? '';
    });
  });

  function onchange(): void {
    void patchDraft({ pehchanId: pehchanId.trim() || undefined });
  }

  async function next(): Promise<void> {
    await goto('/register/cluster');
  }
</script>

<svelte:head>
  <title>{t('register.pehchan.heading')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep
  index={3}
  heading={t('register.pehchan.heading')}
  speakText={`${t('register.pehchan.heading')}. ${t('register.pehchan.body')}`}
  backHref="/register/district"
>
  {#snippet children()}
    <p class="hint">{t('register.pehchan.body')}</p>
    <FieldGroup label={t('register.pehchan.label')} optional>
      {#snippet children({ id })}
        <Input {id} bind:value={pehchanId} onchange={onchange} />
      {/snippet}
    </FieldGroup>
  {/snippet}
  {#snippet actions()}
    <Button size="xl" onclick={next} tooltip={tooltip(pehchanId.trim() === '' ? 'tooltip.skip' : 'tooltip.next')}>
      {pehchanId.trim() === '' ? t('action.skip') : t('action.next')}
    </Button>
  {/snippet}
</RegisterStep>

<style>
  .hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
