<!--
  apps/artisan/src/lib/ListingStep.svelte

    <ListingStep index={0} heading={t('listing.capture.heading')} backHref="/">
      {#snippet children()}...{/snippet}
      {#snippet actions()}...{/snippet}
    </ListingStep>

  Chrome shared by all /listing/new/* screens -- same shape as RegisterStep,
  duplicated rather than shared because the two wizards' step counts and
  labels differ enough that a shared component would need a labels prop
  passed from both call sites anyway, which is the same line count as this.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Button, SpeakButton, Stepper } from '@kalakriti/ui';

  interface Props {
    /** 0-indexed. */
    index: number;
    heading: string;
    /** Defaults to `heading` alone. */
    speakText?: string;
    backHref: string;
    children: Snippet;
    actions: Snippet;
  }

  let { index, heading, speakText, backHref, children, actions }: Props = $props();

  const t = $derived(locale.t);
  const stepLabels = $derived([
    t('listing.step.capture'),
    t('listing.step.studio'),
    t('listing.step.video'),
    t('listing.step.story'),
    t('listing.step.processing'),
    t('listing.step.review'),
    t('listing.step.pricing'),
    t('listing.step.terms'),
  ]);
</script>

<div class="listing-step">
  <Stepper
    label={t('listing.step.label', { current: index + 1, total: stepLabels.length })}
    steps={stepLabels}
    current={index}
  />

  <h1>{heading}</h1>
  <SpeakButton text={speakText ?? heading} label={t('action.speak')} />

  <div class="listing-step__content">
    {@render children()}
  </div>

  <p class="listing-step__saved">{t('listing.saved')}</p>

  <div class="listing-step__actions">
    <Button size="xl" class="listing-step__back" onclick={() => goto(backHref)}>
      ← {t('action.back')}
    </Button>
    <div class="listing-step__next">
      {@render actions()}
    </div>
  </div>
</div>

<style>
  .listing-step {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-5);
  }

  .listing-step__actions :global(.listing-step__back) {
    flex: 1;
  }

  .listing-step h1 {
    font-size: var(--k-text-xl);
  }

  .listing-step__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .listing-step__saved {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  .listing-step__actions {
    display: flex;
    align-items: stretch;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
  }

  .listing-step__actions .listing-step__next {
    flex: 1;
  }

  .listing-step__actions :global(button) {
    inline-size: 100%;
  }
</style>
