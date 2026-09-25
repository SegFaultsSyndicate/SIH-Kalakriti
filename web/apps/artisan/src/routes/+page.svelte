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
  import { formatRelativeTime, formatDate, locale } from '@kalakriti/i18n';
  import { OutboxView, network, db, getPref, type DraftRecord } from '@kalakriti/offline';
  import { Icon } from '@kalakriti/icons';
  import { syncEngine } from '$lib/sync';
  import { getArtisanId, getDraft as getRegistrationDraft, getCachedDraftSync } from '$lib/registration';
  import { cachedOrders, getCachedOrdersSync, myLots, needsAction, type BulkOrder, type OrderLot } from '$lib/orders';
  import IncomeGrowthChart from '$lib/IncomeGrowthChart.svelte';
  import CoverageStrip from '$lib/CoverageStrip.svelte';
  import DigitalLiteracyTutorial from '$lib/DigitalLiteracyTutorial.svelte';
  import StallCardModal from '$lib/StallCardModal.svelte';
  import { launchDemoListing, cleanupLegacyDraftsAndSeedPaithani } from '$lib/demo-listing';
  import { ensureDemoState } from '$lib/sih-demo-store';

  // Seed the SIH demo store on first load so both portals share the same
  // Eshaan catalog from the very first frame of the recording.
  if (typeof localStorage !== 'undefined') ensureDemoState();

  const t = $derived(locale.t);

  const outbox = new OutboxView();
  $effect(() => outbox.start());

  const initialDraft = getCachedDraftSync();
  const initialOrders = getCachedOrdersSync();

  function getInitialAvatar(): string | undefined {
    try {
      if (typeof localStorage !== 'undefined') {
        return localStorage.getItem('kalakriti.artisan.avatar') || undefined;
      }
    } catch {}
    return undefined;
  }

  let artisanId = $state<string | undefined>(undefined);
  let orders = $state<BulkOrder[]>(initialOrders ?? []);
  let showTutorial = $state(false);
  let showStallModal = $state(false);

  let artisanName = $state(initialDraft.name || 'Eshaan');
  let craftName = $state(initialDraft.craftName || 'Weaving & Handloom');
  let districtName = $state(initialDraft.districtFreeText || 'Varanasi, Uttar Pradesh');
  let clusterName = $state(initialDraft.clusterName || 'Varanasi Silk Weaver Facility Centre');
  let pehchanId = $state(initialDraft.pehchanId || 'UP-VNS-2024-0982');
  let avatarUrl = $state<string | undefined>(getInitialAvatar());

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
      if (storedAvatar) {
        avatarUrl = storedAvatar;
        try {
          if (typeof localStorage !== 'undefined') {
            localStorage.setItem('kalakriti.artisan.avatar', storedAvatar);
          }
        } catch {}
      }

      const seen = await getPref<boolean>('literacy.tutorial_completed');
      if (!seen) {
        showTutorial = true;
      }

      await cleanupLegacyDraftsAndSeedPaithani();
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

  // First photo of each in-progress draft, so the resume card shows the piece.
  let draftThumbs = $state<Record<string, string>>({});
  $effect(() => {
    const urls: Record<string, string> = {};
    let cancelled = false;
    void (async () => {
      for (const d of inProgressDrafts) {
        let url: string | undefined;
        if (d.mediaIds[0]) {
          const media = await db.media.get(d.mediaIds[0]);
          if (media?.blob && media.blob.size > 0) {
            url = URL.createObjectURL(media.blob);
          }
        }
        urls[d.id] = url || '/craft-images/weaving_and_looms/paithani-saree-blue-green.jpeg';
      }
      if (cancelled) for (const url of Object.values(urls)) {
        if (url.startsWith('blob:')) URL.revokeObjectURL(url);
      }
      else draftThumbs = urls;
    })();
    return () => {
      cancelled = true;
      for (const url of Object.values(urls)) {
        if (url.startsWith('blob:')) URL.revokeObjectURL(url);
      }
    };
  });

  const attentionEntries = $derived(
    outbox.entries.filter(
      (entry) =>
        entry.kind.startsWith('listing.') &&
        (entry.status === 'failed' ||
          entry.status === 'needsAttention' ||
          entry.status === 'blocked' ||
          (entry.status === 'syncing' && (entry.attempts > 0 || entry.lastError !== undefined))),
    ),
  );

  const SYNC_ICON: Record<typeof syncEngine.state, 'offline' | 'sync' | 'warning' | 'clock' | 'success'> = {
    offline: 'offline',
    syncing: 'sync',
    attention: 'warning',
    pending: 'clock',
    synced: 'success',
  };

  let manualSyncing = $state(false);
  const isSyncing = $derived(syncEngine.draining || manualSyncing);

  const effectiveSyncState = $derived.by((): typeof syncEngine.state => {
    if (!network.online) return 'offline';
    if (isSyncing) return 'syncing';
    return syncEngine.state;
  });

  async function handleSyncNow(): Promise<void> {
    if (!network.online || isSyncing) return;
    manualSyncing = true;
    try {
      // Guarantee a smooth minimum sync duration (600ms) so micro-second
      // completions don't produce a jarring visual flash glitch.
      await Promise.all([
        syncEngine.syncNow(true),
        new Promise((resolve) => setTimeout(resolve, 600)),
      ]);
    } catch (err) {
      console.warn('Sync failed:', err);
    } finally {
      manualSyncing = false;
    }
  }
</script>

<svelte:head>
  <title>{t('app.name')}</title>
</svelte:head>

<section class="sync-strip sync-strip--{effectiveSyncState}" aria-label={t('sync.pending')}>
  <div class="sync-strip__info">
    <div class="sync-strip__status">
      <span class="sync-strip__icon-wrap">
        <Icon name={SYNC_ICON[effectiveSyncState]} class="sync-strip__icon" size="1rem" />
      </span>
      <span class="sync-strip__label">
        {effectiveSyncState === 'attention' ? t('sync.needsAttention') : t(`sync.${effectiveSyncState}`)}
      </span>
      {#if syncEngine.counts.total > 0}
        <span class="sync-strip__count">({syncEngine.counts.total})</span>
      {/if}
    </div>
    <span class="sync-strip__last-sync">
      {syncEngine.lastSyncAt
        ? t('home.sync.lastSynced', { time: formatDate(syncEngine.lastSyncAt, locale.code, { timeStyle: 'short' }) })
        : t('home.sync.never')}
    </span>
  </div>
  <button
    type="button"
    class="sync-strip__action"
    class:is-spinning={isSyncing}
    onclick={handleSyncNow}
    disabled={!network.online || isSyncing}
    aria-label={t('home.sync.manual')}
  >
    <span class="sync-strip__btn-icon">
      <Icon name="sync" size="0.875rem" />
    </span>
    <span class="sync-strip__btn-text">{t('home.sync.manual')}</span>
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
      <Icon name="help" size="1.1rem" /> {t('literacy.tutorial.open')}
    </button>
    <button type="button" class="sahayak-btn sahayak-btn--primary" onclick={handleStartDemo}>
      <Icon name="play" size="1.1rem" /> {t('literacy.demo.start')}
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
    <span>{t('home.stallPlacard')}</span>
  </button>
  <a class="home-links-row__link" href="/notifications">
    <Icon name="bell" />
    <span>{t('home.notifications')}</span>
  </a>
  <a class="home-links-row__link" href="/trends">
    <Icon name="link" />
    <span>{t('nav.trends')}</span>
  </a>
  <a class="home-links-row__link" href="/finance">
    <Icon name="dollar-sign" />
    <span>{t('finance.title')}</span>
  </a>
  <a class="home-links-row__link" href="/learn">
    <Icon name="badge" />
    <span>{t('learn.title')}</span>
  </a>
</div>

<!-- B2B Enterprise & Boutique Partners Strip -->
<div class="b2b-opportunity-card">
  <div class="b2b-header">
    <Icon name="package" />
    <span class="b2b-kicker">{t('home.sourcing.kicker')}</span>
  </div>
  <p class="b2b-desc">
    {t('home.sourcing.desc')}
  </p>
  <div class="b2b-actions">
    <a href="/orders" class="b2b-btn">{t('home.sourcing.viewLots')}</a>
    <a href="/trends" class="b2b-btn secondary">{t('home.sourcing.viewTrends')}</a>
  </div>
</div>

<!-- Economic Growth & Income Uplift Section -->
<div class="home-growth-section">
  <CoverageStrip />
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
            {#if draftThumbs[draft.id]}
              <img class="drafts__thumb" src={draftThumbs[draft.id]} alt="" />
            {:else}
              <span class="drafts__thumb drafts__thumb--empty" aria-hidden="true"><Icon name="image" /></span>
            {/if}
            <span class="drafts__text">
              <span class="drafts__title">
                {(draft.fields as { workingTitle?: string }).workingTitle || t('home.drafts.untitled')}
              </span>
              <span class="drafts__updated">
                {t('home.drafts.updated', { time: formatRelativeTime(draft.updatedAt, locale.code) })}
              </span>
            </span>
            <span class="drafts__cta" aria-hidden="true">
              {t('listings.action.continue')}
              <Icon name="chevron-right" size="1rem" />
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
    justify-content: space-between;
    gap: var(--k-space-2);
    padding: 0.375rem var(--k-space-4);
    min-block-size: 2.75rem;
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface-base);
    box-sizing: border-box;
    transition: background-color 0.2s ease, border-color 0.2s ease;
  }

  .sync-strip--attention {
    background-color: #fef7ed;
    border-block-end-color: #fed7aa;
  }

  .sync-strip--syncing {
    background-color: #f0f9ff;
    border-block-end-color: #bae6fd;
  }

  .sync-strip--offline {
    background-color: #f8fafc;
  }

  .sync-strip__info {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 1px;
    min-inline-size: 0;
    flex: 1;
  }

  .sync-strip__status {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    min-inline-size: 0;
  }

  .sync-strip__icon-wrap {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    inline-size: 1rem;
    block-size: 1rem;
    color: var(--k-text-secondary);
  }

  .sync-strip--attention .sync-strip__icon-wrap {
    color: #c2410c;
  }

  .sync-strip--synced .sync-strip__icon-wrap {
    color: #166534;
  }

  .sync-strip--syncing .sync-strip__icon-wrap {
    color: #0284c7;
    animation: sync-spin 0.8s linear infinite;
  }

  .sync-strip--pending .sync-strip__icon-wrap {
    color: #b45309;
  }

  .sync-strip--offline .sync-strip__icon-wrap {
    color: #64748b;
  }

  .sync-strip__label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.25;
  }

  .sync-strip--attention .sync-strip__label {
    color: #9a3412;
  }

  .sync-strip__count {
    font-size: 0.6875rem;
    font-weight: var(--k-weight-bold);
    padding: 0.05rem 0.35rem;
    border-radius: var(--k-radius-full);
    background-color: var(--k-surface-sunken, #ece8df);
    color: var(--k-text-secondary);
    line-height: 1.2;
    flex-shrink: 0;
  }

  .sync-strip--attention .sync-strip__count {
    background-color: #ffedd5;
    color: #c2410c;
  }

  .sync-strip__last-sync {
    font-size: 0.6875rem;
    color: var(--k-text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
    padding-inline-start: calc(1rem + 0.375rem);
  }

  .sync-strip__action {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    min-block-size: 1.875rem;
    padding: 0.25rem 0.625rem;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-full);
    background-color: var(--k-surface-elevated, #ffffff);
    color: var(--k-text-primary);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
    flex-shrink: 0;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    transition: background-color 0.15s ease, border-color 0.15s ease, opacity 0.2s ease, transform 0.1s ease;
  }

  /* Accessible hit target (44px) on touch screens without expanding visual footprint */
  .sync-strip__action::after {
    content: '';
    position: absolute;
    inset: -7px -4px;
  }

  .sync-strip__action:hover:not(:disabled) {
    background-color: var(--k-surface-sunken);
    border-color: var(--k-ink-500, #888);
  }

  .sync-strip__action:active:not(:disabled) {
    transform: scale(0.96);
  }

  .sync-strip__action:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  .sync-strip__btn-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .sync-strip__action.is-spinning .sync-strip__btn-icon {
    animation: sync-spin 0.8s linear infinite;
  }

  @keyframes sync-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  @media (min-width: 480px) {
    .sync-strip__info {
      flex-direction: row;
      align-items: center;
      gap: var(--k-space-3);
    }
    .sync-strip__last-sync {
      padding-inline-start: 0;
    }
    .sync-strip__last-sync::before {
      content: '·';
      margin-inline-end: var(--k-space-2);
      font-weight: bold;
    }
  }

  .add-product {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    margin-block: var(--k-space-4);
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
    margin: 0 0 var(--k-space-4);
    min-block-size: var(--k-touch-min);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-sm);
  }

  .home-links-row {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--k-space-3);
    margin: 0 0 var(--k-space-4);
  }

  .home-links-row__link {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-1);
    min-block-size: calc(var(--k-touch-min) * 1.5);
    padding: var(--k-space-2) var(--k-space-1);
    text-align: center;
    line-height: 1.2;
    overflow-wrap: break-word;
    hyphens: auto;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-sm);
    background: var(--k-surface-base);
  }

  /* Phones: four columns squeeze "Notifications" into a mid-word break.
     2x2 with icon beside label keeps every word whole. */
  @media (max-width: 30rem) {
    .home-links-row {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--k-space-2);
    }

    .home-links-row__link {
      flex-direction: row;
      justify-content: flex-start;
      gap: var(--k-space-2);
      min-block-size: var(--k-touch-min);
      padding: var(--k-space-2) var(--k-space-3);
      text-align: start;
    }
  }

  .home-links-row__btn {
    cursor: pointer;
    font-family: inherit;
  }

  .sahayak-banner {
    margin: var(--k-space-4) 0 0;
    padding: var(--k-space-3) var(--k-space-4);
    background: var(--k-surface-raised);
    border: 1px solid var(--k-border-hairline);
    border-inline-start: 3px solid var(--k-accent-warning-bg);
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
    color: var(--k-accent-primary-text);
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
    /* Icon + label on one row. The global reset gives every <svg> display:block
       (packages/tokens/src/reset.css) so a plain <button> with no flex context
       stacks its icon above its text instead of beside it -- this is the fix. */
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
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
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-border-accent);
  }

  .home-growth-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    margin: 0 0 var(--k-space-4);
  }

  .drafts,
  .attention,
  .lots {
    margin-block: var(--k-space-4);
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
    gap: var(--k-space-2);
  }

  /* Each draft is one big tappable card with an explicit "Continue" CTA,
     so it's obvious the row resumes the listing. */
  .drafts__link {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    min-block-size: calc(var(--k-touch-min) * 1.5);
    padding: var(--k-space-2) var(--k-space-3) var(--k-space-2) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: 0.875rem;
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    text-decoration: none;
    box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  .drafts__link:hover,
  .drafts__link:focus-visible {
    border-color: var(--k-accent-primary-bg);
    box-shadow: 0 4px 14px rgb(0 0 0 / 0.08);
  }

  .drafts__thumb {
    flex-shrink: 0;
    inline-size: 3.5rem;
    block-size: 3.5rem;
    border-radius: 0.625rem;
    object-fit: cover;
  }

  .drafts__thumb--empty {
    display: grid;
    place-items: center;
    background: var(--k-surface-sunken);
    color: var(--k-text-secondary);
  }

  .drafts__text {
    flex: 1;
    min-inline-size: 0;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .drafts__title {
    font-weight: var(--k-weight-semibold);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .drafts__updated {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .drafts__cta {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 0.15rem;
    padding: 0.4rem 0.6rem 0.4rem 0.8rem;
    border-radius: var(--k-radius-pill);
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
  }

  :global([dir='rtl']) .drafts__cta :global(svg) {
    transform: scaleX(-1);
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

  .b2b-opportunity-card {
    border: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface-raised);
    padding: var(--k-space-3);
    margin-block: var(--k-space-3);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .b2b-header {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-terracotta);
  }

  .b2b-kicker {
    font-size: var(--k-text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .b2b-desc {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    line-height: 1.4;
    margin: 0;
  }

  .b2b-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-1);
  }

  .b2b-btn {
    display: inline-flex;
    align-items: center;
    padding: var(--k-space-2) var(--k-space-3);
    font-size: var(--k-text-xs);
    font-weight: 600;
    text-decoration: none;
    border: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface);
    color: var(--k-ink);
  }

  .b2b-btn:not(.secondary) {
    background-color: var(--k-terracotta);
    color: var(--k-khadi);
    border-color: transparent;
  }
</style>
