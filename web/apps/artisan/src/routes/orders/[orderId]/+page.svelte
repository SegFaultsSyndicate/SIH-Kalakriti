<!--
  apps/artisan/src/routes/orders/[orderId]/+page.svelte

  One bulk order: its own lots (this artisan's, at least) plus the ORDER
  TIMELINE (OrderTimeline, $lib/order-timeline.ts) -- the deliverable 4 SSE
  narrative, rendered here rather than duplicated per lot. Order-level
  notifications (PAYMENT_SETTLED, SHIPMENT_DELIVERED) deep-link here;
  lot-level ones deep-link straight to the lot's own page.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { Money } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { Icon } from '@kalakriti/icons';
  import StateBadge from '$lib/StateBadge.svelte';
  import OrderTimeline from '$lib/OrderTimeline.svelte';
  import { fetchOrder, cachedOrder, network, type BulkOrder } from '$lib/orders';
  import { getArtisanId } from '$lib/registration';

  const t = $derived(locale.t);
  const orderId = $derived(page.params.orderId ?? '');

  let loading = $state(true);
  let order = $state<BulkOrder | undefined>(undefined);
  let artisanId = $state<string | undefined>(undefined);

  $effect(() => {
    void (async () => {
      loading = true;
      artisanId = await getArtisanId();
      order = (await cachedOrder(orderId)) ?? undefined;
      loading = false;
      if (network.online) {
        try {
          order = await fetchOrder(orderId);
        } catch {
          /* keep the cached copy. */
        }
      }
    })();
  });

  const myLots = $derived((order?.lots ?? []).filter((l) => l.artisan_id === artisanId));

  function lotHref(lotId: string, state?: string): string {
    if (state === 'OFFERED') return `/orders/${orderId}/lots/${lotId}/offer`;
    return `/orders/${orderId}/lots/${lotId}`;
  }
</script>

<svelte:head>
  <title>{t('orderDetail.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="order-detail-page">
  <h1>{t('orderDetail.heading')}</h1>

  {#if loading && !order}
    <Skeleton shape="card" height="8rem" />
  {:else if !order}
    <p role="alert">{t('lotOffer.notFound')}</p>
  {:else}
    {#if myLots.length > 0}
      <section class="order-detail-page__lots">
        <h2>{t('orderDetail.yourLots')}</h2>
        <ul role="list">
          {#each myLots as lot (lot.id)}
            <li>
              <a href={lotHref(lot.id ?? '', lot.state)} class="order-detail-page__lot-link">
                <Card variant="hairline" element="div" class="order-detail-page__lot">
                  <StateBadge group={lot.state === 'OFFERED' || lot.state === 'QC_FAILED' ? 'needsAttention' : 'pending'} />
                  {#if lot.lot_value?.amount_paise != null}
                    <Money paise={lot.lot_value.amount_paise} />
                  {/if}
                  <Icon name="chevron-right" />
                </Card>
              </a>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <section class="order-detail-page__timeline">
      <h2>{t('timeline.heading')}</h2>
      <OrderTimeline {orderId} />
    </section>
  {/if}
</main>

<style>
  .order-detail-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .order-detail-page__lots ul {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .order-detail-page__lot-link {
    color: inherit;
    text-decoration: none;
  }

  :global(.order-detail-page__lot) {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
  }

  .order-detail-page__timeline {
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }
</style>
