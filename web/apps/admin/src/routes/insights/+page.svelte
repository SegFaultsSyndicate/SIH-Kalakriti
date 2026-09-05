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
  import { Button, Input, FieldGroup, showToast } from '@kalakriti/ui';
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
      if (import.meta.env.DEV) {
        // Fallback mock data during development when the backend server is offline
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
</script>

<svelte:head>
  <title>{t('nav.insights')} — {t('admin.home.title')}</title>
</svelte:head>

{#if !authorized}
  <h1>{t('nav.insights')}</h1>
  <p role="alert">{t('insights.accessRestricted')}</p>
{:else}
  <div class="insights-header">
    <h1>{t('nav.insights')}</h1>
    <Button variant="secondary" onclick={doRefresh} loading={refreshing}>
      <Icon name="refresh" />
      {t('insights.refresh')}
    </Button>
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
    <BarChart caption={t('insights.declineRate')} rows={dyingCraftRows} formatValue={(v) => `${v}%`} />
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

  <section class="insights-panel">
    <div class="insights-panel__head">
      <h2>{t('insights.incomeUplift')}</h2>
      <Button variant="ghost" onclick={exportUplift}>
        <Icon name="download" />
        {t('insights.exportCsv')}
      </Button>
    </div>
    <p class="insights-panel__note">{t('insights.upliftNote')}</p>
    <div class="insights-panel__grid">
      <BarChart caption={t('insights.medianBefore')} rows={upliftBeforeRows} formatValue={formatPaise} />
      <BarChart caption={t('insights.medianAfter')} rows={upliftAfterRows} formatValue={formatPaise} />
    </div>
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
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
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
    grid-template-columns: repeat(auto-fit, minmax(20rem, 1fr));
    gap: var(--k-space-4);
  }
</style>
