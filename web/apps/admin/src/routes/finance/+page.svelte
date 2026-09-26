<!--
  apps/admin/src/routes/finance/+page.svelte

  F12 review queue: officers verify or reject the finance-corporation links
  artisans self-reported. Only the last 4 characters of a loan reference
  exist anywhere (core-svc hashes the rest on arrival), so that is all this
  page can show. The corporation summary on top is the same k>=5-suppressed
  aggregate the impact dashboard uses; the queue itself is individual rows
  by necessity -- a reviewer has to see whose link they are verifying. When
  VITE_USE_MOCKS=1 is set, $lib/stubs.ts appends sample rows to any section
  real rows don't already cover (deduped) so the page never looks empty;
  unreachable endpoints fall back to the stubs and review actions are
  applied to the local queue (a "[mock fallback]" console.warn is logged).
-->
<script lang="ts">
  import { locale, formatDate, formatNumber, type MessageKey } from '@kalakriti/i18n';
  import { Button, FieldGroup, Input, Money, Select, Sheet, showToast } from '@kalakriti/ui';
  import {
    session,
    listFinanceLinksForReview,
    reviewFinanceLink,
    getFinanceCoverage,
    ApiError,
    messageKeyFor,
    type FinanceLinkForReview,
    type FinanceCoverageRow,
  } from '@kalakriti/api';
  import { FINANCE_LINKS_FOR_REVIEW, FINANCE_COVERAGE, mergeWithStubs } from '$lib/stubs';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY' || role === 'CLUSTER_OFFICER');

  const STATUS: Record<string, MessageKey> = {
    SELF_REPORTED: 'finance.status.self',
    VERIFIED: 'finance.status.verified',
    REJECTED: 'finance.status.rejected',
  };
  const corpLabel = (c?: string): string => (c === 'OTHER' ? t('impact.other') : (c ?? '').replace('_', '-'));

  let status = $state('SELF_REPORTED');
  let stateCode = $state('');
  let district = $state('');
  let links = $state<FinanceLinkForReview[]>([]);
  let summary = $state<FinanceCoverageRow[]>([]);
  let loading = $state(false);
  let error = $state('');

  let open = $state(false);
  let current = $state<FinanceLinkForReview | null>(null);
  let reason = $state('');
  let saving = $state(false);

  async function load(): Promise<void> {
    loading = true;
    error = '';
    try {
      const filter = { state_code: stateCode.trim() || undefined, district: district.trim() || undefined };
      const [q, s] = await Promise.all([
        listFinanceLinksForReview({ ...filter, status: status || undefined }),
        getFinanceCoverage(filter),
      ]);
      links = mergeWithStubs(q.links ?? [], FINANCE_LINKS_FOR_REVIEW, (x) => x.id);
      summary = mergeWithStubs(s.rows ?? [], FINANCE_COVERAGE, (x) => x.corporation);
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] finance load:', cause);
        links = FINANCE_LINKS_FOR_REVIEW;
        summary = FINANCE_COVERAGE;
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

  function review(link: FinanceLinkForReview): void {
    current = link;
    reason = '';
    open = true;
  }

  async function decide(verified: boolean): Promise<void> {
    if (!current?.id) return;
    saving = true;
    try {
      await reviewFinanceLink(current.id, verified ? { verified: true } : { verified: false, reason: reason.trim() });
      open = false;
      showToast({ message: t('financeAdmin.reviewed'), variant: 'success' });
      await load();
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] reviewFinanceLink:', cause);
        links = links.filter((l) => l.id !== current?.id);
        open = false;
        showToast({ message: t('financeAdmin.reviewed'), variant: 'success' });
      } else {
        showToast({ message: t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown'), variant: 'error' });
      }
    } finally {
      saving = false;
    }
  }
</script>

<svelte:head>
  <title>{t('financeAdmin.title')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="fin">
  <h1>{t('financeAdmin.title')}</h1>

  {#if !authorized}
    <p>{t('admin.home.noAccess')}</p>
  {:else}
    <form
      class="fin__filters"
      onsubmit={(e) => {
        e.preventDefault();
        void load();
      }}
    >
      <FieldGroup label={t('financeAdmin.status')}>
        {#snippet children({ id })}
          <Select
            {id}
            bind:value={status}
            options={[{ value: '', label: t('impact.all') }, ...Object.entries(STATUS).map(([value, key]) => ({ value, label: t(key) }))]}
          />
        {/snippet}
      </FieldGroup>
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
      <Button type="submit" loading={loading}>{t('impact.apply')}</Button>
    </form>

    {#if error}
      <p class="fin__error" role="alert">{error}</p>
    {/if}

    <ul class="fin__summary" role="list">
      {#each summary as r (r.corporation)}
        <li>
          <strong>{corpLabel(r.corporation)}</strong>
          {r.suppressed ? '—' : formatNumber(r.beneficiaries ?? 0, locale.code)}
          <span class="fin__muted">({t('impact.col.verified')}: {r.suppressed ? '—' : formatNumber(r.verified ?? 0, locale.code)})</span>
        </li>
      {/each}
    </ul>

    <div class="fin__scroll">
      <table class="fin__table">
        <thead>
          <tr>
            <th scope="col">{t('financeAdmin.artisan')}</th>
            <th scope="col">{t('insights.district')}</th>
            <th scope="col">{t('impact.filter.corporation')}</th>
            <th scope="col">{t('finance.reference')}</th>
            <th scope="col">{t('finance.coverage.emi')}</th>
            <th scope="col">{t('financeAdmin.status')}</th>
            <th scope="col"><span class="k-visually-hidden">{t('financeAdmin.review')}</span></th>
          </tr>
        </thead>
        <tbody>
          {#each links as l (l.id)}
            <tr>
              <th scope="row">{l.artisan_name}</th>
              <td>{[l.district, l.state_code].filter(Boolean).join(', ')}</td>
              <td>{corpLabel(l.corporation)}</td>
              <td class="fin__mono">•••• {l.reference_last4}</td>
              <td>{#if l.emi_paise}<Money paise={l.emi_paise} />{:else}—{/if}</td>
              <td>{t(STATUS[l.status ?? ''] ?? 'finance.status.self')}</td>
              <td><Button size="sm" variant="secondary" onclick={() => review(l)}>{t('financeAdmin.review')}</Button></td>
            </tr>
          {:else}
            <tr><td colspan="7">{t('financeAdmin.empty')}</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<Sheet bind:open title={t('financeAdmin.review')}>
  {#if current}
    <div class="fin__sheet">
      <dl class="fin__facts">
        <dt>{t('financeAdmin.artisan')}</dt>
        <dd>{current.artisan_name}</dd>
        <dt>{t('impact.filter.corporation')}</dt>
        <dd>{corpLabel(current.corporation)}{current.channelizing_agency ? ` · ${current.channelizing_agency}` : ''}</dd>
        <dt>{t('finance.reference')}</dt>
        <dd class="fin__mono">•••• {current.reference_last4}</dd>
        {#if current.sanctioned_paise}
          <dt>{t('financeAdmin.sanctioned')}</dt>
          <dd><Money paise={current.sanctioned_paise} /></dd>
        {/if}
        {#if current.created_at}
          <dt>{t('financeAdmin.submitted')}</dt>
          <dd>{formatDate(current.created_at, locale.code)}</dd>
        {/if}
      </dl>
      <p class="fin__muted">{t('financeAdmin.howTo')}</p>
      <Button loading={saving} onclick={() => decide(true)}>{t('financeAdmin.verify')}</Button>
      <FieldGroup label={t('financeAdmin.reason')}>
        {#snippet children({ id })}
          <Input {id} bind:value={reason} />
        {/snippet}
      </FieldGroup>
      <Button variant="danger" disabled={!reason.trim()} loading={saving} onclick={() => decide(false)}>{t('financeAdmin.reject')}</Button>
    </div>
  {/if}
</Sheet>

<style>
  .fin {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .fin h1 {
    margin: 0;
  }

  .fin__filters {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    align-items: end;
  }

  .fin__filters :global(.k-field-group) {
    margin-block-end: 0;
  }

  .fin__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .fin__error {
    margin: 0;
    color: var(--k-accent-danger);
  }

  .fin__summary {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2) var(--k-space-4);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .fin__scroll {
    overflow-x: auto;
  }

  .fin__table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .fin__table th,
  .fin__table td {
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    text-align: start;
  }

  .fin__mono {
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .fin__sheet {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .fin__facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--k-space-1) var(--k-space-3);
    margin: 0;
  }

  .fin__facts dt {
    color: var(--k-text-secondary);
  }

  .fin__facts dd {
    margin: 0;
    font-weight: 600;
  }
</style>
