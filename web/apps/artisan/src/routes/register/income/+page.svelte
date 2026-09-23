<!-- apps/artisan/src/routes/register/income/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Button } from '@kalakriti/ui';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import BracketPicker from '$lib/BracketPicker.svelte';
  import { getDraft, patchDraft } from '$lib/registration';

  const t = $derived(locale.t);

  let bracket = $state('');

  $effect(() => {
    void getDraft().then((draft) => {
      bracket = draft.incomeBracket ?? '';
    });
  });
</script>

<svelte:head>
  <title>{t('income.baseline.step')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep
  index={5}
  heading={t('income.baseline.question')}
  speakText={`${t('income.baseline.question')}. ${t('income.baseline.why')}`}
  backHref="/register/social-category"
>
  {#snippet children()}
    <p class="hint">{t('income.baseline.why')}</p>
    <BracketPicker bind:value={bracket} onselect={(code) => void patchDraft({ incomeBracket: code })} />
  {/snippet}
  {#snippet actions()}
    <Button size="xl" onclick={() => goto('/register/cluster')} tooltip={tooltip(bracket ? 'tooltip.next' : 'tooltip.skip')}>
      {bracket ? `${t('action.next')} →` : t('action.skip')}
    </Button>
  {/snippet}
</RegisterStep>

<style>
  .hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    line-height: 1.4;
  }
</style>
