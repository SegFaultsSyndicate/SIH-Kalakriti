<!--
  apps/artisan/src/routes/earnings/+page.svelte

  Earnings over time, per order, and income statement generation. Real,
  online-only endpoints throughout: POST /statements (generate) returns the
  full record immediately -- statement_id, short_code, download_url,
  verification_url -- because insight-svc has no fetch-by-id RPC (see
  client/insight.go's GetStatement doc comment); GET /statements (newest
  first) is the earnings-over-time list.

  Platform fee threshold: the brief asks for the fee shown as zero below a
  stated threshold. No such threshold exists anywhere in the real backend --
  commission is a flat calculation with no minimum-order exemption field
  (checked collab-svc's compensation.go and payment.go; nothing). Rather
  than invent a number for a money-correctness-sensitive figure, this screen
  shows the real fee_amount insight-svc actually charged, unqualified. See
  ml_wiring.md's batch 10 section.

  "Per order": insight-svc's own granularity is a generated statement over a
  period (order_count, not a line per order), so the per-period list below
  is the real granularity. The per-order list beneath it uses this device's
  own local order cache (completed lots only, real lot_value each) as a
  genuinely per-order complement, not a fabrication.
-->
<script lang="ts">
  import { locale, formatDate, tooltip } from '@kalakriti/i18n';
  import { Button, Input, Label, Money, showToast } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { Icon } from '@kalakriti/icons';
  import { generateStatement, listIncomeStatements, type components } from '@kalakriti/api';
  import { cachedOrders, network, type BulkOrder } from '$lib/orders';
  import { getArtisanId } from '$lib/registration';
  import IncomeGrowthChart from '$lib/IncomeGrowthChart.svelte';

  type IncomeStatementSummary = components['schemas']['IncomeStatementSummary'];

  const t = $derived(locale.t);

  let loading = $state(true);
  let statements = $state<IncomeStatementSummary[]>([]);
  let orders = $state<BulkOrder[]>([]);
  let artisanId = $state<string | undefined>(undefined);

  let start = $state('');
  let end = $state('');
  let generating = $state(false);
  let latest = $state<IncomeStatementSummary | undefined>(undefined);

  async function load(): Promise<void> {
    loading = true;
    artisanId = await getArtisanId();
    orders = await cachedOrders();
    if (network.online) {
      try {
        const res = await listIncomeStatements();
        statements = res.statements ?? [];
      } catch {
        /* offline or transient -- the range picker still works when back online. */
      }
    }
    loading = false;
  }

  $effect(() => {
    void load();
  });

  const completedLotRows = $derived.by(() => {
    if (!artisanId) return [];
    return orders
      .flatMap((o) => (o.lots ?? []).map((l) => ({ order: o, lot: l })))
      .filter((r) => r.lot.artisan_id === artisanId && r.lot.state === 'COMPLETED');
  });

  async function onGenerate(): Promise<void> {
    if (!start || !end) return;
    generating = true;
    try {
      const res = await generateStatement({ start, end });
      latest = {
        statement_id: res.statement_id,
        short_code: res.short_code,
        download_url: res.download_url,
        period_start: start,
        period_end: end,
      } as IncomeStatementSummary;
      statements = [latest, ...statements];
      showToast({ variant: 'success', message: t('earnings.generated') });
    } catch {
      showToast({ variant: 'error', message: t('api.error.unknown') });
    } finally {
      generating = false;
    }
  }

  function whatsappShareUrl(url: string): string {
    return `https://wa.me/?text=${encodeURIComponent(url)}`;
  }
</script>

<svelte:head>
  <title>{t('earnings.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="earnings-page">
  <h1>{t('earnings.heading')}</h1>

  <IncomeGrowthChart />

  <section class="earnings-page__generate">
    <h2>{t('earnings.statement.heading')}</h2>
    <p class="earnings-page__purpose">{t('earnings.statement.purpose')}</p>

    <div class="earnings-page__range">
      <div>
        <Label for="range-start">{t('earnings.statement.start')}</Label>
        <Input id="range-start" type="date" bind:value={start} />
      </div>
      <div>
        <Label for="range-end">{t('earnings.statement.end')}</Label>
        <Input id="range-end" type="date" bind:value={end} />
      </div>
    </div>

    <Button
      onclick={onGenerate}
      disabled={!start || !end || !network.online || generating}
      loading={generating}
      tooltip={tooltip('tooltip.generateStatement')}
    >
      {t('earnings.statement.generate')}
    </Button>
    {#if !network.online}
      <p class="earnings-page__offline-note">{t('earnings.offlineNote')}</p>
    {/if}

    {#if latest}
      <Card variant="hairline" element="div" class="earnings-page__latest">
        <p>{t('earnings.statement.ready')}</p>
        <div class="earnings-page__latest-actions">
          <a href={latest.download_url} target="_blank" rel="noreferrer" class="earnings-page__action-link">
            <Icon name="download" />
            {t('earnings.statement.download')}
          </a>
          {#if latest.download_url}
            <a
              href={whatsappShareUrl(latest.download_url)}
              target="_blank"
              rel="noreferrer"
              class="earnings-page__action-link"
            >
              <Icon name="whatsapp" />
              {t('earnings.statement.share')}
            </a>
          {/if}
        </div>
      </Card>
    {/if}
  </section>

  <section class="earnings-page__history">
    <h2>{t('earnings.history.heading')}</h2>
    {#if loading}
      <Skeleton shape="card" height="4rem" />
    {:else if statements.length === 0}
      <p class="earnings-page__empty">{t('earnings.history.empty')}</p>
    {:else}
      <ul class="earnings-page__rows" role="list">
        {#each statements as statement (statement.statement_id)}
          <li>
            <Card variant="hairline" element="div" class="earnings-page__row">
              <div class="earnings-page__row-body">
                <p>
                  {statement.period_start
                    ? formatDate(statement.period_start, locale.code, { dateStyle: 'medium' })
                    : ''}
                  {' – '}
                  {statement.period_end ? formatDate(statement.period_end, locale.code, { dateStyle: 'medium' }) : ''}
                </p>
                {#if statement.order_count != null}
                  <p class="earnings-page__row-meta">{t('earnings.history.orderCount', { count: String(statement.order_count) })}</p>
                {/if}
                <div class="earnings-page__row-amounts">
                  {#if statement.gross_amount?.amount_paise != null}
                    <span>{t('earnings.history.gross')} <Money paise={statement.gross_amount.amount_paise} /></span>
                  {/if}
                  {#if statement.fee_amount?.amount_paise != null}
                    <span>{t('earnings.history.fee')} <Money paise={statement.fee_amount.amount_paise} /></span>
                  {/if}
                  {#if statement.net_amount?.amount_paise != null}
                    <span class="earnings-page__row-net">
                      {t('earnings.history.net')} <Money paise={statement.net_amount.amount_paise} />
                    </span>
                  {/if}
                </div>
              </div>
              {#if statement.download_url}
                <a href={statement.download_url} target="_blank" rel="noreferrer" aria-label={t('earnings.statement.download')}>
                  <Icon name="download" />
                </a>
              {/if}
            </Card>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  {#if completedLotRows.length > 0}
    <section class="earnings-page__per-order">
      <h2>{t('earnings.perOrder.heading')}</h2>
      <ul class="earnings-page__rows" role="list">
        {#each completedLotRows as row (row.lot.id)}
          <li>
            <Card variant="hairline" element="div" class="earnings-page__row">
              <span>{t('orders.units', { count: String(row.lot.quantity ?? '') })}</span>
              {#if row.lot.lot_value?.amount_paise != null}
                <Money paise={row.lot.lot_value.amount_paise} />
              {/if}
            </Card>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
</main>

<style>
  .earnings-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-5);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .earnings-page__generate {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .earnings-page__purpose {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .earnings-page__range {
    display: flex;
    gap: var(--k-space-3);
  }

  @media (max-width: 30rem) {
    .earnings-page__range {
      flex-direction: column;
    }
  }

  .earnings-page__offline-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  :global(.earnings-page__latest) {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
  }

  .earnings-page__latest-actions {
    display: flex;
    gap: var(--k-space-3);
  }

  .earnings-page__action-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-accent-primary-text);
    text-decoration: none;
  }

  .earnings-page__history,
  .earnings-page__per-order {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .earnings-page__empty {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .earnings-page__rows {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  :global(.earnings-page__row) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
  }

  .earnings-page__row-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .earnings-page__row-meta {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .earnings-page__row-amounts {
    display: flex;
    gap: var(--k-space-3);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .earnings-page__row-net {
    font-weight: 700;
    color: var(--k-text-primary);
  }
</style>
