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
  import { page } from '$app/state';
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

  // Route segment per step, same order as stepLabels.
  const STEP_ROUTES = ['capture', 'studio', 'video', 'story', 'processing', 'review', 'pricing', 'terms'];
  const draftId = $derived(page.url.searchParams.get('d') ?? '');
  const stepHref = (i: number): string => `/listing/new/${STEP_ROUTES[i]}?d=${draftId}`;
</script>

<div class="listing-step">
  <Stepper
    label={t('listing.step.label', { current: index + 1, total: stepLabels.length })}
    steps={stepLabels}
    current={index}
    {stepHref}
  />

  <h1>{heading}</h1>
  <SpeakButton class="listing-step__speak" text={speakText ?? heading} label={t('action.speak')} />

  <div class="listing-step__content">
    {@render children()}
  </div>

  <p class="listing-step__saved">{t('listing.saved')}</p>

  <div class="listing-step__actions">
    <Button size="xl" variant="secondary" class="listing-step__back" onclick={() => goto(backHref)}>
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

  .listing-step :global(.listing-step__speak) {
    align-self: flex-start;
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

  /* Pinned above the bottom nav so Back/Next are always in thumb reach
     without scrolling (the shell uses overflow-x: clip, so sticky works). */
  .listing-step__actions {
    position: sticky;
    inset-block-end: calc(4.5rem + env(safe-area-inset-bottom, 0px));
    z-index: 1;
    display: flex;
    align-items: stretch;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
    padding-block: var(--k-space-3);
    background: var(--k-premium-canvas, var(--k-surface-base));
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .listing-step__actions .listing-step__next {
    flex: 1;
  }

  .listing-step__actions :global(button) {
    inline-size: 100%;
  }
</style>
