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
    /** Optional second series for grouped (before/after) bars. Must be same length as rows. */
    secondaryRows?: BarRow[];
    /** Labels for the two series when secondaryRows is provided. */
    legend?: [string, string];
    formatValue?: (v: number) => string;
    /** Table header for the row-label column; defaults to "District". */
    labelHeader?: string;
  }

  let { caption, rows, secondaryRows, legend, formatValue = (v) => String(v), labelHeader }: Props = $props();

  const t = $derived(locale.t);
  const grouped = $derived(!!secondaryRows && secondaryRows.length > 0);
  const allValues = $derived(() => {
    const primary = rows.filter((r) => !r.suppressed && Number.isFinite(r.value)).map((r) => r.value);
    const secondary = (secondaryRows ?? []).filter((r) => !r.suppressed && Number.isFinite(r.value)).map((r) => r.value);
    return [...primary, ...secondary];
  });
  const maxValue = $derived(Math.max(1, ...allValues()));

  let selected = $state('chart');
</script>

<section class="barchart" aria-label={caption}>
  <h3 class="barchart__caption">{caption}</h3>
  {#if grouped && legend && selected === 'chart'}
    <div class="barchart__legend">
      <span class="barchart__legend-item"><span class="barchart__legend-swatch barchart__legend-swatch--primary"></span>{legend[0]}</span>
      <span class="barchart__legend-item"><span class="barchart__legend-swatch barchart__legend-swatch--secondary"></span>{legend[1]}</span>
    </div>
  {/if}
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
            {#each rows as row, i (row.label + '-' + i)}
              <li class="barchart__row" class:barchart__row--grouped={grouped}>
                <span class="barchart__label">{row.label}</span>
                {#if row.suppressed}
                  <span class="barchart__suppressed-bar" role="img" aria-label={t('insights.suppressed')}></span>
                  <span class="barchart__value barchart__value--suppressed">{t('insights.suppressed')}</span>
                {:else}
                  <div class="barchart__bar-group">
                    <span class="barchart__track">
                      <span class="barchart__fill" style:inline-size="{(row.value / maxValue) * 100}%"></span>
                    </span>
                    {#if grouped && secondaryRows?.[i]}
                      <span class="barchart__track">
                        <span class="barchart__fill barchart__fill--secondary" style:inline-size="{((secondaryRows[i].suppressed ? 0 : secondaryRows[i].value) / maxValue) * 100}%"></span>
                      </span>
                    {/if}
                  </div>
                  <span class="barchart__value">
                    {formatValue(row.value)}
                    {#if grouped && secondaryRows?.[i] && !secondaryRows[i].suppressed}
                      <span class="barchart__value--secondary"> / {formatValue(secondaryRows[i].value)}</span>
                    {/if}
                  </span>
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
                <th scope="col">{labelHeader ?? t('insights.district')}</th>
                <th scope="col">{grouped && legend ? legend[0] : caption}</th>
                {#if grouped && legend}
                  <th scope="col">{legend[1]}</th>
                {/if}
              </tr>
            </thead>
            <tbody>
              {#each rows as row, i (row.label + '-' + i)}
                <tr>
                  <th scope="row">{row.label}</th>
                  <td>{row.suppressed ? t('insights.suppressed') : formatValue(row.value)}</td>
                  {#if grouped && secondaryRows}
                    <td>{secondaryRows[i]?.suppressed ? t('insights.suppressed') : formatValue(secondaryRows[i]?.value ?? 0)}</td>
                  {/if}
                </tr>
              {:else}
                <tr><td colspan={grouped ? 3 : 2}>{t('insights.noData')}</td></tr>
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

  .barchart__legend {
    display: flex;
    gap: var(--k-space-4);
    margin-block-end: var(--k-space-3);
    font-size: var(--k-text-xs);
  }

  .barchart__legend-item {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-secondary);
  }

  .barchart__legend-swatch {
    display: inline-block;
    inline-size: 0.75rem;
    block-size: 0.75rem;
    border-radius: 2px;
  }

  .barchart__legend-swatch--primary {
    background: var(--k-accent-primary-bg);
  }

  .barchart__legend-swatch--secondary {
    background: var(--k-neem-500);
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

  /* On a narrow phone the fixed label/value columns leave no room for the
     track; stack label+value, then the bar on its own full-width row. */
  @media (max-width: 34rem) {
    .barchart__row {
      grid-template-columns: 1fr auto;
      grid-template-areas:
        'label value'
        'track track';
    }

    .barchart__label {
      grid-area: label;
      white-space: normal;
      overflow: visible;
      text-overflow: clip;
    }

    .barchart__track,
    .barchart__suppressed-bar {
      grid-area: track;
    }

    .barchart__value {
      grid-area: value;
    }

    .barchart__value--suppressed {
      text-align: end;
    }
  }

  .barchart__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--k-text-secondary);
  }

  .barchart__bar-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .barchart__track {
    block-size: 0.85rem;
    background: var(--k-surface-sunken);
    border-radius: var(--k-radius-xs, 0.2rem);
    overflow: hidden;
  }

  .barchart__row--grouped .barchart__track {
    block-size: 0.55rem;
  }

  .barchart__fill {
    display: block;
    block-size: 100%;
    background: var(--k-accent-primary-bg);
    min-inline-size: 2px;
  }

  .barchart__fill--secondary {
    background: var(--k-neem-500);
  }

  .barchart__value--secondary {
    color: var(--k-neem-500);
    font-weight: 600;
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
    /* th is bold by browser default; keep the whole table at regular
       weight so the data reads flat. */
    font-weight: var(--k-weight-regular);
  }

  .sr-only {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }

  @media print {
    .barchart {
      break-inside: avoid;
      border-color: var(--k-stone-400);
    }
  }
</style>
