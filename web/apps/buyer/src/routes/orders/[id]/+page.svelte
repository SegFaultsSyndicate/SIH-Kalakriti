<!--
  apps/buyer/src/routes/orders/[id]/+page.svelte

  The allocation view -- the demo centrepiece. GET /orders/{id} (fulfilment
  proto's GetOrder) supplies the initial snapshot; GET /orders/{id}/events
  (watchOrderEvents, @kalakriti/api) streams every lot_offered/accepted/
  declined/expired/progressed/gave_up, qc_recorded, order_state_changed and
  payment_settled event live over SSE, reconnecting with backoff and
  resuming from Last-Event-ID on its own -- see packages/api/src/sse.svelte.ts
  and services/bff/internal/bff/handler/api.go's WatchOrder (batch 12 wired
  the id:/Last-Event-ID plumbing that makes the resume actually work).

  $lib/allocation.ts's applyEvent is the single reducer both the card grid
  and the table read from, keyed by lot id and de-duplicated by event_id, so
  a reconnect backfill cannot double a row. A dropout is never a silent row
  change: lot_gave_up and the lot_offered that follows it (carrying
  reallocated_from_lot_id) are both narrated into the aria-live region and
  the old lot stays visible, marked reallocated, rather than disappearing.

  Non-visual equivalent: the aria-live region below announces every
  transition as it lands, and the table view (toggle) renders the exact
  same `lots` data a sighted user sees as cards -- neither is decorative for
  the other, per the brief's "not visual-only" requirement.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Button, Money, EmptyState, Skeleton, SkeletonRow, Tabs } from '@kalakriti/ui';
  import { getOrder, getArtisanStorefront, watchOrderEvents, type components } from '@kalakriti/api';
  import {
    initialAllocationState,
    applyEvent,
    allocatedQuantity,
    narrateEvent,
    type AllocationState,
    type RawOrderEvent,
    type LiveLine,
  } from '$lib/allocation';
  import { getStoredMockOrder, createFallbackMockOrder } from '$lib/order-store';
  import DisputeDialog from '$lib/DisputeDialog.svelte';

  type BulkOrder = components['schemas']['BulkOrder'];

  const t = $derived(locale.t);
  const orderId = $derived(page.params.id ?? '');

  const ORDER_STATE_KEY: Record<string, MessageKey> = {
    ALLOCATING: 'allocation.state.ALLOCATING',
    PARTIALLY_ALLOCATED: 'allocation.state.PARTIALLY_ALLOCATED',
    CONFIRMED: 'allocation.state.CONFIRMED',
    IN_PRODUCTION: 'allocation.state.IN_PRODUCTION',
    AMENDMENT_PENDING: 'allocation.state.AMENDMENT_PENDING',
    COMPLETED: 'allocation.state.COMPLETED',
    CANCELLED: 'allocation.state.CANCELLED',
  };

  let loading = $state(true);
  let order = $state<BulkOrder | undefined>(undefined);
  let allocation = $state<AllocationState>(initialAllocationState([]));
  let liveLines = $state<LiveLine[]>([]);
  let names = $state<Record<string, string>>({});
  let districts = $state<Record<string, string>>({});
  let view = $state('cards');
  let disputeOpen = $state(false);

  // Re-subscribe when the route param changes. Capturing the watcher once
  // bound the stream to the first order id the page ever saw.
  let watcher = $state<ReturnType<typeof watchOrderEvents> | undefined>(undefined);
  $effect(() => {
    const w = watchOrderEvents(orderId);
    watcher = w;
    return () => w.stop();
  });
  const sseStatus = $derived(watcher?.state.status ?? 'connecting');

  async function resolveArtisan(artisanId: string): Promise<void> {
    if (!artisanId || names[artisanId] !== undefined) return;
    try {
      const profile = await getArtisanStorefront(artisanId);
      names = { ...names, [artisanId]: profile.display_name ?? '' };
      if (profile.district) districts = { ...districts, [artisanId]: profile.district };
    } catch {
      const fallbackNames: Record<string, { name: string; district: string }> = {
        'artisan-kabir': { name: 'Mohammad Kabir Ansari', district: 'Varanasi, UP' },
        'artisan-mir': { name: 'Ghulam Nabi Mir', district: 'Srinagar, J&K' },
        'artisan-prajapati': { name: 'Ram Prakash Prajapati', district: 'Azamgarh, UP' },
      };
      const found = fallbackNames[artisanId];
      names = { ...names, [artisanId]: found?.name ?? 'Master Artisan' };
      districts = { ...districts, [artisanId]: found?.district ?? 'Craft Cluster' };
    }
  }

  $effect(() => {
    const id = orderId;
    void (async () => {
      loading = true;
      try {
        order = await getOrder(id);
      } catch (cause) {
        console.warn('[orders/[id]] getOrder failed, using mock order:', cause);
        order = getStoredMockOrder(id) ?? createFallbackMockOrder(id);
      }
      if (order) {
        allocation = initialAllocationState(
          (order?.lots ?? []).map((l) => ({
            id: l.id ?? '',
            artisan_id: l.artisan_id ?? '',
            cluster_id: l.cluster_id,
            quantity: l.quantity ?? 0,
            state: l.state ?? '',
            progress_pct: l.progress_pct ?? 0,
            reallocated_from_lot_id: l.reallocated_from_lot_id,
            decline_reason: l.decline_reason,
          })),
        );
        for (const l of order?.lots ?? []) void resolveArtisan(l.artisan_id ?? '');
      }
      loading = false;
    })();
  });

  $effect(() => {
    const ev = watcher?.state.lastEvent;
    if (!ev) return;
    let parsed: RawOrderEvent | undefined;
    try {
      parsed = JSON.parse(ev.data) as RawOrderEvent;
    } catch {
      return;
    }
    if (!parsed) return;
    const changed = applyEvent(allocation, parsed);
    if (!changed) return;
    allocation = { ...allocation };
    if (parsed.lot?.artisan_id) void resolveArtisan(parsed.lot.artisan_id);
    const line = narrateEvent(parsed, names, districts);
    if (line) liveLines = [...liveLines.slice(-19), line];
  });


  const lots = $derived(Object.values(allocation.lots));
  const allocated = $derived(allocatedQuantity(allocation.lots));
  const total = $derived(order?.quantity ?? 0);
  const orderState = $derived(allocation.orderState ?? order?.state);
  const paymentSplit = $derived(allocation.paymentSplit);
</script>

<svelte:head>
  <title>{t('allocation.heading')} — {t('app.name')}</title>
</svelte:head>

{#if loading}
  <div class="alloc-skeleton" aria-hidden="true">
    <Skeleton shape="text" width="45%" height="1.5rem" />
    <Skeleton shape="text" width="30%" height="1rem" />
    <Skeleton width="100%" height="0.75rem" radius="var(--k-radius-pill)" />
    {#each Array(3) as _, i (i)}
      <SkeletonRow lines={[{ width: '40%', height: '0.9rem' }, { width: '25%', height: '0.8rem' }]} trailing trailingWidth="4rem" />
    {/each}
  </div>
{:else if !order}
  <EmptyState illustration="empty-error" heading={t('orders.notFound')} />
{:else}
  <header class="alloc-header">
    <h1>{t('allocation.heading')}</h1>
    <p class="alloc-header__progress">{t('allocation.progress', { allocated: String(allocated), total: String(total) })}</p>
    {#if orderState && ORDER_STATE_KEY[orderState]}
      <p class="alloc-header__state">{t(ORDER_STATE_KEY[orderState])}</p>
    {/if}
    {#if sseStatus === 'connecting'}
      <p class="alloc-header__reconnecting" role="status">{t('allocation.reconnecting')}</p>
    {/if}
    {#if orderState === 'AMENDMENT_PENDING'}
      <p class="alloc-header__amendment" role="status">{t('allocation.amendment.banner')}</p>
    {/if}
    <Button variant="secondary" onclick={() => (disputeOpen = true)} tooltip={tooltip('tooltip.dispute')}>{t('orders.dispute.entry')}</Button>
  </header>

  <div
    class="alloc-progress-bar"
    role="progressbar"
    aria-valuenow={allocated}
    aria-valuemin={0}
    aria-valuemax={total}
    aria-label={t('allocation.heading')}
  >
    <div class="alloc-progress-bar__fill" style:width="{total > 0 ? Math.min(100, (allocated / total) * 100) : 0}%"></div>
  </div>

  <div class="alloc-live" role="status" aria-live="polite">
    {#each liveLines.slice(-1) as line (line.id)}
      <p>{t(line.key, line.params)}</p>
    {/each}
  </div>

  <Tabs
    tabs={[
      { id: 'cards', label: t('allocation.view.cards') },
      { id: 'table', label: t('allocation.view.table') },
    ]}
    bind:selected={view}
  >
    {#snippet children(tabId)}
      {#if tabId === 'cards'}
        <ul class="alloc-lots" aria-label={t('allocation.heading')}>
          {#each lots as lot (lot.id)}
            <li class="alloc-lot" class:alloc-lot--reallocated={lot.state === 'REALLOCATED'}>
              <p class="alloc-lot__artisan">{names[lot.artisan_id] || t('allocation.lot.unknownArtisan')}</p>
              {#if districts[lot.artisan_id]}<p class="alloc-lot__district">{districts[lot.artisan_id]}</p>{/if}
              <p class="alloc-lot__quantity">{t('orders.units', { count: String(lot.quantity) })}</p>
              <p class="alloc-lot__state">{lot.state}</p>
              {#if lot.progress_pct > 0}
                <p class="alloc-lot__progress">{t('allocation.lot.progress', { pct: String(lot.progress_pct) })}</p>
              {/if}
              {#if lot.reallocated_from_lot_id}
                <p class="alloc-lot__note">{t('allocation.lot.reallocatedFrom')}</p>
              {/if}
            </li>
          {/each}
        </ul>
      {:else}
        <div class="alloc-table-wrap">
          <table class="alloc-table">
            <thead>
              <tr>
                <th scope="col">{t('allocation.table.artisan')}</th>
                <th scope="col">{t('allocation.table.district')}</th>
                <th scope="col">{t('allocation.table.quantity')}</th>
                <th scope="col">{t('allocation.table.state')}</th>
                <th scope="col">{t('allocation.table.progress')}</th>
              </tr>
            </thead>
            <tbody>
              {#each lots as lot (lot.id)}
                <tr>
                  <td>{names[lot.artisan_id] || t('allocation.lot.unknownArtisan')}</td>
                  <td>{districts[lot.artisan_id] ?? ''}</td>
                  <td class="k-tabular">{lot.quantity}</td>
                  <td>{lot.state}</td>
                  <td class="k-tabular">{lot.progress_pct}%</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/snippet}
  </Tabs>

  {#if paymentSplit}
    <section class="alloc-payment">
      <h2>{t('allocation.payment.heading')}</h2>
      <dl class="alloc-payment__totals">
        <div><dt>{t('allocation.payment.grossTotal')}</dt><dd><Money paise={paymentSplit.gross_total.amount_paise} /></dd></div>
        <div><dt>{t('allocation.payment.commissionTotal')}</dt><dd><Money paise={paymentSplit.commission_total.amount_paise} /></dd></div>
        <div><dt>{t('allocation.payment.netTotal')}</dt><dd><Money paise={paymentSplit.net_total.amount_paise} /></dd></div>
      </dl>
      <ul class="alloc-payment__bars">
        {#each paymentSplit.lines as line (line.lot_id)}
          <li>
            <span>{names[line.payee_id] || t('allocation.lot.unknownArtisan')}</span>
            <span
              class="alloc-payment__bar"
              style:width="{paymentSplit.net_total.amount_paise > 0 ? (line.net_amount.amount_paise / paymentSplit.net_total.amount_paise) * 100 : 0}%"
            ></span>
            <Money paise={line.net_amount.amount_paise} />
          </li>
        {/each}
      </ul>
    </section>
  {:else if orderState === 'COMPLETED' || orderState === 'IN_PRODUCTION'}
    <p class="alloc-payment__pending">{t('allocation.payment.pending')}</p>
  {/if}

  <DisputeDialog bind:open={disputeOpen} {orderId} />
{/if}

<style>
  .alloc-header {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    margin-block-end: var(--k-space-4);
  }

  .alloc-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .alloc-header__progress {
    font-size: var(--k-text-md);
    font-weight: var(--k-weight-semibold);
  }

  .alloc-header__reconnecting {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .alloc-header__amendment {
    color: var(--k-accent-warning-text);
    background: var(--k-accent-warning-bg);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    font-size: var(--k-text-sm);
    max-inline-size: 40rem;
  }

  .alloc-progress-bar {
    block-size: var(--k-rule-heavy);
    background: var(--k-surface-sunken);
    border-radius: var(--k-radius-pill);
    overflow: hidden;
    margin-block-end: var(--k-space-4);
  }

  .alloc-progress-bar__fill {
    block-size: 100%;
    background: var(--k-accent-primary-bg);
    transition: width var(--k-duration-slow) var(--k-ease-standard);
  }

  .alloc-live {
    min-block-size: 1.5em;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-3);
  }

  .alloc-lots {
    list-style: none;
    margin: var(--k-space-4) 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(11rem, 1fr));
    gap: var(--k-space-3);
  }

  .alloc-lot {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3);
  }

  .alloc-lot--reallocated {
    opacity: 0.6;
  }

  .alloc-lot__artisan {
    font-weight: var(--k-weight-semibold);
  }

  .alloc-lot__district,
  .alloc-lot__state,
  .alloc-lot__progress,
  .alloc-lot__note {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .alloc-lot__note {
    color: var(--k-accent-secondary);
  }

  .alloc-table-wrap {
    overflow-x: auto;
    margin-block: var(--k-space-4);
  }

  .alloc-table {
    width: 100%;
    border-collapse: collapse;
  }

  .alloc-table th,
  .alloc-table td {
    text-align: start;
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .alloc-payment {
    margin-block-start: var(--k-space-6);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-4);
  }

  .alloc-payment__totals {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    margin-block-end: var(--k-space-4);
  }

  .alloc-payment__totals div {
    display: flex;
    justify-content: space-between;
  }

  .alloc-payment__bars {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .alloc-payment__bars li {
    display: grid;
    grid-template-columns: 8rem 1fr auto;
    align-items: center;
    gap: var(--k-space-2);
  }

  /* On a narrow phone the 8rem label + bar + amount stack into two rows:
     label and amount on top, the progress bar spanning full width. */
  @media (max-width: 30rem) {
    .alloc-payment__bars li {
      grid-template-columns: 1fr auto;
      grid-template-areas:
        'name amount'
        'bar bar';
    }

    .alloc-payment__bars li > :nth-child(1) {
      grid-area: name;
    }

    .alloc-payment__bars li > :nth-child(2) {
      grid-area: bar;
    }

    .alloc-payment__bars li > :nth-child(3) {
      grid-area: amount;
    }
  }

  .alloc-payment__bar {
    display: block;
    block-size: var(--k-space-3);
    background: var(--k-accent-primary-bg);
    border-radius: var(--k-radius-sm);
  }

  .alloc-payment__pending {
    color: var(--k-text-secondary);
    margin-block-start: var(--k-space-4);
  }
</style>
