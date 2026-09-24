<!--
  apps/admin/src/routes/insights/+page.svelte

  The ministry dashboard. MINISTRY role only -- gated by hiding the page's
  content behind a plain access-restricted message, not a redirect to a
  /login the role claim can't grant: there is no way to become MINISTRY
  short of the account already carrying that role in its token.

  Suppression: insight-svc omits small-bucket rows (<5 artisans) entirely
  from GET /insights/earnings-by-district and /insights/income-comparison --
  there is no `suppressed: bool` field anywhere in the API, only silent row
  omission (see ml_wiring.md #10). GET /insights/artisans-by-category applies
  NO suppression at all and is used here purely as the district roster: any
  district it lists but an earnings/income response omits is rendered as
  "suppressed for privacy", never as a zero or a blank cell.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Button, Input, FieldGroup, Skeleton, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    getArtisansByCategory,
    getEarningsByDistrict,
    getIncomeComparison,
    getListingsByCraftMonth,
    getDyingCrafts,
    refreshInsights,
    ApiError,
    messageKeyFor,
  } from '@kalakriti/api';
  import BarChart, { type BarRow } from '$lib/BarChart.svelte';
  import { toCsv, downloadCsv } from '$lib/csv';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY');

  let stateCode = $state('');
  let district = $state('');
  let loading = $state(false);
  let refreshing = $state(false);
  let loadError = $state('');

  interface DistrictKey {
    stateCode: string;
    district: string;
  }

  type EarningsRow = NonNullable<Awaited<ReturnType<typeof getEarningsByDistrict>>['data']>[number];
  type IncomeRow = NonNullable<Awaited<ReturnType<typeof getIncomeComparison>>['data']>[number];
  type CraftMonthRow = NonNullable<Awaited<ReturnType<typeof getListingsByCraftMonth>>['data']>[number];
  type DyingCraftRow = NonNullable<Awaited<ReturnType<typeof getDyingCrafts>>['data']>[number];

  let roster = $state<{ key: DistrictKey; label: string; artisanCount: number }[]>([]);
  let earningsRows = $state<EarningsRow[]>([]);
  let incomeRows = $state<IncomeRow[]>([]);
  let craftRows = $state<CraftMonthRow[]>([]);
  let dyingRows = $state<DyingCraftRow[]>([]);

  const MOCK_DYING_CRAFTS: DyingCraftRow[] = [
    { craft_id: 'c1', craft_name: 'Rogan Art of Kutch', decline_rate: 0.78, peak_artisans: 45, current_artisans: 10 },
    { craft_id: 'c2', craft_name: 'Toda Tribal Embroidery', decline_rate: 0.65, peak_artisans: 82, current_artisans: 29 },
    { craft_id: 'c3', craft_name: 'Surat Real Zari', decline_rate: 0.52, peak_artisans: 120, current_artisans: 58 },
    { craft_id: 'c4', craft_name: 'Tangaliya Shawl Weaving', decline_rate: 0.44, peak_artisans: 68, current_artisans: 38 },
    { craft_id: 'c5', craft_name: 'Usta Camel Hide Art', decline_rate: 0.36, peak_artisans: 50, current_artisans: 32 },
    { craft_id: 'c6', craft_name: 'Mata ni Pachedi Ritual Cloth', decline_rate: 0.28, peak_artisans: 74, current_artisans: 53 },
  ];

  const MOCK_DISTRICT_DATA = [
    { stateCode: 'IN-GJ', district: 'Kutch', artisanCount: 142, gmv: 84500000, avgEarnings: 5950000, medianBefore: 1800000, medianAfter: 4200000 },
    { stateCode: 'IN-GJ', district: 'Surat', artisanCount: 128, gmv: 76000000, avgEarnings: 5930000, medianBefore: 1900000, medianAfter: 4400000 },
    { stateCode: 'IN-GJ', district: 'Patan', artisanCount: 88, gmv: 62000000, avgEarnings: 7040000, medianBefore: 2200000, medianAfter: 5100000 },
    { stateCode: 'IN-GJ', district: 'Ahmedabad', artisanCount: 64, gmv: 38000000, avgEarnings: 5930000, medianBefore: 1700000, medianAfter: 3900000 },
    { stateCode: 'IN-RJ', district: 'Jaipur', artisanCount: 156, gmv: 91000000, avgEarnings: 5830000, medianBefore: 2100000, medianAfter: 4800000 },
    { stateCode: 'IN-RJ', district: 'Jodhpur', artisanCount: 94, gmv: 52000000, avgEarnings: 5530000, medianBefore: 1650000, medianAfter: 3950000 },
    { stateCode: 'IN-BR', district: 'Madhubani', artisanCount: 112, gmv: 56000000, avgEarnings: 5000000, medianBefore: 1500000, medianAfter: 3900000 },
    { stateCode: 'IN-WB', district: 'Bankura', artisanCount: 76, gmv: 37000000, avgEarnings: 4860000, medianBefore: 1400000, medianAfter: 3600000 },
    { stateCode: 'IN-KA', district: 'Mysuru', artisanCount: 82, gmv: 41000000, avgEarnings: 5000000, medianBefore: 1900000, medianAfter: 4100000 },
    { stateCode: 'IN-MH', district: 'Thane', artisanCount: 68, gmv: 33000000, avgEarnings: 4850000, medianBefore: 1600000, medianAfter: 3800000 },
  ];

  const MOCK_CRAFTS: CraftMonthRow[] = [
    { craft_name: 'Patan Patola Silk', listing_count: 84 },
    { craft_name: 'Blue Pottery of Jaipur', listing_count: 76 },
    { craft_name: 'Madhubani Painting', listing_count: 68 },
    { craft_name: 'Kutch Ajrakh Block Print', listing_count: 55 },
    { craft_name: 'Bankura Terracotta Horses', listing_count: 42 },
    { craft_name: 'Warli Tribal Painting', listing_count: 39 },
    { craft_name: 'Bidriware Silver Inlay', listing_count: 31 },
    { craft_name: 'Sandalwood Carving', listing_count: 26 },
  ];

  function districtKey(k: { state_code?: string; district?: string }): string {
    return `${k.state_code ?? ''}::${k.district ?? ''}`;
  }

  function districtLabel(k: { state_code?: string; district?: string }): string {
    return [k.district, k.state_code].filter(Boolean).join(', ') || t('insights.unknownDistrict');
  }

  async function load(): Promise<void> {
    loading = true;
    loadError = '';
    try {
      const query = {
        state_code: stateCode || undefined,
        district: district || undefined,
      };
      const [category, earnings, income, craftMonth, dying] = await Promise.all([
        getArtisansByCategory(query),
        getEarningsByDistrict(query),
        getIncomeComparison(query),
        getListingsByCraftMonth({}),
        getDyingCrafts(),
      ]);

      const byDistrict = new Map<string, { key: DistrictKey; label: string; artisanCount: number }>();
      for (const row of category.data ?? []) {
        const key = districtKey(row);
        const existing = byDistrict.get(key);
        const count = (existing?.artisanCount ?? 0) + (row.artisan_count ?? 0);
        byDistrict.set(key, {
          key: { stateCode: row.state_code ?? '', district: row.district ?? '' },
          label: districtLabel(row),
          artisanCount: count,
        });
      }
      roster = [...byDistrict.values()].sort((a, b) => b.artisanCount - a.artisanCount);

      earningsRows = earnings.data ?? [];
      incomeRows = income.data ?? [];
      craftRows = craftMonth.data ?? [];
      dyingRows = [...(dying.data ?? [])].sort((a, b) => Math.abs(b.decline_rate ?? 0) - Math.abs(a.decline_rate ?? 0));
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] load insights:', cause);
        const filtered = MOCK_DISTRICT_DATA.filter((d) => {
          const matchState = !stateCode || d.stateCode.toLowerCase().includes(stateCode.trim().toLowerCase());
          const matchDistrict = !district || d.district.toLowerCase().includes(district.trim().toLowerCase());
          return matchState && matchDistrict;
        });

        roster = filtered.map((d) => ({
          key: { stateCode: d.stateCode, district: d.district },
          label: `${d.district}, ${d.stateCode}`,
          artisanCount: d.artisanCount,
        }));

        earningsRows = filtered.map((d) => ({
          state_code: d.stateCode,
          district: d.district,
          total_gmv: { amount_paise: d.gmv },
          avg_earnings: { amount_paise: d.avgEarnings },
        }));

        incomeRows = filtered.map((d) => ({
          state_code: d.stateCode,
          district: d.district,
          median_before: { amount_paise: d.medianBefore },
          median_after: { amount_paise: d.medianAfter },
          artisan_count: d.artisanCount,
        }));

        craftRows = MOCK_CRAFTS;
        dyingRows = MOCK_DYING_CRAFTS;
      } else {
        loadError = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (authorized) void load();
  });

  function formatPaise(paise: number | undefined): string {
    return `₹${Math.round((paise ?? 0) / 100).toLocaleString('en-IN')}`;
  }

  const gmvRows = $derived<BarRow[]>(
    roster.map((r) => {
      const match = earningsRows.find((e) => districtKey(e) === districtKey({ state_code: r.key.stateCode, district: r.key.district }));
      return match
        ? { label: r.label, value: match.total_gmv?.amount_paise ?? 0 }
        : { label: r.label, value: 0, suppressed: true };
    }),
  );

  const avgEarningsRows = $derived<BarRow[]>(
    roster.map((r) => {
      const match = earningsRows.find((e) => districtKey(e) === districtKey({ state_code: r.key.stateCode, district: r.key.district }));
      return match
        ? { label: r.label, value: match.avg_earnings?.amount_paise ?? 0 }
        : { label: r.label, value: 0, suppressed: true };
    }),
  );

  const upliftBeforeRows = $derived<BarRow[]>(
    roster.map((r) => {
      const match = incomeRows.find((i) => districtKey(i) === districtKey({ state_code: r.key.stateCode, district: r.key.district }));
      return match
        ? { label: r.label, value: match.median_before?.amount_paise ?? 0 }
        : { label: r.label, value: 0, suppressed: true };
    }),
  );

  const upliftAfterRows = $derived<BarRow[]>(
    roster.map((r) => {
      const match = incomeRows.find((i) => districtKey(i) === districtKey({ state_code: r.key.stateCode, district: r.key.district }));
      return match
        ? { label: r.label, value: match.median_after?.amount_paise ?? 0 }
        : { label: r.label, value: 0, suppressed: true };
    }),
  );

  const artisanRosterRows = $derived<BarRow[]>(roster.map((r) => ({ label: r.label, value: r.artisanCount })));

  const craftByListingRows = $derived<BarRow[]>(
    (() => {
      const byCraft = new Map<string, number>();
      for (const row of craftRows) {
        byCraft.set(row.craft_name ?? '', (byCraft.get(row.craft_name ?? '') ?? 0) + (row.listing_count ?? 0));
      }
      return [...byCraft.entries()]
        .map(([label, value]) => ({ label, value }))
        .sort((a, b) => b.value - a.value)
        .slice(0, 20);
    })(),
  );

  const dyingCraftRows = $derived<BarRow[]>(
    dyingRows.map((r) => ({ label: r.craft_name ?? '', value: Math.round(Math.abs(r.decline_rate ?? 0) * 100) })),
  );

  function exportSection(name: string, headers: string[], rows: BarRow[]): void {
    const csv = toCsv(headers, rows.map((r) => [r.label, r.suppressed ? 'suppressed for privacy' : r.value]));
    downloadCsv(`${name}.csv`, csv);
  }

  function exportUplift(): void {
    const csv = toCsv(
      ['district', 'median_before', 'median_after'],
      roster.map((r, i) => [
        r.label,
        upliftBeforeRows[i]?.suppressed ? 'suppressed for privacy' : upliftBeforeRows[i]?.value,
        upliftAfterRows[i]?.suppressed ? 'suppressed for privacy' : upliftAfterRows[i]?.value,
      ]),
    );
    downloadCsv('income-uplift-by-district.csv', csv);
  }

  async function doRefresh(): Promise<void> {
    refreshing = true;
    try {
      await refreshInsights();
      showToast({ variant: 'success', message: t('insights.refreshed') });
      await load();
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    } finally {
      refreshing = false;
    }
  }

  // ── KPI computations (derived from already-loaded data, no extra calls) ──

  const kpiTotalArtisans = $derived(roster.reduce((sum, r) => sum + r.artisanCount, 0));

  const kpiTotalGmv = $derived(
    earningsRows.reduce((sum, r) => sum + (r.total_gmv?.amount_paise ?? 0), 0),
  );

  const kpiAvgUplift = $derived.by(() => {
    let count = 0;
    let totalPct = 0;
    for (const row of incomeRows) {
      const before = row.median_before?.amount_paise ?? 0;
      const after = row.median_after?.amount_paise ?? 0;
      if (before > 0) {
        totalPct += ((after - before) / before) * 100;
        count++;
      }
    }
    return count > 0 ? Math.round(totalPct / count) : 0;
  });

  const kpiCraftsAtRisk = $derived(dyingRows.filter((r) => Math.abs(r.decline_rate ?? 0) >= 0.3).length);

  // ── Dying-craft severity helpers ──

  function dyingSeverity(rate: number): 'critical' | 'warning' | 'watch' {
    const abs = Math.abs(rate);
    if (abs >= 0.6) return 'critical';
    if (abs >= 0.3) return 'warning';
    return 'watch';
  }

  function severityLabel(sev: 'critical' | 'warning' | 'watch'): string {
    return t(`insights.dyingCraft.${sev}`);
  }

  function printDashboard(): void {
    window.print();
  }
</script>

<svelte:head>
  <title>{t('nav.insights')} — {t('admin.home.title')}</title>
</svelte:head>

{#if !authorized}
  <h1>{t('nav.insights')}</h1>
  <p role="alert">{t('insights.accessRestricted')}</p>
{:else}
  <div class="insights-header">
    <div>
      <h1>{t('nav.insights')}</h1>
      <p class="insights-subtitle">{t('insights.dashboardSubtitle')}</p>
    </div>
    <div class="insights-header__actions">
      <Button variant="secondary" onclick={printDashboard}>
        <Icon name="print" />
        {t('insights.printDashboard')}
      </Button>
      <Button variant="secondary" onclick={doRefresh} loading={refreshing}>
        <Icon name="refresh" />
        {t('insights.refresh')}
      </Button>
    </div>
  </div>

  <form class="insights-filters" onsubmit={(e) => { e.preventDefault(); void load(); }}>
    <FieldGroup label={t('insights.stateCode')}>
      {#snippet children({ id })}
        <Input {id} bind:value={stateCode} placeholder="IN-GJ" />
      {/snippet}
    </FieldGroup>
    <FieldGroup label={t('insights.district')}>
      {#snippet children({ id })}
        <Input {id} bind:value={district} />
      {/snippet}
    </FieldGroup>
    <Button type="submit" loading={loading}>{t('insights.applyFilters')}</Button>
  </form>

  {#if loadError}
    <p role="alert" class="insights-error">{loadError}</p>
  {/if}

  {#if loading}
    <div class="insights-skeleton" aria-hidden="true">
      <div class="insights-kpi-strip">
        {#each Array(4) as _, i (i)}
          <div class="insights-kpi">
            <Skeleton shape="text" width="4rem" height="1.5rem" />
            <Skeleton shape="text" width="6rem" height="0.75rem" />
          </div>
        {/each}
      </div>
      <ul class="dying-craft-list">
        {#each Array(3) as _, i (i)}
          <li class="dying-craft-item">
            <div class="dying-craft-item__header">
              <Skeleton shape="text" width="8rem" height="1rem" />
              <Skeleton width="4rem" height="1.25rem" radius="var(--k-radius-pill)" />
            </div>
            <div class="dying-craft-item__stats">
              <Skeleton shape="text" width="6rem" height="0.85rem" />
              <Skeleton shape="text" width="8rem" height="0.85rem" />
            </div>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <!-- KPI Hero Strip -->
  {#if !loading}
    <div class="insights-kpi-strip">
      <div class="insights-kpi">
        <span class="insights-kpi__value">{kpiTotalArtisans.toLocaleString('en-IN')}</span>
        <span class="insights-kpi__label">{t('insights.kpi.totalArtisans')}</span>
      </div>
      <div class="insights-kpi">
        <span class="insights-kpi__value">{formatPaise(kpiTotalGmv)}</span>
        <span class="insights-kpi__label">{t('insights.kpi.totalGmvValue')}</span>
      </div>
      <div class="insights-kpi">
        <span class="insights-kpi__value insights-kpi__value--uplift">+{kpiAvgUplift}%</span>
        <span class="insights-kpi__label">{t('insights.kpi.avgUplift')}</span>
      </div>
      <div class="insights-kpi">
        <span class="insights-kpi__value insights-kpi__value--risk">{kpiCraftsAtRisk}</span>
        <span class="insights-kpi__label">{t('insights.kpi.craftsAtRisk')}</span>
      </div>
    </div>
  {/if}

  <!-- Dying-craft watch -->
  <section class="insights-panel insights-panel--dying" aria-labelledby="dying-craft-heading">
    <div class="insights-panel__head">
      <h2 id="dying-craft-heading">
        <Icon name="dying-craft" />
        {t('insights.dyingCraftWatch')}
      </h2>
      <Button variant="ghost" onclick={() => exportSection('dying-craft-watch', ['craft', 'decline_rate_pct'], dyingCraftRows)}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <p class="insights-panel__note">{t('insights.dyingCraftNote')}</p>

    <ul class="dying-craft-list">
      {#each dyingRows as craft (craft.craft_id)}
        {@const severity = dyingSeverity(craft.decline_rate ?? 0)}
        <li class="dying-craft-item">
          <div class="dying-craft-item__header">
            <span class="dying-craft-item__name">{craft.craft_name}</span>
            <span class="dying-craft-severity dying-craft-severity--{severity}">
              {severityLabel(severity)}
            </span>
          </div>
          <div class="dying-craft-item__stats">
            <span class="dying-craft-item__rate">{Math.round(Math.abs(craft.decline_rate ?? 0) * 100)}% decline</span>
            <span class="dying-craft-item__count">
              {t('insights.dyingCraft.peakArtisans')}: {craft.peak_artisans ?? '—'}
              →
              {t('insights.dyingCraft.currentArtisans')}: {craft.current_artisans ?? '—'}
            </span>
          </div>
          <div class="dying-craft-item__bar">
            <span class="dying-craft-item__bar-fill dying-craft-item__bar-fill--{severity}" style:inline-size="{Math.round(Math.abs(craft.decline_rate ?? 0) * 100)}%"></span>
          </div>
        </li>
      {/each}
    </ul>
  </section>

  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.artisansByDistrict')}</h2>
      <Button variant="ghost" onclick={() => exportSection('artisans-by-district', ['district', 'artisan_count'], artisanRosterRows)}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <BarChart caption={t('insights.artisanCount')} rows={artisanRosterRows} />
  </section>

  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.gmvByDistrict')}</h2>
      <Button variant="ghost" onclick={() => exportSection('gmv-by-district', ['district', 'gmv'], gmvRows)}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <p class="insights-panel__note">{t('insights.suppressionNote')}</p>
    <BarChart caption={t('insights.totalGmv')} rows={gmvRows} formatValue={formatPaise} />
  </section>

  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.medianEarnings')}</h2>
      <Button variant="ghost" onclick={() => exportSection('median-earnings-by-district', ['district', 'median_earnings'], avgEarningsRows)}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <BarChart caption={t('insights.averageEarnings')} rows={avgEarningsRows} formatValue={formatPaise} />
  </section>

  <!-- Income uplift: grouped before/after bar chart -->
  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.incomeUplift')}</h2>
      <Button variant="ghost" onclick={exportUplift}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <p class="insights-panel__note">{t('insights.upliftNote')}</p>
    <BarChart
      caption={t('insights.upliftComparison')}
      rows={upliftBeforeRows}
      secondaryRows={upliftAfterRows}
      legend={[t('insights.legend.before'), t('insights.legend.after')]}
      formatValue={formatPaise}
    />
  </section>

  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.listingsByCraft')}</h2>
      <Button variant="ghost" onclick={() => exportSection('listings-by-craft', ['craft', 'listing_count'], craftByListingRows)}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <BarChart caption={t('insights.listingCount')} rows={craftByListingRows} />
  </section>
{/if}

<style>
  .insights-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--k-space-3);
    flex-wrap: wrap;
  }

  .insights-header__actions {
    display: flex;
    gap: var(--k-space-2);
  }

  .insights-subtitle {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin: var(--k-space-1) 0 0;
  }

  .insights-filters {
    display: flex;
    align-items: flex-end;
    gap: var(--k-space-4);
    margin-block: var(--k-space-4) var(--k-space-6);
    flex-wrap: wrap;
  }

  .insights-filters :global(.k-field-group) {
    margin-block-end: 0;
  }

  .insights-filters :global(.k-button) {
    min-block-size: var(--k-touch-min);
  }

  .insights-error {
    color: var(--k-accent-danger);
  }

  /* ── KPI hero strip ── */

  .insights-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    margin-block: var(--k-space-4);
  }

  .insights-kpi-strip {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    margin-block-end: var(--k-space-6);
  }

  .insights-kpi {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    text-align: center;
  }

  .insights-kpi__value {
    font-size: 1.75rem;
    font-weight: 800;
    font-variant-numeric: var(--k-numeric-tabular);
    color: var(--k-text-primary);
    line-height: 1.1;
  }

  .insights-kpi__value--uplift {
    color: var(--k-accent-success-muted);
  }

  .insights-kpi__value--risk {
    color: var(--k-accent-danger);
  }

  .insights-kpi__label {
    font-size: var(--k-text-xs, 0.75rem);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-text-secondary);
  }

  /* ── Dying-craft list ── */

  .dying-craft-list {
    list-style: none;
    margin: var(--k-space-3) 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .dying-craft-item {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .dying-craft-item__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
  }

  .dying-craft-item__name {
    font-weight: 600;
    font-size: var(--k-text-sm);
  }

  .dying-craft-severity {
    font-size: var(--k-text-xs, 0.75rem);
    font-weight: 700;
    padding: 2px var(--k-space-2);
    border-radius: var(--k-radius-pill);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .dying-craft-severity--critical {
    background: var(--k-terracotta-300);
    color: var(--k-accent-danger);
  }

  .dying-craft-severity--warning {
    background: var(--k-accent-warning-bg);
    color: var(--k-haldi-600);
  }

  .dying-craft-severity--watch {
    background: var(--k-neem-300);
    color: var(--k-accent-success-muted);
  }

  .dying-craft-item__stats {
    display: flex;
    gap: var(--k-space-4);
    font-size: var(--k-text-xs, 0.75rem);
    color: var(--k-text-secondary);
  }

  .dying-craft-item__rate {
    font-weight: 600;
  }

  .dying-craft-item__bar {
    block-size: 0.35rem;
    border-radius: 2px;
    background: var(--k-surface-sunken);
    overflow: hidden;
  }

  .dying-craft-item__bar-fill {
    display: block;
    block-size: 100%;
    border-radius: 2px;
    transition: inline-size 0.3s ease;
  }

  .dying-craft-item__bar-fill--critical {
    background: var(--k-madder-700);
  }

  .dying-craft-item__bar-fill--warning {
    background: var(--k-haldi-500);
  }

  .dying-craft-item__bar-fill--watch {
    background: var(--k-neem-500);
  }

  /* ── Panels ── */

  .insights-panel {
    margin-block-end: var(--k-space-6);
  }

  .insights-panel--dying {
    padding: var(--k-space-4);
    border: var(--k-rule-heavy) solid var(--k-accent-danger);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-sunken);
  }

  .insights-panel__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    margin-block-end: var(--k-space-2);
  }

  .insights-panel__head h2 {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-lg);
    margin: 0;
  }

  .insights-panel__note {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block: 0 var(--k-space-2);
  }

  .insights-panel__grid {
    display: grid;
    /* min() caps the track floor at 100% so narrow phones get one full-
       width panel instead of a 20rem overflow. */
    grid-template-columns: repeat(auto-fit, minmax(min(20rem, 100%), 1fr));
    gap: var(--k-space-4);
  }

  /* ── Print ── */

  @media print {
    .insights-header__actions {
      display: none;
    }

    .insights-filters {
      display: none;
    }

    .insights-kpi-strip {
      grid-template-columns: repeat(4, 1fr);
    }

    .insights-kpi {
      border-color: var(--k-stone-400);
    }

    .insights-panel--dying {
      border-color: var(--k-stone-400);
    }

    .insights-panel {
      break-inside: avoid;
    }
  }
</style>
