<!--
  apps/buyer/src/routes/orders/+page.svelte

  Buyer order history. fulfilment.proto has no ListOrders-for-a-buyer RPC --
  GetOrder only fetches one order by id, so there is nowhere on the backend
  to page a buyer's own orders from (see $lib/order-store.ts). This device
  remembers the order ids PurchaseForm/BulkOrderWizard created and rehydrates
  each one through the real GetOrder; the list is per-device, said plainly
  in orders.empty.hint rather than presented as a synced account history.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { EmptyState, Skeleton, Money } from '@kalakriti/ui';
  import { getOrder, type components } from '@kalakriti/api';
  import { listRememberedOrders } from '$lib/order-store';

  type BulkOrder = components['schemas']['BulkOrder'];

  const t = $derived(locale.t);

  let loading = $state(true);
  let orders = $state<BulkOrder[]>([]);

  $effect(() => {
    void (async () => {
      loading = true;
      try {
        const ids = listRememberedOrders();
        const fetched = await Promise.allSettled(ids.map((id) => getOrder(id)));
        orders = fetched
          .filter((r): r is PromiseFulfilledResult<BulkOrder> => r.status === 'fulfilled')
          .map((r) => r.value);
      } finally {
        loading = false;
      }
    })();
  });
</script>

<svelte:head>
  <title>{t('buyer.orders.heading')} — {t('app.name')}</title>
</svelte:head>

<h1>{t('buyer.orders.heading')}</h1>

{#if loading}
  <Skeleton shape="card" height="6rem" />
{:else if orders.length === 0}
  <EmptyState illustration="empty-error" heading={t('buyer.orders.empty')} />
  <p class="orders-list__hint">{t('orders.empty.hint')}</p>
{:else}
  <p class="orders-list__hint">{t('orders.empty.hint')}</p>
  <ul class="orders-list">
    {#each orders as order (order.id)}
      <li class="orders-list__item">
        <a href={`/orders/${order.id}`}>
          <span class="orders-list__quantity">{order.quantity} units</span>
          <span class="orders-list__state">{order.state}</span>
          {#if order.total_value}<Money paise={order.total_value.amount_paise ?? 0} />{/if}
          <span class="orders-list__link">{t('orders.viewAllocation')}</span>
        </a>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .orders-list__hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-block-end: var(--k-space-4);
  }

  .orders-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  .orders-list__item a {
    display: flex;
    align-items: center;
    gap: var(--k-space-4);
    padding-block: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    color: inherit;
    text-decoration: none;
  }

  .orders-list__state {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .orders-list__link {
    margin-inline-start: auto;
    color: var(--k-accent-secondary);
    font-size: var(--k-text-sm);
  }

  @media (max-width: 32rem) {
    .orders-list__item a {
      flex-wrap: wrap;
      gap: var(--k-space-2);
    }
    .orders-list__link {
      inline-size: 100%;
      margin-inline-start: 0;
      margin-block-start: var(--k-space-1);
    }
  }
</style>
