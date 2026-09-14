<!-- apps/artisan/src/routes/register/cluster/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { FieldGroup, Input, Button } from '@kalakriti/ui';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft, submitRegistration } from '$lib/registration';

  const t = $derived(locale.t);

  let clusterName = $state('');
  let submitting = $state(false);

  $effect(() => {
    void getDraft().then((draft) => {
      clusterName = draft.clusterName ?? '';
    });
  });

  function onchange(): void {
    void patchDraft({ clusterName: clusterName.trim() || undefined });
  }

  async function finish(): Promise<void> {
    submitting = true;
    try {
      const draft = await getDraft();
      await submitRegistration(draft, locale.code);
      await goto('/');
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>{t('register.cluster.heading')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep
  index={4}
  heading={t('register.cluster.heading')}
  speakText={`${t('register.cluster.heading')}. ${t('register.cluster.body')}`}
  backHref="/register/pehchan"
>
  {#snippet children()}
    <p class="hint">{t('register.cluster.body')}</p>
    <FieldGroup label={t('register.cluster.label')} optional>
      {#snippet children({ id })}
        <Input {id} bind:value={clusterName} onchange={onchange} />
      {/snippet}
    </FieldGroup>
  {/snippet}
  {#snippet actions()}
    <Button size="xl" onclick={finish} loading={submitting} tooltip={tooltip('tooltip.submitRegistration')}>{t('register.submit')}</Button>
  {/snippet}
</RegisterStep>

<style>
  .hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
