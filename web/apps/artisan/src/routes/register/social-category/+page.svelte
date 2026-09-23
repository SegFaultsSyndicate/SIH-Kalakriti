<!-- apps/artisan/src/routes/register/social-category/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Button, SpeakButton } from '@kalakriti/ui';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft } from '$lib/registration';

  const t = $derived(locale.t);

  interface CategoryOption {
    readonly code: string;
    readonly labelKey: MessageKey;
  }

  const CATEGORIES: readonly CategoryOption[] = [
    { code: 'GENERAL', labelKey: 'registration.socialCategory.general' },
    { code: 'OBC', labelKey: 'registration.socialCategory.obc' },
    { code: 'SC', labelKey: 'registration.socialCategory.sc' },
    { code: 'ST', labelKey: 'registration.socialCategory.st' },
    { code: 'EWS', labelKey: 'registration.socialCategory.ews' },
    { code: 'PREFER_NOT_TO_SAY', labelKey: 'registration.socialCategory.preferNotToSay' },
  ];

  let selected = $state<string>('');

  $effect(() => {
    void getDraft().then((draft) => {
      selected = draft.socialCategory ?? '';
    });
  });

  function selectCategory(code: string): void {
    selected = code;
    void patchDraft({ socialCategory: code });
  }

  async function next(): Promise<void> {
    await goto('/register/income');
  }

  const spokenPrompt = $derived(
    `${t('registration.socialCategory.label')}. ${CATEGORIES.map((c) => t(c.labelKey)).join(', ')}`,
  );
</script>

<svelte:head>
  <title>{t('registration.socialCategory.label')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep
  index={4}
  heading={t('registration.socialCategory.label')}
  speakText={spokenPrompt}
  backHref="/register/pehchan"
>
  {#snippet children()}
    <p class="social-hint">{t('schemes.subtitle')}</p>

    <div
      class="social-grid"
      role="radiogroup"
      aria-label={t('registration.socialCategory.label')}
    >
      {#each CATEGORIES as cat (cat.code)}
        {@const isSelected = selected === cat.code}
        {@const label = t(cat.labelKey)}
        <div class="social-tile-wrap">
          <button
            type="button"
            class="social-tile"
            class:social-tile--selected={isSelected}
            role="radio"
            aria-checked={isSelected}
            onclick={() => selectCategory(cat.code)}
          >
            <span class="social-tile__radio-mark" aria-hidden="true">
              {#if isSelected}
                <span class="social-tile__radio-dot"></span>
              {/if}
            </span>
            <span class="social-tile__label">{label}</span>
          </button>
          <div class="social-tile__voice">
            <SpeakButton text={label} label={label} iconOnly />
          </div>
        </div>
      {/each}
    </div>
  {/snippet}

  {#snippet actions()}
    <Button
      size="xl"
      onclick={next}
      tooltip={tooltip(selected !== '' ? 'tooltip.next' : 'tooltip.skip')}
    >
      {selected !== '' ? `${t('action.next')} →` : t('action.skip')}
    </Button>
  {/snippet}
</RegisterStep>

<style>
  .social-hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    line-height: 1.4;
    margin-block-end: var(--k-space-2);
  }

  .social-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--k-space-3);
  }

  @media (max-width: 420px) {
    .social-grid {
      grid-template-columns: 1fr;
    }
  }

  .social-tile-wrap {
    position: relative;
    display: flex;
    align-items: stretch;
  }

  .social-tile {
    flex: 1;
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    min-block-size: calc(var(--k-touch-min) * 1.25);
    padding: var(--k-space-3);
    padding-inline-end: calc(var(--k-touch-min) + var(--k-space-1));
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    font-weight: 500;
    text-align: start;
    cursor: pointer;
    box-shadow: none;
    transition: background-color 0.15s ease, border-color 0.15s ease;
  }

  .social-tile:hover {
    background-color: var(--k-surface-sunken);
  }

  .social-tile--selected {
    border-color: var(--k-accent-primary-bg);
    border-width: var(--k-rule);
    background-color: var(--k-surface-sunken);
    font-weight: 600;
  }

  .social-tile__radio-mark {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    inline-size: 1.25rem;
    block-size: 1.25rem;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-base);
  }

  .social-tile--selected .social-tile__radio-mark {
    border-color: var(--k-accent-primary-bg);
  }

  .social-tile__radio-dot {
    inline-size: 0.625rem;
    block-size: 0.625rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-accent-primary-bg);
  }

  .social-tile__label {
    flex: 1;
    line-height: 1.3;
  }

  .social-tile__voice {
    position: absolute;
    inset-inline-end: var(--k-space-2);
    inset-block-start: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
  }
</style>
