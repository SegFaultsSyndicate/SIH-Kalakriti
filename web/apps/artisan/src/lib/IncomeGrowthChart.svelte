<!--
  apps/artisan/src/lib/IncomeGrowthChart.svelte

  F13: the artisan's own income against what they told us they earned
  before Kalakriti. Every number comes from GET /income/summary (core-svc,
  pkg/impact's rules -- the same ones the ministry dashboard aggregates
  with). No figure is ever invented: when the server says there is not
  enough data yet (no baseline, under 90 days on the platform, no sales),
  the card says exactly that instead of showing a growth number.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Button, Money, Sheet, Skeleton, showToast } from '@kalakriti/ui';
  import { enqueue } from '@kalakriti/offline';
  import { getIncomeSummary, getActingFor, type IncomeSummary } from '@kalakriti/api';
  import BracketPicker from './BracketPicker.svelte';
  import LogSaleSheet from './LogSaleSheet.svelte';

  interface Props {
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const t = $derived(locale.t);

  let summary = $state<IncomeSummary | null>(null);
  let failed = $state(false);
  let saleOpen = $state(false);
  let baselineOpen = $state(false);
  let bracket = $state('');

  async function load(): Promise<void> {
    failed = false;
    try {
      summary = (await getIncomeSummary()).summary ?? null;
    } catch {
      failed = true;
    }
  }

  $effect(() => {
    void load();
  });

  const REASONS: Record<string, MessageKey> = {
    NO_BASELINE: 'income.card.noBaseline',
    TOO_NEW: 'income.card.tooNew',
    NO_SALES: 'income.card.noSales',
  };

  const months = $derived(summary?.months ?? []);
  const barMax = $derived(
    Math.max(1, summary?.baseline_monthly_paise ?? 0, ...months.map((m) => (m.platform_paise ?? 0) + (m.offline_paise ?? 0))),
  );
  const monthFmt = $derived(new Intl.DateTimeFormat(locale.meta.tag, { month: 'short' }));
  const pct = (v: number): string => `${v > 0 ? '+' : ''}${Math.round(v)}%`;

  async function saveBaseline(): Promise<void> {
    if (!bracket) return;
    await enqueue({ kind: 'income.baseline', onBehalfOf: getActingFor(), payload: { monthly_bracket: bracket } });
    baselineOpen = false;
    showToast({ message: t('income.sale.saved'), variant: 'success' });
  }
</script>

<section class="income" aria-labelledby="income-title">
  <header class="income__head">
    <h2 id="income-title" class="income__title">{t('income.card.title')}</h2>
    <Button size="md" variant="secondary" onclick={() => (saleOpen = true)}>{t('income.sale.log')}</Button>
  </header>

  {#if failed}
    <p class="income__note">{t('state.error.body')}</p>
    <Button size="md" variant="ghost" onclick={load}>{t('state.error.retry')}</Button>
  {:else if summary === null}
    <Skeleton shape="card" />
  {:else}
    {#if summary.insufficient_data}
      <p class="income__note">{t(REASONS[summary.insufficient_data] ?? 'income.card.noSales')}</p>
      {#if summary.insufficient_data === 'NO_BASELINE'}
        <Button size="md" onclick={() => (baselineOpen = true)}>{t('income.card.addBaseline')}</Button>
      {/if}
    {:else if summary.uplift_pct !== undefined}
      <div class="income__hero">
        <span class="income__pct">{pct(summary.uplift_pct)}</span>
        <span>{t('income.card.vsBefore')}</span>
      </div>
    {/if}

    <dl class="income__pair">
      <div>
        <dt>{t('income.card.now')}</dt>
        <dd><Money paise={summary.current_monthly_paise ?? 0} /></dd>
      </div>
      {#if (summary.baseline_monthly_paise ?? 0) > 0}
        <div>
          <dt>{t('income.card.before')}</dt>
          <dd><Money paise={summary.baseline_monthly_paise ?? 0} /></dd>
        </div>
      {/if}
    </dl>

    {#if !compact}
      <figure class="income__bars" aria-label={t('income.card.last3')}>
        {#each months as m (m.month)}
          {@const online = m.platform_paise ?? 0}
          {@const offline = m.offline_paise ?? 0}
          <div class="income__month">
            <div class="income__track">
              {#if (summary.baseline_monthly_paise ?? 0) > 0}
                <span class="income__base" style:inset-block-end="{((summary.baseline_monthly_paise ?? 0) / barMax) * 100}%"></span>
              {/if}
              <span class="income__seg income__seg--off" style:block-size="{(offline / barMax) * 100}%"></span>
              <span class="income__seg" style:block-size="{(online / barMax) * 100}%"></span>
            </div>
            <span class="income__label">{monthFmt.format(new Date(`${m.month}-01T00:00:00`))}</span>
            <span class="k-visually-hidden"><Money paise={online + offline} /></span>
          </div>
        {/each}
      </figure>
      <p class="income__legend">
        <span class="income__key"></span>{t('income.card.online')}
        <span class="income__key income__key--off"></span>{t('income.card.offline')}
      </p>

      <dl class="income__split">
        <div><dt>{t('income.card.online')}</dt><dd><Money paise={summary.platform_paise_90d ?? 0} /></dd></div>
        <div><dt>{t('income.card.offline')}</dt><dd><Money paise={summary.offline_paise_90d ?? 0} /></dd></div>
        <div><dt>{t('income.card.fair')}</dt><dd><Money paise={summary.fair_paise_90d ?? 0} /></dd></div>
      </dl>
      {#if summary.digital_share_pct !== undefined}
        <p class="income__note">{t('income.card.digitalShare', { pct: Math.round(summary.digital_share_pct) })}</p>
      {/if}
    {/if}

    {#if (summary.platform_pending_paise ?? 0) > 0}
      <p class="income__note">
        {t('income.card.pending')}: <Money paise={summary.platform_pending_paise ?? 0} />
      </p>
    {/if}
    {#if (summary.baseline_monthly_paise ?? 0) > 0}
      <p class="income__foot">{t('income.card.selfReported')}</p>
    {/if}
  {/if}
</section>

<LogSaleSheet bind:open={saleOpen} />

<Sheet bind:open={baselineOpen} title={t('income.baseline.question')}>
  <div class="income__sheet">
    <p class="income__note">{t('income.baseline.why')}</p>
    <BracketPicker bind:value={bracket} />
    <Button size="xl" disabled={!bracket} onclick={saveBaseline}>{t('action.save')}</Button>
  </div>
</Sheet>

<style>
  .income {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-lg, 1rem);
    background: var(--k-surface-raised);
  }

  .income__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
    flex-wrap: wrap;
  }

  .income__title {
    margin: 0;
    font-size: var(--k-text-lg);
  }

  .income__hero {
    display: flex;
    flex-direction: column;
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: var(--k-accent-success-bg);
    color: var(--k-text-on-accent);
  }

  .income__pct {
    font-size: 2.25rem;
    font-weight: 800;
    line-height: 1;
  }

  .income__pair,
  .income__split {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(7rem, 1fr));
    gap: var(--k-space-2);
    margin: 0;
  }

  .income__pair dt,
  .income__split dt {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .income__pair dd,
  .income__split dd {
    margin: 0;
    font-size: var(--k-text-md);
    font-weight: 700;
  }

  .income__bars {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-3);
    margin: 0;
  }

  .income__month {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-1);
  }

  .income__track {
    position: relative;
    display: flex;
    flex-direction: column-reverse;
    inline-size: 2.5rem;
    block-size: 7rem;
    border-radius: 4px 4px 0 0;
    background: var(--k-surface-sunken);
    overflow: hidden;
  }

  .income__seg {
    inline-size: 100%;
    background: var(--k-accent-success-bg);
  }

  .income__seg--off {
    background: var(--k-indigo-400);
  }

  /* Before-Kalakriti monthly figure, as a line across each month. */
  .income__base {
    position: absolute;
    inset-inline: 0;
    block-size: 2px;
    background: var(--k-text-primary);
  }

  .income__label,
  .income__legend,
  .income__foot {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .income__legend {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin: 0;
  }

  .income__key {
    display: inline-block;
    inline-size: 0.75rem;
    block-size: 0.75rem;
    border-radius: 2px;
    background: var(--k-accent-success-bg);
  }

  .income__key--off {
    background: var(--k-indigo-400);
  }

  .income__note {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .income__foot {
    margin: 0;
    font-style: italic;
  }

  .income__sheet {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }
</style>
