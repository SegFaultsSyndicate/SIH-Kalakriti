<!--
  apps/artisan/src/routes/listings/+page.svelte

  The artisan's catalog: every listing they own, grouped by state, with
  filter, voice search, and bulk actions. GET /listings?artisan_id= is real
  (see ml_wiring.md's batch 9 section) -- this screen is online-first with a
  cached last-fetch for an offline reopen (network.online gates the fetch
  and the bulk actions, not the page itself).
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, Checkbox, EmptyState, Money, VisuallyHidden, VoiceInput, showToast } from '@kalakriti/ui';
  import { Card, SkeletonRow } from '@kalakriti/patterns';
  import { searchVoice } from '@kalakriti/api';
  import StateBadge from '$lib/StateBadge.svelte';
  import {
    STATE_GROUPS,
    STATE_GROUP_LABEL,
    fetchMyListings,
    cachedListings,
    getCachedListingsSync,
    groupFor,
    titleFor,
    localPrimaryImageUrl,
    canPauseOrResume,
    pauseSelling,
    resumeSelling,
    duplicateAsDraft,
    network,
    type Listing,
    type StateGroup,
  } from '$lib/listings';
  import { loadDemoState } from '$lib/sih-demo-store';
  import { getSihListingTitleKey, getSihMyWorks } from '$lib/sih-my-works';

  /** Tiny cross-tab listener wired to the same localStorage key. */
  function onDemoStateChange(cb: () => void): () => void {
    function handler(e: StorageEvent): void {
      if (e.key === 'kalakriti.sih.demo') cb();
    }
    if (typeof window !== 'undefined') window.addEventListener('storage', handler);
    return () => { if (typeof window !== 'undefined') window.removeEventListener('storage', handler); };
  }

  const t = $derived(locale.t);

  function displayTitle(listing: Listing): string {
    const titleKey = getSihListingTitleKey(listing.id);
    return titleKey ? t(titleKey) : titleFor(listing, locale.code);
  }

  const initialListings = getCachedListingsSync();
  let listings = $state<Listing[]>(initialListings ?? []);
  let loading = $state(initialListings === null);
  let stateFilter = $state<StateGroup | 'all'>('all');
  let searchQuery = $state('');
  let voiceSearching = $state(false);
  let selected = $state(new Set<string>());
  let thumbnails = $state<Record<string, string>>({});
  let pauseable = $state<Record<string, boolean>>({});

  /** Image URLs keyed by SIH listing id, for mock listings without IndexedDB blobs. */
  const sihImageMap = $derived.by((): Record<string, string> => {
    const state = loadDemoState();
    const map: Record<string, string> = {};
    for (const l of state.listings) {
      if (l.imageUrl) map[l.id] = l.imageUrl;
    }
    return map;
  });

  /** Merges SIH mock listings with real/cached listings, deduplicating by id. */
  function mergeSihListings(base: Listing[]): Listing[] {
    const mock = getSihMyWorks();
    const existingIds = new Set(base.map((l) => l.id));
    const novel = mock.filter((m) => m.id && !existingIds.has(m.id));
    return [...novel, ...base].filter((listing) => {
      const title = titleFor(listing, locale.code);
      const price = listing.price?.amount_paise;
      return !(listing.state === 'DRAFT' && !title && (price === 600000 || price === 500000));
    });
  }

  async function load(): Promise<void> {
    if (initialListings === null) loading = true;
    const cached = await cachedListings();
    listings = mergeSihListings(cached);
    if (network.online) {
      try {
        const real = await fetchMyListings();
        listings = mergeSihListings(real);
      } catch {
        // Backend has no real listing data in local/dev environments yet --
        // fall back to whatever's cached (usually empty) rather than
        // surfacing a raw "Request failed with status 500" to the artisan.
      }
    }
    loading = false;

    const urls: Record<string, string> = {};
    const canPause: Record<string, boolean> = {};
    for (const listing of listings) {
      if (!listing.id) continue;
      // SIH mock listings expose their imageUrl through translations as a data-attr workaround;
      // real listings use localPrimaryImageUrl from IndexedDB.
      const url = await localPrimaryImageUrl(listing.id);
      if (url) urls[listing.id] = url;
      canPause[listing.id] = await canPauseOrResume(listing);
    }
    thumbnails = urls;
    pauseable = canPause;
  }

  // Reactively reload when the artisan publishes the Paithani listing
  // (e.g. if they navigate back to this tab).
  $effect(() => onDemoStateChange(() => void load()));

  $effect(() => {
    void load();
    return () => {
      for (const url of Object.values(thumbnails)) URL.revokeObjectURL(url);
    };
  });

  const grouped = $derived.by(() => {
    const byGroup = new Map<StateGroup, Listing[]>(STATE_GROUPS.map((g) => [g, []]));
    const query = searchQuery.trim().toLowerCase();
    for (const listing of listings) {
      if (query && !displayTitle(listing).toLowerCase().includes(query)) continue;
      const group = groupFor(listing);
      if (stateFilter !== 'all' && stateFilter !== group) continue;
      byGroup.get(group)?.push(listing);
    }
    return byGroup;
  });

  const totalCount = $derived(listings.length);

  async function onVoiceSearch(blob: Blob): Promise<void> {
    voiceSearching = true;
    try {
      const { query } = await searchVoice(blob, locale.meta.tag);
      searchQuery = query ?? '';
    } catch {
      showToast({ variant: 'error', message: t('listings.search.voiceError') });
    } finally {
      voiceSearching = false;
    }
  }

  function toggleSelect(id: string): void {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }

  function selectedListings(): Listing[] {
    return listings.filter((l) => l.id && selected.has(l.id));
  }

  const anySelectedPauseable = $derived(
    selectedListings().some((l) => l.id && pauseable[l.id]),
  );

  async function bulkPause(): Promise<void> {
    for (const listing of selectedListings()) {
      try {
        await pauseSelling(listing);
      } catch {
        showToast({ variant: 'error', message: t('listings.bulk.pauseError', { title: displayTitle(listing) }) });
      }
    }
    selected = new Set();
    await load();
  }

  async function bulkResume(): Promise<void> {
    for (const listing of selectedListings()) {
      try {
        await resumeSelling(listing);
      } catch {
        showToast({ variant: 'error', message: t('listings.bulk.resumeError', { title: displayTitle(listing) }) });
      }
    }
    selected = new Set();
    await load();
  }

  async function bulkDuplicate(): Promise<void> {
    const targets = selectedListings();
    for (const listing of targets) {
      await duplicateAsDraft(listing);
    }
    selected = new Set();
    showToast({ variant: 'success', message: t('listings.bulk.duplicated', { count: String(targets.length) }) });
  }

  function primaryActionHref(listing: Listing): string {
    const group = groupFor(listing);
    if (group === 'needsAttention' || group === 'draft') return `/listings/${listing.id}`;
    if (group === 'pending') return `/listings/${listing.id}`;
    if (group === 'published') return `/listings/${listing.id}/provenance`;
    return `/listings/${listing.id}`;
  }

  function primaryActionLabel(listing: Listing): string {
    const group = groupFor(listing);
    if (group === 'needsAttention') return t('listings.action.finish');
    if (group === 'draft') return t('listings.action.continue');
    if (group === 'pending') return t('listings.action.review');
    if (group === 'published' && !listing.provenance_id) return t('listings.action.seal');
    return t('listings.action.view');
  }

  function primaryActionTooltip(listing: Listing): string {
    const group = groupFor(listing);
    if (group === 'needsAttention' || group === 'draft') return tooltip('tooltip.editListing');
    if (group === 'published' && !listing.provenance_id) return tooltip('tooltip.sealProvenance');
    return tooltip('tooltip.viewListing');
  }
</script>

<svelte:head>
  <title>{t('listings.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="listings-page">
  <header class="listings-page__header">
    <h1>{t('listings.heading')}</h1>
    <Button size="sm" onclick={() => goto('/listing/new/capture')} tooltip={tooltip('tooltip.newListing')}>
      <Icon name="plus" />
      {t('listings.add')}
    </Button>
  </header>

  <div class="listings-page__search">
    <div class="listings-page__search-input">
      <Icon name="search" aria-hidden="true" />
      <input
        type="search"
        placeholder={t('listings.search.placeholder')}
        bind:value={searchQuery}
        aria-label={t('listings.search.placeholder')}
      />
    </div>
    <VoiceInput class="k-voice--inline" onrecording={onVoiceSearch} disabled={!network.online || voiceSearching} />
  </div>

  <div class="listings-page__filters" role="group" aria-label={t('listings.filter.label')}>
    <button
      type="button"
      class="listings-page__filter"
      class:listings-page__filter--active={stateFilter === 'all'}
      onclick={() => (stateFilter = 'all')}
    >
      {t('listings.filter.all')} ({totalCount})
    </button>
    {#each STATE_GROUPS as group (group)}
      <button
        type="button"
        class="listings-page__filter"
        class:listings-page__filter--active={stateFilter === group}
        onclick={() => (stateFilter = group)}
      >
        {t(STATE_GROUP_LABEL[group])} ({grouped.get(group)?.length ?? 0})
      </button>
    {/each}
  </div>

  {#if selected.size > 0}
    <div class="listings-page__bulk-bar" role="toolbar" aria-label={t('listings.bulk.label')}>
      <p>{t('listings.bulk.count', { count: String(selected.size) })}</p>
      <Button size="sm" variant="secondary" disabled={!anySelectedPauseable} onclick={bulkPause} tooltip={tooltip('tooltip.pauseListing')}>{t('listings.bulk.pause')}</Button>
      <Button size="sm" variant="secondary" disabled={!anySelectedPauseable} onclick={bulkResume} tooltip={tooltip('tooltip.resumeListing')}>{t('listings.bulk.resume')}</Button>
      <Button size="sm" variant="secondary" onclick={bulkDuplicate} tooltip={tooltip('tooltip.duplicateListing')}>{t('listings.bulk.duplicate')}</Button>
      <Button size="sm" variant="ghost" onclick={() => (selected = new Set())} tooltip={tooltip('tooltip.cancel')}>{t('action.cancel')}</Button>
    </div>
  {/if}

  {#if !network.online}
    <p class="listings-page__offline-note">{t('listings.offlineNote')}</p>
  {/if}

  {#if loading && listings.length === 0}
    <div class="listings-page__skeletons">
      {#each Array(4) as _, i (i)}
        <SkeletonRow
          thumbnail
          thumbnailSize="3.5rem"
          lines={[
            { width: '55%', height: '1rem' },
            { width: '30%', height: '0.8rem' },
            { width: '25%', height: '1rem' },
          ]}
          trailing
          trailingWidth="5rem"
        />
      {/each}
    </div>
  {:else if totalCount === 0}
    <EmptyState
      illustration="empty-no-listings"
      heading={t('listings.empty.heading')}
      body={t('listings.empty.body')}
    >
      {#snippet action()}
        <Button onclick={() => goto('/listing/new/capture')} tooltip={tooltip('tooltip.newListing')}>{t('listings.add')}</Button>
      {/snippet}
    </EmptyState>
  {:else}
    {#each STATE_GROUPS as group (group)}
      {@const rows = grouped.get(group) ?? []}
      {#if rows.length > 0 && (stateFilter === 'all' || stateFilter === group)}
        <section class="listings-page__group">
          <h2>{t(STATE_GROUP_LABEL[group])} ({rows.length})</h2>
          <ul class="listings-page__rows" role="list">
            {#each rows as listing (listing.id)}
              <li>
                <Card variant="hairline" element="div" class="listings-page__row">
                  <Checkbox
                    checked={listing.id ? selected.has(listing.id) : false}
                    onchange={() => listing.id && toggleSelect(listing.id)}
                  >
                    <VisuallyHidden>{t('listings.selectRow', { title: displayTitle(listing) })}</VisuallyHidden>
                  </Checkbox>

                  {@const imgSrc = (listing.id && thumbnails[listing.id]) || (listing.id && sihImageMap[listing.id])}
                  {#if imgSrc}
                    <img
                      src={imgSrc}
                      alt=""
                      class="listings-page__thumb"
                      onerror={(e) => {
                        (e.currentTarget as HTMLImageElement).src = '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg';
                      }}
                    />
                  {:else}
                    <div class="listings-page__thumb listings-page__thumb--placeholder">
                      <Icon name="image" />
                    </div>
                  {/if}

                  <div class="listings-page__row-body">
                    <p class="listings-page__row-title">
                      {displayTitle(listing) || t('listings.untitled')}
                    </p>
                    <StateBadge {group} />
                    {#if listing.price?.amount_paise != null}
                      <Money paise={listing.price.amount_paise} />
                    {/if}
                  </div>

                  <Button size="sm" variant="secondary" onclick={() => goto(primaryActionHref(listing))} tooltip={primaryActionTooltip(listing)}>
                    {primaryActionLabel(listing)}
                  </Button>
                  {#if group === 'published'}
                    <Button size="sm" variant="ghost" onclick={() => goto(`/listings/${listing.id}`)} tooltip={tooltip('tooltip.viewGem')}>
                      <Icon name="external-link" />
                      GeM
                    </Button>
                  {/if}
                </Card>
              </li>
            {/each}
          </ul>
        </section>
      {/if}
    {/each}
  {/if}
</div>

<style>
  .listings-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .listings-page__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .listings-page__search {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .listings-page__search-input {
    flex: 1;
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding-inline: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
  }

  .listings-page__search-input input {
    flex: 1;
    min-block-size: var(--k-touch-min);
    border: none;
    background: none;
    font-size: var(--k-text-md);
    color: var(--k-text-primary);
  }

  .listings-page__search-input input:focus {
    outline: none;
  }

  .listings-page__filters {
    display: flex;
    gap: var(--k-space-2);
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    padding-block: var(--k-space-1);
    /* Fade the trailing edge so it's obvious more chips scroll into view. */
    mask-image: linear-gradient(to right, #000 85%, transparent);
  }

  :global([dir='rtl']) .listings-page__filters {
    mask-image: linear-gradient(to left, #000 85%, transparent);
  }

  .listings-page__filters::-webkit-scrollbar {
    display: none;
  }

  .listings-page__filter {
    min-block-size: var(--k-touch-min);
    flex-shrink: 0;
    padding-block: var(--k-space-1);
    padding-inline: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: none;
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    cursor: pointer;
  }

  .listings-page__filter--active {
    border-color: var(--k-accent-primary-bg);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
  }

  .listings-page__bulk-bar {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
    flex-wrap: wrap;
  }

  .listings-page__offline-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .listings-page__skeletons {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .listings-page__group h2 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .listings-page__rows {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  :global(.listings-page__row) {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-2);
  }

  .listings-page__thumb {
    flex-shrink: 0;
    inline-size: 3.5rem;
    block-size: 3.5rem;
    border-radius: var(--k-radius-md);
    object-fit: cover;
  }

  .listings-page__thumb--placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
  }

  .listings-page__row-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    min-inline-size: 0;
  }

  .listings-page__row-title {
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
