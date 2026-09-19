<!--
  apps/artisan/src/lib/RegisterStep.svelte

    <RegisterStep index={0} heading={t('register.name.heading')} backHref="/welcome">
      {#snippet children()}...{/snippet}
      {#snippet actions()}...{/snippet}
    </RegisterStep>

  Chrome shared by all five /register/* screens: the Stepper, the spoken
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
    t('register.cluster.heading'),
  ]);
</script>

<div class="register-step">
  <Stepper
    label={t('register.step.label', { current: index + 1, total: stepLabels.length })}
    steps={stepLabels}
    current={index}
  />

  <h1>{heading}</h1>
  <SpeakButton text={speakText ?? heading} label={t('action.speak')} />

  <div class="register-step__content">
    {@render children()}
  </div>

  <p class="register-step__saved">{t('register.saved')}</p>

  <div class="register-step__actions">
    <Button size="xl" class="register-step__back" onclick={() => goto(backHref)}>
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

  .register-step__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .register-step__saved {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  .register-step__actions {
    display: flex;
    align-items: stretch;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
  }

  .register-step__actions .register-step__next {
    flex: 1;
  }

  .register-step__actions :global(button) {
    inline-size: 100%;
  }
</style>
