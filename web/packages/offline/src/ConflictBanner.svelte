<!--
  packages/offline/src/ConflictBanner.svelte

  Shown when conflict.ts's detectConflict finds the draft was edited against a
  version the server has since moved past. Logic lives in conflict.ts; this
  file only renders the two choices and reports which one the artisan picked.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';

  interface Props {
    onResolve: (resolution: 'keep-mine' | 'use-theirs') => void;
  }

  const { onResolve }: Props = $props();

  const t = $derived(locale.t);
</script>

<div class="banner" role="alert">
  <p class="banner__title">{t('offline.conflict.title')}</p>
  <p class="banner__body">{t('offline.conflict.body')}</p>
  <div class="banner__actions">
    <button type="button" class="banner__action" onclick={() => onResolve('keep-mine')}>
      {t('offline.conflict.keepMine')}
    </button>
    <button
      type="button"
      class="banner__action banner__action--primary"
      onclick={() => onResolve('use-theirs')}
    >
      {t('offline.conflict.useTheirs')}
    </button>
  </div>
</div>

<style>
  .banner {
    padding: var(--k-space-4);
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-danger);
    background-color: var(--k-surface-sunken);
  }

  .banner__title {
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .banner__body {
    margin-block-start: var(--k-space-1);
    color: var(--k-text-secondary);
  }

  .banner__actions {
    display: flex;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-4);
  }

  .banner__action {
    min-block-size: var(--k-touch-min);
    padding: var(--k-space-2) var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background-color: transparent;
    color: var(--k-text-primary);
    font-size: var(--k-text-md);
    cursor: pointer;
  }

  .banner__action--primary {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
    font-weight: var(--k-weight-medium);
  }
</style>
