<!--
  apps/artisan/src/routes/welcome/+page.svelte

  One illustration, one short spoken introduction (offered, never
  auto-played -- see the do-not), one large primary action. No wall of text:
  welcome.body is two sentences, and that is the entire copy on this screen.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Illustration } from '@kalakriti/illustrations';
  import { SpeakButton, Button } from '@kalakriti/ui';

  const t = $derived(locale.t);

  async function start(): Promise<void> {
    await goto('/login');
  }
</script>

<svelte:head>
  <title>{t('welcome.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="welcome">
  <Illustration name="onboard-paid-directly" size="14rem" class="welcome__illustration" />

  <h1 class="welcome__heading">{t('welcome.heading')}</h1>
  <p class="welcome__body">{t('welcome.body')}</p>

  <SpeakButton text={`${t('welcome.heading')}. ${t('welcome.body')}`} label={t('welcome.listen')} />

  <Button size="xl" class="welcome__cta" onclick={start}>{t('welcome.start')}</Button>
</div>

<style>
  .welcome {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-5);
    padding-block: var(--k-space-6);
    text-align: center;
  }

  .welcome :global(.welcome__illustration) {
    max-inline-size: 100%;
  }

  .welcome__heading {
    font-size: var(--k-text-2xl);
    color: var(--k-text-primary);
  }

  .welcome__body {
    max-inline-size: var(--k-measure-narrow);
    color: var(--k-text-secondary);
    font-size: var(--k-text-md);
    line-height: var(--k-leading-normal);
  }

  .welcome :global(.welcome__cta) {
    inline-size: 100%;
    margin-block-start: var(--k-space-4);
  }
</style>
