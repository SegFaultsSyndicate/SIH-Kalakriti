<!--
  apps/artisan/src/lib/CoverageStrip.svelte

  F12 home strip: one line on whether this month's sales cover the EMI,
  linking to /finance. Renders nothing for an artisan with no EMI on file
  (or when offline) -- it is a reminder, not an empty state.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { getRepaymentCoverage, type RepaymentCoverage } from '@kalakriti/api';

  const t = $derived(locale.t);

  let coverage = $state<RepaymentCoverage | null>(null);

  $effect(() => {
    getRepaymentCoverage()
      .then((r) => (coverage = r.coverage ?? null))
      .catch(() => (coverage = null));
  });

  const TEXT: Record<string, MessageKey> = {
    COVERED: 'finance.coverage.covered',
    ALMOST: 'finance.coverage.almost',
    NOT_YET: 'finance.coverage.notYet',
  };
  const key = $derived(coverage?.status ? TEXT[coverage.status] : undefined);
</script>

{#if key}
  <a class="strip strip--{coverage?.status?.toLowerCase()}" href="/finance">
    <span>{t(key)}</span>
    {#if (coverage?.days_to_emi ?? -1) >= 0}
      <span class="strip__days">{t('finance.coverage.days', { days: coverage?.days_to_emi ?? 0 })}</span>
    {/if}
  </a>
{/if}

<style>
  .strip {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    padding: var(--k-space-3) var(--k-space-4);
    border-inline-start: 4px solid var(--k-accent-warning-bg);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-weight: 600;
    text-decoration: none;
  }

  .strip--covered {
    border-inline-start-color: var(--k-accent-success-bg);
  }

  .strip__days {
    font-size: var(--k-text-sm);
    font-weight: 400;
    color: var(--k-text-secondary);
  }
</style>
