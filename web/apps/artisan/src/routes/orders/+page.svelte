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
  import { EmptyState, Money, Card, SkeletonRow } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import StateBadge from '$lib/StateBadge.svelte';
  import { getArtisanId } from '$lib/registration';
  import { localPrimaryImageUrl } from '$lib/listings';
  import {
    cachedOrders,
    getCachedOrdersSync,
    refreshTouchedOrders,
    myLots,
    needsAction,
    network,
    type BulkOrder,
    type OrderLot,
  } from '$lib/orders';
  import { getSihListingTitleKey, getSihMockOrders } from '$lib/sih-my-works';

  const t = $derived(locale.t);

  const initialOrders = getCachedOrdersSync();
  let loading = $state(initialOrders === null);
  let orders = $state<BulkOrder[]>(initialOrders ?? []);
  let artisanId = $state<string | undefined>(undefined);
  let orderImages = $state<Record<string, string>>({});
  let orderImageUrls: string[] = [];

  async function loadOrderImages(currentOrders: BulkOrder[]): Promise<void> {
    for (const url of orderImageUrls) URL.revokeObjectURL(url);
    orderImageUrls = [];
    const images: Record<string, string> = {};
    for (const order of currentOrders) {
      if (!order.id || !order.listing_id) continue;
      const imageUrl = await localPrimaryImageUrl(order.listing_id);
      if (!imageUrl) continue;
      images[order.id] = imageUrl;
      orderImageUrls.push(imageUrl);
    }
    orderImages = images;
  }

  async function load(): Promise<void> {
    if (initialOrders === null) loading = true;
    artisanId = await getArtisanId();
    orders = await cachedOrders();
    await loadOrderImages(orders);
    loading = false;
    await refreshTouchedOrders();
    orders = await cachedOrders();
    await loadOrderImages(orders);
  }

  $effect(() => {
    void load();
    return () => {
      for (const url of orderImageUrls) URL.revokeObjectURL(url);
    };
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

<div class="orders-page">
  <h1>{t('orders.heading')}</h1>

  {#if !network.online}
    <p class="orders-page__offline-note">{t('orders.offlineNote')}</p>
  {/if}

  {#if loading && rows.length === 0}
    <div class="orders-page__skeletons">
      {#each Array(3) as _, i (i)}
        <SkeletonRow
          thumbnail
          thumbnailSize="3.5rem"
          lines={[{ width: '40%', height: '0.9rem' }, { width: '55%', height: '0.85rem' }, { width: '30%', height: '1rem' }]}
          trailing
          trailingWidth="1rem"
          trailingHeight="1rem"
        />
      {/each}
    </div>
  {:else if rows.length === 0}
    {@const mockOrders = getSihMockOrders()}
    {#if mockOrders.length > 0}
      <!-- SIH Demo: show mock order history when the backend has no real orders -->
      <ul class="orders-page__rows" role="list">
        {#each mockOrders as order (order.id)}
          {@const listingTitleKey = getSihListingTitleKey(order.listingId)}
          <li>
            <Card
              variant="hairline"
              element={order.id === 'sih-order-003' ? 'a' : 'div'}
              class="orders-page__row {order.id === 'sih-order-003' ? 'orders-page__row-link' : ''}"
              href={order.id === 'sih-order-003' ? `/orders/${order.id}` : undefined}
            >
              {#if order.imageUrl}
                <img class="orders-page__thumb" src={order.imageUrl} alt="" />
              {:else}
                <div class="orders-page__thumb orders-page__thumb--placeholder" aria-hidden="true">
                  <Icon name="image" />
                </div>
              {/if}
              <div class="orders-page__row-body">
                <p class="orders-page__row-kind orders-page__row-listing">
                  {listingTitleKey ? t(listingTitleKey) : order.listingTitle}
                </p>
                <p class="orders-page__row-buyer">{order.buyerName}</p>
                <div class="orders-page__row-state">
                  <StateBadge group={order.state === 'OFFERED' ? 'needsAttention' : order.state === 'IN_PROGRESS' ? 'pending' : 'published'} />
                  <span>{t('orders.units', { count: String(order.quantity) })}</span>
                </div>
                <Money paise={order.totalPaise} />
              </div>
              <Icon name="chevron-right" aria-hidden="true" />
            </Card>
          </li>
        {/each}
      </ul>
    {:else}
      <EmptyState illustration="empty-no-orders" heading={t('orders.empty')} body={t('orders.emptyBody')} />
    {/if}
  {:else}
    <ul class="orders-page__rows" role="list">
      {#each rows as row (row.lot.id)}
        <li>
          <a href={lotHref(row)} class="orders-page__row-link">
            <Card variant="hairline" element="div" class="orders-page__row">
              {#if orderImages[row.order.id ?? '']}
                <img class="orders-page__thumb" src={orderImages[row.order.id ?? '']} alt="" />
              {:else}
                <div class="orders-page__thumb orders-page__thumb--placeholder" aria-hidden="true">
                  <Icon name="image" />
                </div>
              {/if}
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
</div>

<style>
  .orders-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-4);
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

  :global(.orders-page__row-link) {
    color: inherit;
    text-decoration: none;
  }

  :global(.orders-page__row-link:visited) {
    color: inherit;
  }

  :global(.orders-page__row-link.k-card:hover),
  :global(.orders-page__row-link.k-card:focus-visible),
  :global(.orders-page__row-link.orders-page__row:hover),
  :global(.orders-page__row-link.orders-page__row:focus-visible),
  :global(.orders-page__row-link:hover .orders-page__row),
  :global(.orders-page__row-link:focus-visible .orders-page__row) {
    border-color: var(--k-accent-primary-bg);
    background-color: color-mix(in srgb, var(--k-accent-primary-bg) 8%, var(--k-surface-raised));
  }

  .orders-page__row-link:focus-visible {
    outline: 2px solid var(--k-accent-primary-bg);
    outline-offset: 2px;
  }

  :global(.orders-page__row-link:hover svg),
  :global(.orders-page__row-link:focus-visible svg) {
    color: var(--k-accent-primary-text);
  }

  :global(.orders-page__row) {
    display: flex;
    align-items: stretch;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding: var(--k-space-2);
  }

  :global(.orders-page__row > svg) {
    align-self: center;
    flex-shrink: 0;
  }

  @media (max-width: 30rem) {
    .orders-page {
      padding: var(--k-space-3);
    }
  }

  .orders-page__row-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    min-inline-size: 0;
  }

  .orders-page__thumb {
    flex-shrink: 0;
    inline-size: 4.5rem;
    block-size: auto;
    min-block-size: 4.5rem;
    align-self: stretch;
    border-radius: var(--k-radius-md);
    object-fit: cover;
  }

  .orders-page__thumb--placeholder {
    display: grid;
    place-items: center;
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
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

  .orders-page__row-listing {
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-inline-size: 100%;
  }

  .orders-page__row-buyer {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }
</style>
