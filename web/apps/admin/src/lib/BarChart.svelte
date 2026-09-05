<!--
  apps/admin/src/lib/BarChart.svelte

    <BarChart
      caption="GMV by district"
      unit="₹"
      rows={[{ label: 'Kutch, GJ', value: 420000 }, { label: 'Barmer, RJ', value: 0, suppressed: true }]}
    />

  Hand-rolled SVG (no chart library installed) with a Tabs toggle to a real
  <table> holding the same rows -- the acceptance criterion is a keyboard-
  accessible table equivalent for every chart, not a screen-reader label
  bolted onto a canvas. A suppressed row never gets a bar and never reads as
  0: it renders as a hatched placeholder in the chart and the literal text
  "Suppressed for privacy" in the table, in both value cells. Bars carry
  their value as text next to them, not colour alone.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Tabs } from '@kalakriti/ui';

  export interface BarRow {
    label: string;
    value: number;
    suppressed?: boolean;
  }

  interface Props {
    caption: string;
    rows: BarRow[];
    formatValue?: (v: number) => string;
  }

  let { caption, rows, formatValue = (v) => String(v) }: Props = $props();

  const t = $derived(locale.t);
  const maxValue = $derived(Math.max(1, ...rows.filter((r) => !r.suppressed).map((r) => r.value)));

  let selected = $state('chart');
</script>

<section class="barchart" aria-label={caption}>
  <h3 class="barchart__caption">{caption}</h3>
  <Tabs
    tabs={[
      { id: 'chart', label: t('insights.chart') },
      { id: 'table', label: t('insights.table') },
    ]}
    bind:selected
  >
    {#snippet children(tabId)}
      {#if tabId === 'chart'}
        {#if rows.length === 0}
          <p class="barchart__empty">{t('insights.noData')}</p>
        {:else}
          <ul class="barchart__bars">
            {#each rows as row (row.label)}
              <li class="barchart__row">
                <span class="barchart__label">{row.label}</span>
                {#if row.suppressed}
                  <span class="barchart__suppressed-bar" role="img" aria-label={t('insights.suppressed')}></span>
                  <span class="barchart__value barchart__value--suppressed">{t('insights.suppressed')}</span>
                {:else}
                  <span class="barchart__track">
                    <span class="barchart__fill" style:inline-size="{(row.value / maxValue) * 100}%"></span>
                  </span>
                  <span class="barchart__value">{formatValue(row.value)}</span>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      {:else}
        <div class="barchart__table-scroll">
          <table class="barchart__table">
            <caption class="sr-only">{caption}</caption>
            <thead>
              <tr>
                <th scope="col">{t('insights.district')}</th>
                <th scope="col">{caption}</th>
              </tr>
            </thead>
            <tbody>
              {#each rows as row (row.label)}
                <tr>
                  <th scope="row">{row.label}</th>
                  <td>{row.suppressed ? t('insights.suppressed') : formatValue(row.value)}</td>
                </tr>
              {:else}
                <tr><td colspan="2">{t('insights.noData')}</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/snippet}
  </Tabs>
</section>

<style>
  .barchart {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-4);
  }

  .barchart__caption {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    margin: 0 0 var(--k-space-3);
  }

  .barchart__empty {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .barchart__bars {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .barchart__row {
    display: grid;
    grid-template-columns: 9rem 1fr 7rem;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
  }

  .barchart__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--k-text-secondary);
  }

  .barchart__track {
    block-size: 0.85rem;
    background: var(--k-surface-sunken);
    border-radius: var(--k-radius-xs, 0.2rem);
    overflow: hidden;
  }

  .barchart__fill {
    display: block;
    block-size: 100%;
    background: var(--k-accent-primary-bg);
    min-inline-size: 2px;
  }

  .barchart__suppressed-bar {
    display: block;
    block-size: 0.85rem;
    border-radius: var(--k-radius-xs, 0.2rem);
    background: repeating-linear-gradient(
      45deg,
      var(--k-surface-sunken),
      var(--k-surface-sunken) 4px,
      var(--k-border-hairline) 4px,
      var(--k-border-hairline) 8px
    );
  }

  .barchart__value {
    font-variant-numeric: var(--k-numeric-tabular);
    text-align: end;
    color: var(--k-text-primary);
  }

  .barchart__value--suppressed {
    color: var(--k-text-secondary);
    font-style: italic;
    text-align: start;
  }

  .barchart__table-scroll {
    overflow-x: auto;
  }

  .barchart__table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .barchart__table th,
  .barchart__table td {
    text-align: start;
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .sr-only {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
</style>
