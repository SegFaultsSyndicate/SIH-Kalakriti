<!--
  apps/artisan/src/lib/IncomeGrowthChart.svelte

  Artisan-Facing Income Growth & Economic Uplift Visualization:
  Directly addresses the SIH Problem Statement's core impact goal —
  "increasing average annual income of traditional craftspeople" by converting
  temporary exhibition sales spikes into continuous year-round digital prosperity.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Tabs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const t = $derived(locale.t);
  let activeTab = $state('chart');

  interface QuarterData {
    periodKey: MessageKey;
    highlightKey: MessageKey;
    baseline: number;     // Earnings without Kalakriti (only physical melas)
    kalakriti: number;    // Earnings with Kalakriti (year-round digital + melas)
  }

  const QUARTERS: QuarterData[] = [
    { periodKey: 'growth.q1', highlightKey: 'growth.q1.highlight', baseline: 12000, kalakriti: 38500 },
    { periodKey: 'growth.q2', highlightKey: 'growth.q2.highlight', baseline: 6000, kalakriti: 31000 },
    { periodKey: 'growth.q3', highlightKey: 'growth.q3.highlight', baseline: 18000, kalakriti: 54000 },
    { periodKey: 'growth.q4', highlightKey: 'growth.q4.highlight', baseline: 32000, kalakriti: 61500 },
  ];

  const totalBaseline = $derived(QUARTERS.reduce((acc, q) => acc + q.baseline, 0));
  const totalKalakriti = $derived(QUARTERS.reduce((acc, q) => acc + q.kalakriti, 0));
  const totalUpliftPaise = $derived(totalKalakriti - totalBaseline);
  const upliftPercentage = $derived(Math.round(((totalKalakriti - totalBaseline) / totalBaseline) * 100));

  const maxVal = $derived(Math.max(...QUARTERS.map((q) => q.kalakriti)));

  function formatInr(val: number): string {
    return `₹${val.toLocaleString('en-IN')}`;
  }
</script>

<section class="income-growth-card" aria-label={t('growth.title')}>
  <header class="growth-header">
    <span class="growth-badge">
      <Icon name="verified-artisan" size="0.85rem" />
      {t('growth.badge')}
    </span>
    <h2 class="growth-title">{t('growth.title')}</h2>
    <p class="growth-subhead">{t('growth.subtitle')}</p>
  </header>

  <!-- Hero: the headline uplift, with a tiny per-quarter bar trail. -->
  <div class="uplift-hero">
    <div class="uplift-hero__text">
      <span class="uplift-hero__label">{t('growth.annualUplift')}</span>
      <span class="uplift-hero__num">+{upliftPercentage}%</span>
      <span class="uplift-hero__sub">{t('growth.extraIncome', { amount: formatInr(totalUpliftPaise) })}</span>
    </div>
    <div class="uplift-hero__spark" aria-hidden="true">
      {#each QUARTERS as q (q.periodKey)}
        <span class="spark-pair">
          <span class="spark spark--base" style:block-size="{(q.baseline / maxVal) * 100}%"></span>
          <span class="spark" style:block-size="{(q.kalakriti / maxVal) * 100}%"></span>
        </span>
      {/each}
    </div>
  </div>

  <!-- Supporting stats (the uplift % already leads the hero above). -->
  <ul class="stat-strip" role="list">
    <li class="stat-item">
      <span class="stat-item__icon"><Icon name="income-statement" size="1.1rem" /></span>
      <span class="stat-item__val">{formatInr(totalUpliftPaise)}</span>
      <span class="stat-item__lbl">{t('growth.statAdditional')}</span>
    </li>
    <li class="stat-item">
      <span class="stat-item__icon"><Icon name="fair-price" size="1.1rem" /></span>
      <span class="stat-item__val">0%</span>
      <span class="stat-item__lbl">{t('growth.statZeroBrokers')}</span>
    </li>
    <li class="stat-item">
      <span class="stat-item__icon"><Icon name="users" size="1.1rem" /></span>
      <span class="stat-item__val">4.2x</span>
      <span class="stat-item__lbl">{t('growth.statReach')}</span>
    </li>
  </ul>

  {#if !compact}
    <!-- Tabs: Chart vs Table -->
    <div class="growth-tabs-wrap">
      <div class="chart-legend">
        <span class="legend-item">
          <span class="legend-swatch legend-swatch--baseline"></span>
          {t('growth.baselineLabel')}
        </span>
        <span class="legend-item">
          <span class="legend-swatch legend-swatch--kalakriti"></span>
          {t('growth.withKalakritiLabel')}
        </span>
      </div>

      <Tabs
        tabs={[
          { id: 'chart', label: t('growth.chartTab') },
          { id: 'table', label: t('growth.tableTab') },
        ]}
        bind:selected={activeTab}
      >
        {#snippet children(tabId)}
          {#if tabId === 'chart'}
            <!-- Visual Grouped Bar Chart -->
            <div class="chart-canvas" role="figure" aria-label={t('growth.chartAriaLabel')}>
              <div class="bars-container">
                {#each QUARTERS as q (q.periodKey)}
                  {@const baselinePct = Math.round((q.baseline / maxVal) * 100)}
                  {@const kalakritiPct = Math.round((q.kalakriti / maxVal) * 100)}
                  {@const quarterUplift = Math.round(((q.kalakriti - q.baseline) / q.baseline) * 100)}

                  <div class="quarter-col">
                    <div class="quarter-bars">
                      <!-- Baseline Bar -->
                      <div class="bar-slot">
                        <span class="bar-value">{formatInr(q.baseline)}</span>
                        <div class="bar-track">
                          <div class="bar-fill bar-fill--baseline" style:block-size="{baselinePct}%"></div>
                        </div>
                      </div>

                      <!-- Kalakriti Bar -->
                      <div class="bar-slot">
                        <span class="bar-value bar-value--highlight">{formatInr(q.kalakriti)}</span>
                        <div class="bar-track">
                          <div class="bar-fill bar-fill--kalakriti" style:block-size="{kalakritiPct}%"></div>
                        </div>
                      </div>
                    </div>

                    <div class="quarter-meta">
                      <span class="quarter-label">{t(q.periodKey).split(' ')[0]}</span>
                      <span class="quarter-uplift-tag">+{quarterUplift}%</span>
                    </div>
                  </div>
                {/each}
              </div>

              <!-- Key Insight Callout -->
              <div class="insight-banner">
                <span class="insight-icon"><Icon name="info" size="1.1rem" /></span>
                <p>
                  <strong>{t('growth.insight.heading')}</strong> {t('growth.insight.body')}
                </p>
              </div>
            </div>
          {:else}
            <!-- Accessible Data Table -->
            <div class="table-scroll">
              <table class="growth-table">
                <thead>
                  <tr>
                    <th scope="col">{t('growth.table.timeline')}</th>
                    <th scope="col">{t('growth.table.physicalMelasOnly')}</th>
                    <th scope="col">{t('growth.table.withKalakritiDigital')}</th>
                    <th scope="col">{t('growth.table.netGain')}</th>
                    <th scope="col">{t('growth.table.driver')}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each QUARTERS as q}
                    <tr>
                      <th scope="row"><strong>{t(q.periodKey)}</strong></th>
                      <td>{formatInr(q.baseline)}</td>
                      <td class="cell-green">{formatInr(q.kalakriti)}</td>
                      <td class="cell-gain">+{formatInr(q.kalakriti - q.baseline)}</td>
                      <td class="cell-highlight">{t(q.highlightKey)}</td>
                    </tr>
                  {/each}
                  <tr class="table-total">
                    <th scope="row"><strong>{t('growth.table.annualTotal')}</strong></th>
                    <td><strong>{formatInr(totalBaseline)}</strong></td>
                    <td class="cell-green"><strong>{formatInr(totalKalakriti)}</strong></td>
                    <td class="cell-gain"><strong>+{formatInr(totalUpliftPaise)} (+{upliftPercentage}%)</strong></td>
                    <td><strong>{t('growth.table.dbt')}</strong></td>
                  </tr>
                </tbody>
              </table>
            </div>
          {/if}
        {/snippet}
      </Tabs>
    </div>
  {/if}
</section>

<style>
  .income-growth-card {
    background: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: 1rem;
    padding: var(--k-space-4);
    box-shadow: 0 1px 2px rgb(0 0 0 / 0.04), 0 8px 24px rgb(0 0 0 / 0.05);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .growth-header {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--k-space-1);
  }

  .growth-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    background: color-mix(in srgb, var(--k-accent-primary-bg) 10%, transparent);
    color: var(--k-accent-primary-text);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    padding: 0.25rem 0.6rem;
    border-radius: var(--k-radius-pill);
    margin-block-end: var(--k-space-1);
  }

  .growth-title {
    font-size: var(--k-text-lg);
    font-weight: 800;
    line-height: 1.2;
    color: var(--k-text-primary);
    margin: 0;
  }

  .growth-subhead {
    font-size: var(--k-text-sm);
    line-height: var(--k-leading-normal);
    color: var(--k-text-secondary);
    margin: 0;
  }

  .uplift-hero {
    position: relative;
    overflow: hidden;
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: var(--k-space-4);
    background:
      radial-gradient(120% 140% at 100% 0%, rgb(255 255 255 / 0.14), transparent 55%),
      linear-gradient(135deg, var(--k-neem-700), var(--k-accent-success-bg));
    color: var(--k-text-on-accent);
    padding: var(--k-space-4);
    border-radius: 0.875rem;
    box-shadow: 0 6px 18px color-mix(in srgb, var(--k-accent-success-bg) 30%, transparent);
  }

  .uplift-hero__text {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    min-inline-size: 0;
  }

  .uplift-hero__label {
    font-size: var(--k-text-xs);
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    opacity: 0.85;
  }

  .uplift-hero__num {
    font-size: 2.5rem;
    font-weight: 900;
    line-height: 1;
    letter-spacing: -0.02em;
  }

  .uplift-hero__sub {
    font-size: var(--k-text-sm);
    opacity: 0.9;
  }

  .uplift-hero__spark {
    display: flex;
    align-items: flex-end;
    gap: 0.4rem;
    block-size: 3.5rem;
    flex-shrink: 0;
  }

  .spark-pair {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    block-size: 100%;
  }

  .spark {
    inline-size: 0.45rem;
    border-radius: 3px 3px 1px 1px;
    background: rgb(255 255 255 / 0.9);
  }

  .spark--base {
    background: rgb(255 255 255 / 0.35);
  }

  .stat-strip {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--k-space-2);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    padding: var(--k-space-3);
    border-radius: 0.75rem;
    background: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .stat-item__icon {
    display: grid;
    place-items: center;
    inline-size: 2rem;
    block-size: 2rem;
    margin-block-end: var(--k-space-1);
    border-radius: var(--k-radius-pill);
    background: color-mix(in srgb, var(--k-accent-success-bg) 12%, transparent);
    color: var(--k-accent-success-muted);
  }

  .stat-item__val {
    font-size: var(--k-text-md);
    font-weight: 800;
    color: var(--k-text-primary);
    font-variant-numeric: tabular-nums;
  }

  /* Labels wrap rather than ellipsis-truncate: longer locales must stay readable. */
  .stat-item__lbl {
    font-size: var(--k-text-xs);
    line-height: 1.3;
    color: var(--k-text-secondary);
    overflow-wrap: anywhere;
  }

  /* Legend */
  .growth-tabs-wrap {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .chart-legend {
    display: flex;
    gap: var(--k-space-4);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .legend-swatch {
    inline-size: 0.8rem;
    block-size: 0.8rem;
    border-radius: 2px;
  }

  .legend-swatch--baseline {
    background: var(--k-indigo-400);
  }

  .legend-swatch--kalakriti {
    background: var(--k-accent-success-bg);
  }

  /* Visual Grouped Bar Canvas */
  .chart-canvas {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-3) 0;
  }

  .bars-container {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--k-space-4);
    min-block-size: 13rem;
    align-items: end;
    border-block-end: 2px solid var(--k-border-hairline);
    padding-block-end: var(--k-space-2);
  }

  .quarter-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    block-size: 100%;
    justify-content: flex-end;
  }

  .quarter-bars {
    display: flex;
    gap: 6px;
    align-items: flex-end;
    block-size: 10rem;
    inline-size: 100%;
    justify-content: center;
  }

  .bar-slot {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    block-size: 100%;
    justify-content: flex-end;
    inline-size: 2.2rem;
  }

  .bar-value {
    font-size: 0.65rem;
    color: var(--k-text-secondary);
    font-weight: 600;
  }

  .bar-value--highlight {
    color: var(--k-accent-success-muted);
    font-weight: 800;
  }

  .bar-track {
    inline-size: 100%;
    block-size: 8rem;
    display: flex;
    align-items: flex-end;
    background: var(--k-surface-sunken);
    border-radius: 4px 4px 0 0;
    overflow: hidden;
  }

  .bar-fill {
    inline-size: 100%;
    border-radius: 4px 4px 0 0;
    transition: block-size 0.4s ease;
  }

  .bar-fill--baseline {
    background: var(--k-indigo-400);
  }

  .bar-fill--kalakriti {
    background: linear-gradient(180deg, var(--k-neem-500), var(--k-accent-success-bg));
  }

  .quarter-meta {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .quarter-label {
    font-size: var(--k-text-xs);
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .quarter-uplift-tag {
    font-size: 0.65rem;
    background: rgba(34, 197, 94, 0.15);
    color: var(--k-accent-success-muted);
    font-weight: 800;
    padding: 1px 6px;
    border-radius: 999px;
  }

  /* Insight Banner */
  .insight-banner {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-base);
    border: 1px solid var(--k-neem-300);
    color: var(--k-accent-success);
    font-size: var(--k-text-xs);
    line-height: var(--k-leading-normal);
  }

  .insight-icon {
    font-size: 1rem;
    flex-shrink: 0;
  }

  /* Data Table */
  .table-scroll {
    overflow-x: auto;
  }

  .growth-table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-xs);
    text-align: start;
  }

  .growth-table th,
  .growth-table td {
    padding: var(--k-space-2) var(--k-space-3);
    border-block-end: 1px solid var(--k-border-hairline);
  }

  .cell-green {
    color: var(--k-accent-success-muted);
    font-weight: 700;
  }

  .cell-gain {
    color: var(--k-accent-success-muted);
    font-weight: 800;
  }

  .cell-highlight {
    color: var(--k-text-secondary);
    font-style: italic;
  }

  .table-total {
    background: var(--k-surface-raised);
    border-block-start: 2px solid var(--k-border-hairline);
  }

  /* Narrow card: tiles become rows (icon · label · value) so a long
     number like ₹1,17,000 never breaks mid-digit. */
  .income-growth-card {
    container-type: inline-size;
  }

  @container (max-width: 34rem) {
    .stat-strip {
      grid-template-columns: minmax(0, 1fr);
      gap: 0;
      border: var(--k-hairline) solid var(--k-border-hairline);
      border-radius: 0.75rem;
      background: var(--k-surface-base);
    }

    .stat-item {
      display: grid;
      grid-template-columns: auto minmax(0, 1fr) auto;
      align-items: center;
      column-gap: var(--k-space-3);
      border: none;
      border-radius: 0;
      background: none;
    }

    .stat-item + .stat-item {
      border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    }

    .stat-item__icon {
      margin: 0;
    }

    .stat-item__val {
      grid-column: 3;
      grid-row: 1;
      white-space: nowrap;
    }

    .stat-item__lbl {
      grid-column: 2;
      grid-row: 1;
      font-size: var(--k-text-sm);
    }
  }

  @media (max-width: 640px) {
    .bar-slot {
      inline-size: 1.4rem;
    }
  }
</style>
