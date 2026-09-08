<!--
  apps/artisan/src/routes/+page.svelte

  Home. Not a metrics dashboard: one primary action (add a product), what
  needs the artisan's attention right now, and whether their work has left
  the phone. Listings and lot offers themselves are Batch 8+ -- their strips
  render real empty states today because there is no listing or order model
  yet to show, not because the sections are unfinished chrome.

  Ordered sync-strip / primary action / needs-attention first, lot offers
  last, so the three things that matter most sit at the top of a small
  phone's viewport -- "three things maximum above the fold" is unverifiable
  without a real device in this environment, but the ordering is deliberate.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { liveQuery } from 'dexie';
  import { formatRelativeTime, locale } from '@kalakriti/i18n';
  import { OutboxView, network, db, getPref, type DraftRecord } from '@kalakriti/offline';
  import { Icon } from '@kalakriti/icons';
  import { syncEngine } from '$lib/sync';
  import { getArtisanId, getDraft as getRegistrationDraft } from '$lib/registration';
  import { cachedOrders, myLots, needsAction, type BulkOrder, type OrderLot } from '$lib/orders';
  import IncomeGrowthChart from '$lib/IncomeGrowthChart.svelte';
  import DigitalLiteracyTutorial from '$lib/DigitalLiteracyTutorial.svelte';
  import StallCardModal from '$lib/StallCardModal.svelte';
  import { launchDemoListing } from '$lib/demo-listing';

  const t = $derived(locale.t);

  const outbox = new OutboxView();
  $effect(() => outbox.start());

  let artisanId = $state<string | undefined>(undefined);
  let orders = $state<BulkOrder[]>([]);
  let showTutorial = $state(false);
  let showStallModal = $state(false);

  let artisanName = $state('Eshaan');
  let craftName = $state('Weaving & Handloom');
  let districtName = $state('Varanasi, Uttar Pradesh');
  let clusterName = $state('Varanasi Silk Weaver Facility Centre');
  let pehchanId = $state('UP-VNS-2024-0982');
  let avatarUrl = $state<string | undefined>(undefined);

  $effect(() => {
    void (async () => {
      artisanId = await getArtisanId();
      orders = await cachedOrders();

      const reg = await getRegistrationDraft();
      if (reg.name) artisanName = reg.name;
      if (reg.pehchanId) pehchanId = reg.pehchanId;
      if (reg.clusterName) clusterName = reg.clusterName;
      if (reg.districtFreeText) districtName = reg.districtFreeText;

      const storedAvatar = await getPref<string>('profile.avatar_url');
      if (storedAvatar) avatarUrl = storedAvatar;

      const seen = await getPref<boolean>('literacy.tutorial_completed');
      if (!seen) {
        showTutorial = true;
      }
    })();
  });

  async function handleStartDemo(): Promise<void> {
    const demoDraftId = await launchDemoListing();
    await goto(`/listing/new/studio?d=${demoDraftId}`);
  }

  interface HomeLotRow {
    order: BulkOrder;
    lot: OrderLot;
    kind: 'direct' | 'collective';
  }

  const actionableLots = $derived.by((): HomeLotRow[] => {
    if (!artisanId) return [];
    const lots = myLots(orders, artisanId);
    return lots
      .map((lot) => {
        const order = orders.find((o) => o.id === lot.bulk_order_id);
        if (!order) return undefined;
        return { order, lot, kind: (order.lots?.length ?? 1) > 1 ? 'collective' : 'direct' } as HomeLotRow;
      })
      .filter((r): r is HomeLotRow => r !== undefined && needsAction(r.lot));
  });

  function lotHref(row: HomeLotRow): string {
    if (row.lot.state === 'OFFERED') return `/orders/${row.order.id}/lots/${row.lot.id}/offer`;
    return `/orders/${row.order.id}/lots/${row.lot.id}`;
  }

  let drafts = $state<DraftRecord[]>([]);
  $effect(() => {
    const sub = liveQuery(() => db.drafts.orderBy('updatedAt').reverse().toArray()).subscribe(
      (rows) => (drafts = rows),
    );
    return () => sub.unsubscribe();
  });
  // A draft still shows here until the terms step's Publish is pressed --
  // termsAccepted is only ever set at that moment, so it doubles as "this
  // draft is spoken for" without a separate status field to keep in sync.
  // Excludes drafts with no captured media: the layout creates a draft row
  // the instant the wizard is opened, so tapping "Add a product" and
  // backing out before taking a single photo must not leave a permanent
  // "Untitled listing" row here -- only mediaIds ever going empty->1
  // signals real progress worth resuming.
  const inProgressDrafts = $derived(
    drafts.filter(
      (d) => (d.fields as { termsAccepted?: boolean }).termsAccepted !== true && d.mediaIds.length > 0,
    ),
  );

  const attentionEntries = $derived(
    outbox.entries.filter(
      (entry) =>
        entry.kind.startsWith('listing.') &&
        (entry.status === 'failed' || entry.status === 'needsAttention' || entry.status === 'blocked'),
    ),
  );

  const SYNC_ICON: Record<typeof syncEngine.state, 'offline' | 'sync' | 'warning' | 'clock' | 'success'> = {
    offline: 'offline',
    syncing: 'sync',
    attention: 'warning',
    pending: 'clock',
    synced: 'success',
  };
</script>

<svelte:head>
  <title>{t('app.name')}</title>
</svelte:head>

<h1 class="k-visually-hidden">{t('app.name')}</h1>

<section class="sync-strip" aria-label={t('sync.pending')}>
  <span class="sync-strip__status">
    <Icon name={SYNC_ICON[syncEngine.state]} class="sync-strip__icon" />
    {t(`sync.${syncEngine.state}`)}
    {#if syncEngine.counts.total > 0}
      <span class="sync-strip__count">({syncEngine.counts.total})</span>
    {/if}
  </span>
  <span class="sync-strip__lastSync">
    {syncEngine.lastSyncAt
      ? t('home.sync.lastSynced', { time: formatRelativeTime(syncEngine.lastSyncAt, locale.code) })
      : t('home.sync.never')}
  </span>
  <button
    type="button"
    class="sync-strip__action"
    onclick={() => syncEngine.syncNow()}
    disabled={!network.online || syncEngine.draining}
  >
    {t('home.sync.manual')}
  </button>
</section>

<!-- Digital Sahayak & Demo Walkthrough Hero Banner -->
<aside class="sahayak-banner">
  <div class="sahayak-banner__info">
    <div class="sahayak-banner__badge">
      <Icon name="verified-artisan" size="0.85rem" />
      <span>{t('literacy.sahayak.mode')}</span>
    </div>
    <h2 class="sahayak-banner__title">{t('literacy.tutorial.subtitle')}</h2>
  </div>
  <div class="sahayak-banner__actions">
    <button type="button" class="sahayak-btn" onclick={() => (showTutorial = true)}>
      📖 {t('literacy.tutorial.open')}
    </button>
    <button type="button" class="sahayak-btn sahayak-btn--primary" onclick={handleStartDemo}>
      ⚡ {t('literacy.demo.start')}
    </button>
  </div>
</aside>

<a class="add-product" href="/listing/new/capture">
  <Icon name="plus" class="add-product__icon" />
  {t('home.addProduct')}
</a>

<a class="my-listings-link" href="/listings">
  <Icon name="image" />
  {t('home.myListings')}
</a>

<div class="home-links-row">
  <a class="home-links-row__link" href="/earnings">
    <Icon name="income-statement" />
    {t('home.earnings')}
  </a>
  <button type="button" class="home-links-row__link home-links-row__btn" onclick={() => (showStallModal = true)}>
    <Icon name="verified-artisan" />
    <span>Stall Placard</span>
  </button>
  <a class="home-links-row__link" href="/notifications">
    <Icon name="bell" />
    {t('home.notifications')}
  </a>
</div>

<!-- Economic Growth & Income Uplift Section -->
<div class="home-growth-section">
  <IncomeGrowthChart compact />
</div>

<!-- Modals -->
<DigitalLiteracyTutorial
  open={showTutorial}
  onclose={() => (showTutorial = false)}
  onstartDemo={handleStartDemo}
/>

<StallCardModal
  open={showStallModal}
  onclose={() => (showStallModal = false)}
  {artisanName}
  {craftName}
  {districtName}
  {clusterName}
  {pehchanId}
  {avatarUrl}
/>

{#if inProgressDrafts.length > 0}
  <section class="drafts" aria-labelledby="drafts-heading">
    <h2 id="drafts-heading">{t('home.drafts.heading')}</h2>
    <ul role="list" class="drafts__list">
      {#each inProgressDrafts as draft (draft.id)}
        <li class="drafts__row">
          <a class="drafts__link" href="/listing/new/capture?d={draft.id}">
            <span class="drafts__title">
              {(draft.fields as { workingTitle?: string }).workingTitle || t('home.drafts.untitled')}
            </span>
            <span class="drafts__updated">
              {t('home.drafts.updated', { time: formatRelativeTime(draft.updatedAt, locale.code) })}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  </section>
{/if}

<section class="attention" aria-labelledby="attention-heading">
  <h2 id="attention-heading">{t('home.attention.heading')}</h2>
  {#if attentionEntries.length === 0}
    <p class="attention__empty">{t('offline.queue.empty')}</p>
  {:else}
    <ul role="list" class="attention__list">
      {#each attentionEntries as entry (entry.id)}
        <li class="attention__row" data-status={entry.status}>
          <span>{t(`outbox.kind.${entry.kind}`)}</span>
          <span class="attention__row-status">{t(`sync.${entry.status}`)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<section class="lots" aria-labelledby="lots-heading">
  <h2 id="lots-heading">{t('home.lots.heading')}</h2>
  {#if actionableLots.length === 0}
    <p class="lots__empty">{t('home.lots.empty')}</p>
  {:else}
    <ul role="list" class="lots__list">
      {#each actionableLots as row (row.lot.id)}
        <li class="lots__row">
          <a href={lotHref(row)} class="lots__link">
            <span>{row.kind === 'direct' ? t('orders.kind.direct') : t('orders.kind.collective')} ({t('orders.units', { count: String(row.lot.quantity ?? 1) })})</span>
            <span class="lots__row-status">{row.lot.state}</span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .sync-strip {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-4);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    font-size: var(--k-text-sm);
  }

  .sync-strip__status {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-text-primary);
    font-weight: var(--k-weight-medium);
  }

  .sync-strip__count {
    color: var(--k-text-secondary);
  }

  .sync-strip__lastSync {
    flex: 1;
    color: var(--k-text-secondary);
  }

  .sync-strip__action {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: none;
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .sync-strip__action:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .add-product {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    margin: var(--k-space-4);
    min-block-size: calc(var(--k-touch-min) * 1.4);
    border-radius: var(--k-radius-lg);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: var(--k-text-lg);
    font-weight: var(--k-weight-semibold);
    text-decoration: none;
  }

  .my-listings-link {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    margin: 0 var(--k-space-4) var(--k-space-4);
    min-block-size: var(--k-touch-min);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-sm);
  }

  .home-links-row {
    display: flex;
    gap: var(--k-space-3);
    margin: 0 var(--k-space-4) var(--k-space-4);
  }

  .home-links-row__link {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    min-block-size: var(--k-touch-min);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-sm);
    background: var(--k-surface-base);
  }

  .home-links-row__btn {
    cursor: pointer;
    font-family: inherit;
  }

  .sahayak-banner {
    margin: var(--k-space-4) var(--k-space-4) 0;
    padding: var(--k-space-3) var(--k-space-4);
    background: linear-gradient(135deg, rgba(217, 119, 6, 0.12), rgba(245, 158, 11, 0.04));
    border: 1px solid rgba(217, 119, 6, 0.35);
    border-radius: var(--k-radius-lg);
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--k-space-3);
    flex-wrap: wrap;
  }

  .sahayak-banner__info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .sahayak-banner__badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.06em;
    color: #b45309;
  }

  .sahayak-banner__title {
    font-size: var(--k-text-xs);
    color: var(--k-text-primary);
    font-weight: var(--k-weight-medium);
    margin: 0;
  }

  .sahayak-banner__actions {
    display: flex;
    gap: var(--k-space-2);
    flex-wrap: wrap;
  }

  .sahayak-btn {
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-pill);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    cursor: pointer;
    white-space: nowrap;
  }

  .sahayak-btn--primary {
    background: #b45309;
    color: #ffffff;
    border-color: #b45309;
  }

  .home-growth-section {
    margin: 0 var(--k-space-4) var(--k-space-4);
  }

  .drafts,
  .attention,
  .lots {
    margin: var(--k-space-4);
    padding-block-start: var(--k-space-4);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .drafts h2,
  .attention h2,
  .lots h2 {
    font-size: var(--k-text-md);
    margin-block-end: var(--k-space-2);
  }

  .drafts__list {
    display: flex;
    flex-direction: column;
  }

  .drafts__row {
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .drafts__link {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    padding-block: var(--k-space-2);
    color: var(--k-text-primary);
    text-decoration: none;
  }

  .drafts__updated {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .attention__empty,
  .lots__empty {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .attention__list {
    display: flex;
    flex-direction: column;
  }

  .attention__row {
    display: flex;
    justify-content: space-between;
    padding-block: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .attention__row-status {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }

  .lots__list {
    display: flex;
    flex-direction: column;
  }

  .lots__row {
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .lots__link {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-block: var(--k-space-2);
    color: var(--k-text-primary);
    text-decoration: none;
  }

  .lots__row-status {
    color: var(--k-accent-warning);
    font-size: var(--k-text-sm);
  }
</style>
