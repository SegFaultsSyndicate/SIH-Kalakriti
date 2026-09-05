<!--
  apps/artisan/src/routes/orders/+page.svelte

  The artisan's commercial life: every order/lot this device knows about,
  sorted by what needs action. There is no GET /orders (list) anywhere in the
  backend (see $lib/orders.ts's header, and ml_wiring.md's batch 10 section)
  -- this reads the local touched-orders cache and refreshes each entry when
  online, same online-first-with-local-fallback pattern batch 9 used for
  /listings.

  "Direct orders and collective lots in one list, clearly distinguished" --
  the real backend models only one thing, a bulk order split into
  per-artisan lots (fulfilment.proto has no separate "direct order" concept
  at all). Read honestly: an order with exactly one lot on it is, from this
  artisan's side, indistinguishable from a direct one-to-one order, so that's
  the distinction drawn here -- single-lot orders shown as "direct", multi-lot
  orders as "collective". Documented as a judgment call, not a fabricated
  backend concept.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { EmptyState, Money, Card, Skeleton } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import StateBadge from '$lib/StateBadge.svelte';
  import { getArtisanId } from '$lib/registration';
  import {
    cachedOrders,
    refreshTouchedOrders,
    myLots,
    needsAction,
    network,
    type BulkOrder,
    type OrderLot,
  } from '$lib/orders';

  const t = $derived(locale.t);

  let loading = $state(true);
  let orders = $state<BulkOrder[]>([]);
  let artisanId = $state<string | undefined>(undefined);

  async function load(): Promise<void> {
    loading = true;
    artisanId = await getArtisanId();
    orders = await cachedOrders();
    loading = false;
    await refreshTouchedOrders();
    orders = await cachedOrders();
  }

  $effect(() => {
    void load();
  });

  interface Row {
    order: BulkOrder;
    lot: OrderLot;
    kind: 'direct' | 'collective';
  }

  const rows = $derived.by((): Row[] => {
    if (!artisanId) return [];
    const lots = myLots(orders, artisanId);
    return lots
      .map((lot) => {
        const order = orders.find((o) => o.id === lot.bulk_order_id);
        if (!order) return undefined;
        return { order, lot, kind: (order.lots?.length ?? 1) > 1 ? 'collective' : 'direct' } as Row;
      })
      .filter((r): r is Row => r !== undefined)
      .sort((a, b) => Number(needsAction(b.lot)) - Number(needsAction(a.lot)));
  });

  function lotHref(row: Row): string {
    if (row.lot.state === 'OFFERED') return `/orders/${row.order.id}/lots/${row.lot.id}/offer`;
    return `/orders/${row.order.id}/lots/${row.lot.id}`;
  }
</script>

<svelte:head>
  <title>{t('orders.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="orders-page">
  <h1>{t('orders.heading')}</h1>

  {#if !network.online}
    <p class="orders-page__offline-note">{t('orders.offlineNote')}</p>
  {/if}

  {#if loading && rows.length === 0}
    <div class="orders-page__skeletons">
      <Skeleton shape="card" height="4rem" />
      <Skeleton shape="card" height="4rem" />
    </div>
  {:else if rows.length === 0}
    <EmptyState illustration="empty-no-orders" heading={t('orders.empty')} body={t('orders.emptyBody')} />
  {:else}
    <ul class="orders-page__rows" role="list">
      {#each rows as row (row.lot.id)}
        <li>
          <a href={lotHref(row)} class="orders-page__row-link">
            <Card variant="hairline" element="div" class="orders-page__row">
              <div class="orders-page__row-body">
                <p class="orders-page__row-kind">
                  {row.kind === 'direct' ? t('orders.kind.direct') : t('orders.kind.collective')}
                </p>
                <div class="orders-page__row-state">
                  <StateBadge group={needsAction(row.lot) ? 'needsAttention' : 'pending'} />
                  {#if row.lot.quantity != null}
                    <span>{t('orders.units', { count: String(row.lot.quantity) })}</span>
                  {/if}
                </div>
                {#if row.lot.lot_value?.amount_paise != null}
                  <Money paise={row.lot.lot_value.amount_paise} />
                {/if}
              </div>
              <Icon name="chevron-right" />
            </Card>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</main>

<style>
  .orders-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .orders-page__offline-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .orders-page__skeletons {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .orders-page__rows {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .orders-page__row-link {
    color: inherit;
    text-decoration: none;
  }

  :global(.orders-page__row) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
  }

  .orders-page__row-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .orders-page__row-kind {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .orders-page__row-state {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-weight: 600;
  }
</style>
