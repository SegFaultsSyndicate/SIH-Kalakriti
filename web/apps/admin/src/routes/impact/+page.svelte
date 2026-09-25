<!--
  apps/admin/src/routes/impact/+page.svelte

  F13 ministry impact dashboard for MINISTRY and CLUSTER_OFFICER (an
  officer's filters are clamped to their own scope
  server-side, whatever this form sends). Every figure comes from
  insight-svc's /impact/* endpoints, computed with pkg/impact's rules. When
  VITE_USE_MOCKS=1 is set, $lib/stubs.ts appends sample rows to any section
  real rows don't already cover (deduped) so the page never looks empty; if
  the endpoints are unreachable the stubs fill the page and a
  "[mock fallback]" console.warn is logged. Any group of fewer than 5
  artisans arrives flagged `suppressed` and renders as "<5", never as a
  zero. Baselines are self-reported and the page says so.
-->
<script lang="ts">
  import { locale, formatMoney, formatDate, formatNumber, type MessageKey } from '@kalakriti/i18n';
  import { Button, FieldGroup, Input, Select, showToast } from '@kalakriti/ui';
  import {
    session,
    getImpactSummary,
    getImpactByGroup,
    getSalesMix,
    getFinanceCoverage,
    getLiteracyFunnel,
    exportImpactCsv,
    refreshInsights,
    ApiError,
    messageKeyFor,
    type ImpactFilterQuery,
    type ImpactGroupBy,
    type ImpactGroupRow,
    type SalesMixMonth,
    type FinanceCoverageRow,
    type LiteracyFunnelRow,
  } from '@kalakriti/api';
  import BarChart, { type BarRow } from '$lib/BarChart.svelte';
  import {
    IMPACT_SUMMARY,
    IMPACT_GROUPS,
    IMPACT_GROUPS_BY_CORPORATION,
    IMPACT_GROUPS_BY_CATEGORY,
    SALES_MIX,
    FINANCE_COVERAGE,
    LITERACY_FUNNEL,
    mergeWithStubs,
  } from '$lib/stubs';

  type Summary = Awaited<ReturnType<typeof getImpactSummary>>;

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY' || role === 'CLUSTER_OFFICER');
  const SMALL = '<5';

  let stateCode = $state('');
  let district = $state('');
  let category = $state('');
  let corporation = $state('');
  let fromMonth = $state('');
  let toMonth = $state('');
  let groupBy = $state<ImpactGroupBy>('district');

  let summary = $state<Summary | null>(null);
  let groups = $state<ImpactGroupRow[]>([]);
  let mix = $state<SalesMixMonth[]>([]);
  let finance = $state<FinanceCoverageRow[]>([]);
  let funnel = $state<LiteracyFunnelRow[]>([]);
  let loading = $state(false);
  let refreshing = $state(false);
  let exporting = $state(false);
  let error = $state('');

  const CATEGORY_KEYS: Record<string, MessageKey> = {
    GENERAL: 'registration.socialCategory.general',
    OBC: 'registration.socialCategory.obc',
    SC: 'registration.socialCategory.sc',
    ST: 'registration.socialCategory.st',
    EWS: 'registration.socialCategory.ews',
    PREFER_NOT_TO_SAY: 'registration.socialCategory.preferNotToSay',
  };
  const CORPORATIONS = ['NSFDC', 'NBCFDC', 'NSKFDC', 'NDFDC', 'PM_DAKSH', 'PM_AJAY', 'OTHER'];
  const corpLabel = (c: string): string => (c === 'OTHER' ? t('impact.other') : c.replace('_', '-'));

  function groupLabel(row: ImpactGroupRow): string {
    const g = row.group ?? '';
    if (groupBy === 'social_category') return CATEGORY_KEYS[g] ? t(CATEGORY_KEYS[g]) : g || t('impact.notStated');
    if (groupBy === 'corporation') return corpLabel(g);
    return [g || t('insights.unknownDistrict'), row.state_code].filter(Boolean).join(', ');
  }

  function query(): ImpactFilterQuery {
    return {
      state_code: stateCode.trim() || undefined,
      district: district.trim() || undefined,
      social_category: category || undefined,
      corporation: corporation || undefined,
      from_month: fromMonth || undefined,
      to_month: toMonth || undefined,
    };
  }

  const stubGroups: Record<ImpactGroupBy, ImpactGroupRow[]> = {
    district: IMPACT_GROUPS,
    corporation: IMPACT_GROUPS_BY_CORPORATION,
    social_category: IMPACT_GROUPS_BY_CATEGORY,
  };

  async function load(): Promise<void> {
    loading = true;
    error = '';
    try {
      const q = query();
      const [s, g, m, f, l] = await Promise.all([
        getImpactSummary(q),
        getImpactByGroup({ ...q, group_by: groupBy }),
        getSalesMix(q),
        getFinanceCoverage(q),
        getLiteracyFunnel(q),
      ]);
      const groupsData = g.rows ?? [];
      const mixData = m.months ?? [];
      const financeData = f.rows ?? [];
      const funnelData = l.rows ?? [];
      summary = s ?? (import.meta.env.VITE_USE_MOCKS === '1' ? IMPACT_SUMMARY : null);
      groups = mergeWithStubs(groupsData, stubGroups[groupBy], (r) => `${r.group ?? ''}|${r.state_code ?? ''}`);
      mix = mergeWithStubs(mixData, SALES_MIX, (x) => x.month);
      finance = mergeWithStubs(financeData, FINANCE_COVERAGE, (x) => x.corporation);
      funnel = mergeWithStubs(funnelData, LITERACY_FUNNEL, (x) => `${x.state_code}|${x.district ?? ''}`);
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] impact load:', cause);
        summary = IMPACT_SUMMARY;
        groups = stubGroups[groupBy];
        mix = SALES_MIX;
        finance = FINANCE_COVERAGE;
        funnel = LITERACY_FUNNEL;
      } else {
        error = t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown');
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (authorized) void load();
  });

  async function refresh(): Promise<void> {
    refreshing = true;
    try {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        await load();
      } else {
        await refreshInsights();
        await load();
      }
    } catch (cause) {
      showToast({ message: t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown'), variant: 'error' });
    } finally {
      refreshing = false;
    }
  }

  async function exportCsv(): Promise<void> {
    exporting = true;
    try {
      const blob = await exportImpactCsv({ ...query(), group_by: groupBy });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `kalakriti-impact-${groupBy}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (cause) {
      showToast({ message: t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown'), variant: 'error' });
    } finally {
      exporting = false;
    }
  }

  const money = (p: number): string => formatMoney(p, locale.code, { paise: 'never' });
  const num = (n: number | undefined): string => formatNumber(n ?? 0, locale.code);
  const pct = (v: number | undefined): string => (v === undefined ? '—' : `${v > 0 ? '+' : ''}${Math.round(v)}%`);
  const hidden = $derived(summary?.suppressed ?? false);

  const beforeRows = $derived<BarRow[]>(
    groups.map((r) => ({
      label: groupLabel(r),
      value: r.median_baseline_monthly_paise ?? 0,
      suppressed: r.suppressed || (r.with_baseline_count ?? 0) < 5,
    })),
  );
  const nowRows = $derived<BarRow[]>(
    groups.map((r) => ({
      label: groupLabel(r),
      value: r.median_current_monthly_paise ?? 0,
      suppressed: r.suppressed || (r.with_baseline_count ?? 0) < 5,
    })),
  );
  const monthFmt = $derived(new Intl.DateTimeFormat(locale.meta.tag, { month: 'short', year: '2-digit' }));
  const mixLabel = (m: SalesMixMonth): string => monthFmt.format(new Date(`${m.month}-01T00:00:00`));
  const platformRows = $derived<BarRow[]>(mix.map((m) => ({ label: mixLabel(m), value: m.platform_paise ?? 0, suppressed: m.suppressed })));
  const offlineRows = $derived<BarRow[]>(
    mix.map((m) => ({ label: mixLabel(m), value: (m.fair_paise ?? 0) + (m.other_offline_paise ?? 0), suppressed: m.suppressed })),
  );
</script>

<svelte:head>
  <title>{t('impact.title')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="impact">
  <h1>{t('impact.title')}</h1>

  {#if !authorized}
    <p>{t('admin.home.noAccess')}</p>
  {:else}
    <p class="impact__muted">{t('impact.subtitle')}</p>

    <form
      class="impact__filters"
      onsubmit={(e) => {
        e.preventDefault();
        void load();
      }}
    >
      <FieldGroup label={t('impact.filter.state')} optional>
        {#snippet children({ id })}
          <Input {id} bind:value={stateCode} placeholder="IN-UP" />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('insights.district')} optional>
        {#snippet children({ id })}
          <Input {id} bind:value={district} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('registration.socialCategory.label')} optional>
        {#snippet children({ id })}
          <Select
            {id}
            bind:value={category}
            options={[{ value: '', label: t('impact.all') }, ...Object.entries(CATEGORY_KEYS).map(([value, key]) => ({ value, label: t(key) }))]}
          />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('impact.filter.corporation')} optional>
        {#snippet children({ id })}
          <Select
            {id}
            bind:value={corporation}
            options={[{ value: '', label: t('impact.all') }, ...CORPORATIONS.map((c) => ({ value: c, label: corpLabel(c) }))]}
          />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('impact.filter.from')} optional>
        {#snippet children({ id })}
          <input {id} class="impact__month" type="month" bind:value={fromMonth} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('impact.filter.to')} optional>
        {#snippet children({ id })}
          <input {id} class="impact__month" type="month" bind:value={toMonth} />
        {/snippet}
      </FieldGroup>
      <div class="impact__actions">
        <Button type="submit" loading={loading}>{t('impact.apply')}</Button>
        <Button variant="secondary" loading={refreshing} onclick={refresh}>{t('impact.refresh')}</Button>
        <Button variant="secondary" loading={exporting} onclick={exportCsv}>{t('impact.export')}</Button>
      </div>
    </form>

    {#if error}
      <p class="impact__error" role="alert">{error}</p>
    {/if}

    {#if summary}
      {#if summary.refreshed_at}
        <p class="impact__muted">{t('impact.refreshedAt', { time: formatDate(summary.refreshed_at, locale.code, { dateStyle: 'medium', timeStyle: 'short' }) })}</p>
      {/if}
      {#if hidden}
        <p class="impact__note" role="status">{t('impact.suppressedNote')}</p>
      {/if}
      <dl class="kpis">
        <div class="kpi"><dt>{t('impact.kpi.beneficiaries')}</dt><dd>{hidden ? SMALL : num(summary.beneficiaries)}</dd></div>
        <div class="kpi"><dt>{t('impact.kpi.active')}</dt><dd>{hidden ? SMALL : num(summary.active_sellers_90d)}</dd></div>
        <div class="kpi">
          <dt>{t('impact.kpi.uplift')}</dt>
          <dd>{hidden || summary.median_uplift_pct === undefined ? SMALL : pct(summary.median_uplift_pct)}</dd>
          <p class="impact__muted">{t('impact.kpi.upliftSample', { n: hidden ? SMALL : num(summary.uplift_sample) })}</p>
        </div>
        <div class="kpi"><dt>{t('impact.kpi.digitalShare')}</dt><dd>{hidden || summary.digital_share_pct === undefined ? '—' : `${Math.round(summary.digital_share_pct)}%`}</dd></div>
        <div class="kpi"><dt>{t('impact.kpi.certificates')}</dt><dd>{hidden ? SMALL : num(summary.certificates_issued)}</dd></div>
        <div class="kpi">
          <dt>{t('impact.kpi.financeLinked')}</dt>
          <dd>{hidden ? SMALL : num(summary.finance_linked)}</dd>
          <p class="impact__muted">{t('impact.kpi.financeVerified', { n: hidden ? SMALL : num(summary.finance_verified) })}</p>
        </div>
      </dl>
      <p class="impact__muted">{t('impact.selfReported')}</p>

      <section class="impact__section">
        <div class="impact__section-head">
          <h2>{t('impact.byGroup')}</h2>
          <Select
            bind:value={groupBy}
            aria-label={t('impact.groupBy')}
            onchange={() => void load()}
            options={[
              { value: 'district', label: t('insights.district') },
              { value: 'social_category', label: t('registration.socialCategory.label') },
              { value: 'corporation', label: t('impact.filter.corporation') },
            ]}
          />
        </div>
        <BarChart
          caption={t('impact.byGroup')}
          rows={beforeRows}
          secondaryRows={nowRows}
          legend={[t('impact.before'), t('impact.now')]}
          formatValue={money}
          labelHeader={t('impact.groupBy')}
        />
      </section>

      <section class="impact__section">
        <h2>{t('impact.salesMix')}</h2>
        <BarChart
          caption={t('impact.salesMix')}
          rows={platformRows}
          secondaryRows={offlineRows}
          legend={[t('impact.platform'), t('impact.offline')]}
          formatValue={money}
          labelHeader={t('impact.month')}
        />
      </section>

      <section class="impact__section">
        <h2>{t('impact.finance')}</h2>
        <div class="impact__scroll">
          <table class="impact__table">
            <thead>
              <tr>
                <th scope="col">{t('impact.filter.corporation')}</th>
                <th scope="col">{t('impact.kpi.beneficiaries')}</th>
                <th scope="col">{t('impact.col.verified')}</th>
                <th scope="col">{t('impact.kpi.active')}</th>
                <th scope="col">{t('impact.col.coverage')}</th>
              </tr>
            </thead>
            <tbody>
              {#each finance as r (r.corporation)}
                <tr>
                  <th scope="row">{corpLabel(r.corporation ?? '')}</th>
                  <td>{r.suppressed ? SMALL : num(r.beneficiaries)}</td>
                  <td>{r.suppressed ? SMALL : num(r.verified)}</td>
                  <td>{r.suppressed ? SMALL : num(r.active_sellers_90d)}</td>
                  <td>{r.suppressed || r.median_coverage_ratio === undefined ? '—' : `${Math.round(r.median_coverage_ratio * 100)}%`}</td>
                </tr>
              {:else}
                <tr><td colspan="5">{t('insights.noData')}</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>

      <section class="impact__section">
        <h2>{t('impact.literacy')}</h2>
        <div class="impact__scroll">
          <table class="impact__table">
            <thead>
              <tr>
                <th scope="col">{t('insights.district')}</th>
                <th scope="col">{t('impact.col.artisans')}</th>
                <th scope="col">{t('impact.col.started')}</th>
                <th scope="col">{t('impact.col.halfway')}</th>
                <th scope="col">{t('impact.kpi.certificates')}</th>
              </tr>
            </thead>
            <tbody>
              {#each funnel as r (`${r.state_code}:${r.district}`)}
                <tr>
                  <th scope="row">{[r.district || t('insights.unknownDistrict'), r.state_code].filter(Boolean).join(', ')}</th>
                  <td>{r.suppressed ? SMALL : num(r.artisans)}</td>
                  <td>{r.suppressed ? SMALL : num(r.started)}</td>
                  <td>{r.suppressed ? SMALL : num(r.half_way)}</td>
                  <td>{r.suppressed ? SMALL : num(r.certified)}</td>
                </tr>
              {:else}
                <tr><td colspan="5">{t('insights.noData')}</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .impact {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .impact h1 {
    margin: 0;
  }

  .impact h2 {
    margin: 0;
    font-size: var(--k-text-lg);
  }

  .impact__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .impact__note {
    margin: 0;
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: var(--k-accent-warning-bg);
    color: var(--k-accent-warning-text);
  }

  .impact__error {
    margin: 0;
    color: var(--k-accent-danger);
  }

  .impact__filters {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    align-items: end;
  }

  .impact__filters :global(.k-field-group) {
    margin-block-end: 0;
  }

  .impact__month {
    inline-size: 100%;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm, 0.375rem);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
  }

  .impact__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    grid-column: 1 / -1;
  }

  .kpis {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    margin: 0;
  }

  .kpi {
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
  }

  .kpi dt {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .kpi dd {
    margin: 0;
    font-size: var(--k-text-xl);
    font-weight: 800;
    font-variant-numeric: tabular-nums;
  }

  .impact__section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .impact__section-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
  }

  .impact__scroll {
    overflow-x: auto;
  }

  .impact__table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .impact__table th,
  .impact__table td {
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    text-align: start;
    font-variant-numeric: tabular-nums;
  }
</style>
