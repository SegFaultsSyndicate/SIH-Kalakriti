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
  import { formatDate, locale } from '@kalakriti/i18n';
  import { Money } from '@kalakriti/ui';
  import { Card, SkeletonRow } from '@kalakriti/patterns';
  import { Icon } from '@kalakriti/icons';
  import StateBadge from '$lib/StateBadge.svelte';
  import OrderTimeline from '$lib/OrderTimeline.svelte';
  import { fetchOrder, cachedOrder, network, type BulkOrder } from '$lib/orders';
  import { getArtisanId } from '$lib/registration';
  import { getSihMockOrderDetails, type SihMockOrderDetails } from '$lib/sih-my-works';

  const t = $derived(locale.t);
  const orderId = $derived(page.params.orderId ?? '');

  let loading = $state(true);
  let order = $state<BulkOrder | undefined>(undefined);
  let artisanId = $state<string | undefined>(undefined);
  let mockOrder = $state<SihMockOrderDetails | undefined>(undefined);

  $effect(() => {
    void (async () => {
      loading = true;
      artisanId = await getArtisanId();
      mockOrder = getSihMockOrderDetails(orderId);
      if (mockOrder) {
        const lotState = mockOrder.state === 'COMPLETED'
          ? 'COMPLETED'
          : mockOrder.state === 'IN_PROGRESS'
            ? 'IN_PRODUCTION'
            : 'OFFERED';
        order = {
          id: orderId,
          buyer_id: 'sih-buyer',
          listing_id: mockOrder.listingId,
          quantity: mockOrder.quantity,
          unit_price: { amount_paise: mockOrder.unitPricePaise, currency_code: 'INR' },
          total_value: { amount_paise: mockOrder.totalPaise, currency_code: 'INR' },
          state: mockOrder.state === 'COMPLETED' ? 'COMPLETED' : 'IN_PRODUCTION',
          allocated_quantity: mockOrder.quantity,
          lots: [
            {
              id: `lot-${orderId}`,
              artisan_id: artisanId || 'sih-artisan-eshaan',
              bulk_order_id: orderId,
              state: lotState,
              quantity: mockOrder.quantity,
              lot_value: { amount_paise: mockOrder.totalPaise, currency_code: 'INR' },
              offered_at: mockOrder.placedAt,
            },
          ],
        };
        loading = false;
        return;
      }

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

<div class="page-container">
  <header class="editorial-header">
    <p class="kicker">order details</p>
    <h1 class="heading">{t('orderDetail.heading')}</h1>
  </header>

  {#if loading && !order}
    <div class="skeleton-layout">
      {#each Array(2) as _, i (i)}
        <SkeletonRow lines={[{ width: '30%', height: '0.9rem' }, { width: '25%', height: '0.85rem' }]} trailing trailingWidth="1.5rem" />
      {/each}
    </div>
  {:else if !order}
    <p role="alert" class="alert-text">{t('lotOffer.notFound')}</p>
  {:else}
    {#if mockOrder?.shipping}
      <div class="demo-order">
        <Card variant="hairline" element="section" class="demo-order__product">
          {#if mockOrder.imageUrl}
            <img class="demo-order__image" src={mockOrder.imageUrl} alt="" />
          {/if}
          <div class="demo-order__product-copy">
            <span class="demo-order__eyebrow">Order {mockOrder.id}</span>
            <h2>{mockOrder.listingTitle}</h2>
            <p>Banarasi Brocade Weaving · Varanasi</p>
            <div class="demo-order__product-meta">
              <span>{mockOrder.quantity} piece</span>
              <span>Placed {formatDate(mockOrder.placedAt, locale.code, { dateStyle: 'medium' })}</span>
            </div>
          </div>
          <div class="demo-order__total">
            <span>Order total</span>
            <Money paise={mockOrder.totalPaise} />
          </div>
        </Card>

        <div class="demo-order__grid">
          <Card variant="hairline" element="section" class="demo-order__panel">
            <div class="demo-order__panel-heading">
              <span class="demo-order__icon"><Icon name="user" /></span>
              <div>
                <h2>Customer</h2>
                <p>Recipient and delivery address</p>
              </div>
            </div>
            <strong class="demo-order__customer">{mockOrder.buyerName}</strong>
            <a class="demo-order__phone" href={`tel:${mockOrder.shipping.phone.replace(/\s/g, '')}`}>
              <Icon name="phone" />
              {mockOrder.shipping.phone}
            </a>
            <address>
              {#each mockOrder.shipping.addressLines as line (line)}
                <span>{line}</span>
              {/each}
            </address>
          </Card>

          <Card variant="hairline" element="section" class="demo-order__panel">
            <div class="demo-order__panel-heading">
              <span class="demo-order__icon"><Icon name="package" /></span>
              <div>
                <h2>Delivery</h2>
                <p>Shipment tracking and progress</p>
              </div>
              <span class="demo-order__delivered"><Icon name="check" /> Delivered</span>
            </div>

            <dl class="demo-order__tracking">
              <div><dt>Carrier</dt><dd>{mockOrder.shipping.carrier}</dd></div>
              <div><dt>Tracking number</dt><dd>{mockOrder.shipping.trackingCode}</dd></div>
              <div><dt>Delivered</dt><dd>{formatDate(mockOrder.shipping.deliveredAt, locale.code, { dateStyle: 'medium', timeStyle: 'short' })}</dd></div>
            </dl>

            <ol class="demo-order__timeline">
              {#each mockOrder.shipping.milestones as milestone, index (milestone.label)}
                <li class:demo-order__milestone--last={index === mockOrder.shipping.milestones.length - 1}>
                  <span class="demo-order__milestone-mark"><Icon name="check" /></span>
                  <span class="demo-order__milestone-copy">
                    <strong>{milestone.label}</strong>
                    <time datetime={milestone.at}>{formatDate(milestone.at, locale.code, { dateStyle: 'medium', timeStyle: 'short' })}</time>
                  </span>
                </li>
              {/each}
            </ol>
          </Card>
        </div>
      </div>
    {:else}
    <div class="grid-layout">
      {#if myLots.length > 0}
        <section class="editorial-section">
          <div class="section-header">
            <h2 class="section-title">{t('orderDetail.yourLots')}</h2>
          </div>

          <ul role="list" class="lot-list">
            {#each myLots as lot (lot.id)}
              <li>
                <a href={lotHref(lot.id ?? '', lot.state)} class="lot-link">
                  <article class="lot-card">
                    <div class="lot-card__content">
                      <StateBadge group={lot.state === 'OFFERED' || lot.state === 'QC_FAILED' ? 'needsAttention' : 'pending'} />
                      {#if lot.lot_value?.amount_paise != null}
                        <div class="lot-card__price">
                          <Money paise={lot.lot_value.amount_paise} />
                        </div>
                      {/if}
                    </div>
                    <Icon name="chevron-right" class="lot-card__icon" />
                  </article>
                </a>
              </li>
            {/each}
          </ul>
        </section>
      {/if}

      <section class="editorial-section">
        <div class="section-header">
          <h2 class="section-title">{t('timeline.heading')}</h2>
        </div>
        <OrderTimeline {orderId} />
      </section>
    </div>
    {/if}
  {/if}
</div>

<style>
  .page-container {
    background-color: var(--k-color-paper, #fdfbf7);
    background-image: url('data:image/svg+xml;utf8,<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg"><filter id="noiseFilter"><feTurbulence type="fractalNoise" baseFrequency="0.65" numOctaves="3" stitchTiles="stitch"/></filter><rect width="100%" height="100%" filter="url(%23noiseFilter)" opacity="0.03"/></svg>');
    min-height: 100vh;
    padding: var(--k-space-4) var(--k-space-3);
    padding-bottom: calc(var(--k-space-6) + env(safe-area-inset-bottom));
  }

  .editorial-header {
    margin-bottom: var(--k-space-6);
    border-bottom: var(--k-hairline) solid var(--k-border-hairline, #e0dcd3);
    padding-bottom: var(--k-space-3);
  }

  .kicker {
    font-size: var(--k-text-xs);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-text-secondary, #5c5c5c);
    margin: 0 0 var(--k-space-1) 0;
  }

  .heading {
    font-family: var(--k-font-serif, 'Georgia', serif);
    font-size: var(--k-text-3xl, 2rem);
    font-weight: 400;
    line-height: 1.1;
    color: var(--k-text-primary, #1a1a1a);
    margin: 0;
  }

  .grid-layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-6);
  }

  .demo-order {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    max-inline-size: 68rem;
    margin-inline: auto;
    container-type: inline-size;
  }

  :global(.demo-order__product) {
    display: grid;
    grid-template-columns: 7rem minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
  }

  .demo-order__image {
    inline-size: 7rem;
    aspect-ratio: 1;
    border-radius: var(--k-radius-md);
    object-fit: cover;
  }

  .demo-order__product-copy {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    min-inline-size: 0;
  }

  .demo-order__eyebrow {
    color: var(--k-accent-primary-text);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    text-transform: uppercase;
  }

  .demo-order__product-copy h2,
  .demo-order__panel h2 {
    margin: 0;
    color: var(--k-text-primary);
    font-size: var(--k-text-lg);
  }

  .demo-order__product-copy p,
  .demo-order__panel-heading p {
    margin: 0;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .demo-order__product-meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-2);
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  .demo-order__total {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--k-space-1);
    white-space: nowrap;
  }

  .demo-order__total > span {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  :global(.demo-order__total .k-money) {
    color: var(--k-text-primary);
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-bold);
  }

  .demo-order__grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 1fr));
    gap: var(--k-space-4);
  }

  :global(.demo-order__panel) {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
  }

  .demo-order__panel-heading {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .demo-order__icon {
    display: grid;
    place-items: center;
    inline-size: 2.5rem;
    block-size: 2.5rem;
    flex-shrink: 0;
    border-radius: var(--k-radius-md);
    background: color-mix(in srgb, var(--k-accent-primary-bg) 12%, transparent);
    color: var(--k-accent-primary-text);
  }

  .demo-order__panel-heading > div {
    flex: 1;
    min-inline-size: 0;
  }

  .demo-order__delivered {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-2);
    border-radius: var(--k-radius-pill);
    background: color-mix(in srgb, var(--k-neem-300) 25%, transparent);
    color: var(--k-accent-success);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    white-space: nowrap;
  }

  .demo-order__customer {
    font-size: var(--k-text-md);
  }

  .demo-order__phone {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-accent-primary-text);
    text-decoration: none;
  }

  .demo-order__phone:hover {
    text-decoration: underline;
  }

  .demo-order__panel address {
    display: flex;
    flex-direction: column;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    font-style: normal;
    line-height: var(--k-leading-relaxed);
  }

  .demo-order__tracking {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--k-space-3);
    margin: 0;
    padding-block-end: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .demo-order__tracking div:last-child {
    grid-column: 1 / -1;
  }

  .demo-order__tracking dt {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  .demo-order__tracking dd {
    margin: var(--k-space-1) 0 0;
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    overflow-wrap: anywhere;
  }

  .demo-order__timeline {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .demo-order__timeline li {
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
  }

  .demo-order__timeline li:not(:last-child)::after {
    position: absolute;
    inset-inline-start: 0.45rem;
    inset-block-start: 1rem;
    inline-size: 1px;
    block-size: calc(100% + var(--k-space-3) - 0.25rem);
    background: var(--k-border-hairline);
    content: '';
  }

  .demo-order__milestone-mark {
    display: grid;
    place-items: center;
    inline-size: 1rem;
    block-size: 1rem;
    flex-shrink: 0;
    border-radius: var(--k-radius-pill);
    background: var(--k-accent-success-bg);
    color: var(--k-text-on-accent);
  }

  :global(.demo-order__milestone-mark svg) {
    inline-size: 0.7rem;
    block-size: 0.7rem;
  }

  .demo-order__milestone-copy {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .demo-order__milestone-copy strong {
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .demo-order__milestone-copy time {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  @container (max-width: 42rem) {
    :global(.demo-order__product) {
      grid-template-columns: 5rem minmax(0, 1fr);
      gap: var(--k-space-3);
    }

    .demo-order__image {
      inline-size: 5rem;
    }

    .demo-order__total {
      grid-column: 2;
      align-items: flex-start;
    }

    .demo-order__grid {
      grid-template-columns: 1fr;
    }
  }

  @media (min-width: 48rem) {
    .grid-layout {
      grid-template-columns: 1fr 1fr;
      align-items: start;
    }
  }

  .editorial-section {
    display: flex;
    flex-direction: column;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: var(--k-hairline) solid var(--k-border-hairline, #e0dcd3);
    margin-bottom: var(--k-space-3);
    padding-bottom: var(--k-space-1);
  }

  .section-title {
    font-family: var(--k-font-serif, 'Georgia', serif);
    font-size: var(--k-text-xl, 1.25rem);
    font-weight: 400;
    color: var(--k-text-primary, #1a1a1a);
    margin: 0;
  }

  .lot-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .lot-link {
    text-decoration: none;
    color: inherit;
    display: block;
  }

  .lot-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--k-space-3) 0;
    border-bottom: var(--k-hairline) solid var(--k-border-hairline, #e0dcd3);
    transition: background-color 0.2s ease;
  }

  .lot-card:hover, .lot-link:focus-visible .lot-card {
    background-color: rgba(0, 0, 0, 0.02);
  }

  .lot-card__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .lot-card__price {
    font-size: var(--k-text-lg, 1.125rem);
    font-family: var(--k-font-sans, sans-serif);
    font-weight: 500;
    color: var(--k-text-primary, #1a1a1a);
  }

  .lot-card__icon {
    color: var(--k-text-secondary, #5c5c5c);
  }

  .skeleton-layout {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .alert-text {
    color: var(--k-text-secondary, #5c5c5c);
    font-family: var(--k-font-sans, sans-serif);
  }
</style>
