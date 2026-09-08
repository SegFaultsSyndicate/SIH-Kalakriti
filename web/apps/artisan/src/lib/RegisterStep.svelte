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
  import { Stepper, SpeakButton, LanguageSelector } from '@kalakriti/ui';

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
    t('register.cluster.heading'),
  ]);
</script>

<div class="register-step">
  <Stepper
    label={t('register.step.label', { current: index + 1, total: stepLabels.length })}
    steps={stepLabels}
    current={index}
  />

  <div class="register-step__top-bar">
    <button type="button" class="register-step__back" onclick={() => goto(backHref)}>
      ← {t('action.back')}
    </button>

    <div class="register-step__lang-wrap" title="Change language / भाषा बदलें">
      <LanguageSelector />
    </div>
  </div>

  <h1>{heading}</h1>
  <SpeakButton text={speakText ?? heading} label={t('action.speak')} />

  <div class="register-step__content">
    {@render children()}
  </div>

  <p class="register-step__saved">{t('register.saved')}</p>

  <div class="register-step__actions">
    {@render actions()}
  </div>
</div>

<style>
  .register-step {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-5);
  }

  .register-step__top-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
  }

  .register-step__lang-wrap {
    display: flex;
    align-items: center;
  }

  .register-step__back {
    align-self: center;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    padding-block: var(--k-space-2);
    border: none;
    border-radius: var(--k-radius-md);
    background-color: var(--k-premium-button-bg, #7A3E26);
    color: var(--k-premium-button-text, #F4F0EA);
    font-size: var(--k-text-sm);
    cursor: pointer;
    font-weight: 600;
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
    margin-block-start: var(--k-space-3);
  }

  .register-step__actions :global(button) {
    inline-size: 100%;
  }
</style>
