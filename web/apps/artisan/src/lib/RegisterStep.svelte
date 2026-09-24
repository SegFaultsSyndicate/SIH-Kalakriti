<!--
  apps/artisan/src/lib/RegisterStep.svelte

    <RegisterStep index={0} heading={t('register.name.heading')} backHref="/welcome">
      {#snippet children()}...{/snippet}
      {#snippet actions()}...{/snippet}
    </RegisterStep>

  Chrome shared by every /register/* screen: the Stepper, the spoken
  prompt (offered, not auto-played), and a working back button. `actions` is
  a snippet rather than a fixed Next button because whether the primary
  action is enabled, what it is labelled ("Next" vs "Skip" vs "Finish"), and
  what it does on press differ per step -- see each step's own page.
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
    t('register.name.heading'),
    t('register.craft.heading'),
    t('register.district.heading'),
    t('register.pehchan.heading'),
    t('registration.socialCategory.label'),
    t('income.baseline.step'),
    t('register.cluster.heading'),
  ]);

  // Route segment per step, same order as stepLabels.
  const STEP_ROUTES = ['name', 'craft', 'district', 'pehchan', 'social-category', 'income', 'cluster'];
  const stepHref = (i: number): string => `/register/${STEP_ROUTES[i]}`;
</script>

<div class="register-step">
  <Stepper
    label={t('register.step.label', { current: index + 1, total: stepLabels.length })}
    steps={stepLabels}
    current={index}
    {stepHref}
  />

  <h1>{heading}</h1>
  <SpeakButton class="register-step__speak" text={speakText ?? heading} label={t('action.speak')} />

  <div class="register-step__content">
    {@render children()}
  </div>

  <p class="register-step__saved">{t('register.saved')}</p>

  <div class="register-step__actions">
    <Button size="xl" variant="secondary" class="register-step__back" onclick={() => goto(backHref)}>
      ← {t('action.back')}
    </Button>
    <div class="register-step__next">
      {@render actions()}
    </div>
  </div>
</div>

<style>
  .register-step {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-5);
  }

  .register-step__actions :global(.register-step__back) {
    flex: 1;
  }

  .register-step h1 {
    font-size: var(--k-text-xl);
  }

  .register-step :global(.register-step__speak) {
    align-self: flex-start;
  }

  .register-step__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .register-step__saved {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  /* Pinned to the viewport bottom (no bottom nav during onboarding) so
     Back/Next are always in thumb reach without scrolling. */
  .register-step__actions {
    position: sticky;
    inset-block-end: env(safe-area-inset-bottom, 0px);
    z-index: 1;
    display: flex;
    align-items: stretch;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
    padding-block: var(--k-space-3);
  }

  .register-step__actions .register-step__next {
    flex: 1;
  }

  .register-step__actions :global(button) {
    inline-size: 100%;
  }
</style>
