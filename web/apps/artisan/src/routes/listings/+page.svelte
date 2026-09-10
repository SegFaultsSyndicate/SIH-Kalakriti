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
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, Checkbox, EmptyState, Money, VisuallyHidden, VoiceInput, showToast } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { searchVoice } from '@kalakriti/api';
  import StateBadge from '$lib/StateBadge.svelte';
  import {
    STATE_GROUPS,
    STATE_GROUP_LABEL,
    fetchMyListings,
    cachedListings,
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

  const t = $derived(locale.t);

  let listings = $state<Listing[]>([]);
  let loading = $state(true);
  let stateFilter = $state<StateGroup | 'all'>('all');
  let searchQuery = $state('');
  let voiceSearching = $state(false);
  let selected = $state(new Set<string>());
  let thumbnails = $state<Record<string, string>>({});
  let pauseable = $state<Record<string, boolean>>({});

  async function load(): Promise<void> {
    loading = true;
    listings = await cachedListings();
    if (network.online) {
      try {
        listings = await fetchMyListings();
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
      const url = await localPrimaryImageUrl(listing.id);
      if (url) urls[listing.id] = url;
      canPause[listing.id] = await canPauseOrResume(listing);
    }
    thumbnails = urls;
    pauseable = canPause;
  }

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
      if (query && !titleFor(listing, locale.code).toLowerCase().includes(query)) continue;
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
        showToast({ variant: 'error', message: t('listings.bulk.pauseError', { title: titleFor(listing, locale.code) }) });
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
        showToast({ variant: 'error', message: t('listings.bulk.resumeError', { title: titleFor(listing, locale.code) }) });
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
</script>

<svelte:head>
  <title>{t('listings.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="listings-page">
  <header class="listings-page__header">
    <h1>{t('listings.heading')}</h1>
    <Button size="sm" onclick={() => goto('/listing/new/capture')}>
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
    <VoiceInput onrecording={onVoiceSearch} disabled={!network.online || voiceSearching} />
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
      <Button size="sm" variant="secondary" disabled={!anySelectedPauseable} onclick={bulkPause}>{t('listings.bulk.pause')}</Button>
      <Button size="sm" variant="secondary" disabled={!anySelectedPauseable} onclick={bulkResume}>{t('listings.bulk.resume')}</Button>
      <Button size="sm" variant="secondary" onclick={bulkDuplicate}>{t('listings.bulk.duplicate')}</Button>
      <Button size="sm" variant="ghost" onclick={() => (selected = new Set())}>{t('action.cancel')}</Button>
    </div>
  {/if}

  {#if !network.online}
    <p class="listings-page__offline-note">{t('listings.offlineNote')}</p>
  {/if}

  {#if loading && listings.length === 0}
    <div class="listings-page__skeletons">
      <Skeleton shape="card" height="4rem" />
      <Skeleton shape="card" height="4rem" />
      <Skeleton shape="card" height="4rem" />
    </div>
  {:else if totalCount === 0}
    <EmptyState
      illustration="empty-no-listings"
      heading={t('listings.empty.heading')}
      body={t('listings.empty.body')}
    >
      {#snippet action()}
        <Button onclick={() => goto('/listing/new/capture')}>{t('listings.add')}</Button>
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
                    <VisuallyHidden>{t('listings.selectRow', { title: titleFor(listing, locale.code) })}</VisuallyHidden>
                  </Checkbox>

                  {#if listing.id && thumbnails[listing.id]}
                    <img src={thumbnails[listing.id]} alt="" class="listings-page__thumb" />
                  {:else}
                    <div class="listings-page__thumb listings-page__thumb--placeholder">
                      <Icon name="image" />
                    </div>
                  {/if}

                  <div class="listings-page__row-body">
                    <p class="listings-page__row-title">
                      {titleFor(listing, locale.code) || t('listings.untitled')}
                    </p>
                    <StateBadge {group} />
                    {#if listing.price?.amount_paise != null}
                      <Money paise={listing.price.amount_paise} />
                    {/if}
                  </div>

                  <Button size="sm" variant="secondary" onclick={() => goto(primaryActionHref(listing))}>
                    {primaryActionLabel(listing)}
                  </Button>
                  {#if group === 'published'}
                    <Button size="sm" variant="ghost" onclick={() => goto(`/listings/${listing.id}`)}>
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
</main>

<style>
  .listings-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
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
    margin-inline: calc(-1 * var(--k-space-4));
    padding-inline: var(--k-space-4);
  }

  .listings-page__filters::-webkit-scrollbar {
    display: none;
  }

  @media (max-width: 32rem) {
    .listings-page {
      padding: var(--k-space-3);
    }
    .listings-page__filters {
      margin-inline: calc(-1 * var(--k-space-3));
      padding-inline: var(--k-space-3);
    }
  }

  .listings-page__filter {
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
