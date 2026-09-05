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
  import { liveQuery } from 'dexie';
  import { formatRelativeTime, locale } from '@kalakriti/i18n';
  import { OutboxView, network, db, type DraftRecord } from '@kalakriti/offline';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { syncEngine } from '$lib/sync';

  const t = $derived(locale.t);

  const outbox = new OutboxView();
  $effect(() => outbox.start());

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
  <a class="home-links-row__link" href="/notifications">
    <Icon name="bell" />
    {t('home.notifications')}
  </a>
</div>

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
  <p class="lots__empty">{t('home.lots.empty')}</p>
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
</style>
