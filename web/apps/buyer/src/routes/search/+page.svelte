<!--
  apps/buyer/src/routes/search/+page.svelte

  Multilingual search: one input, any script, voice alongside it. The
  understood-filters chips are not decoration -- they are exactly what
  services/bff/internal/bff/client/search.go's understoodToMap returns from
  search-svc's own query understander, rendered as removable state so a
  buyer can see and correct what the system parsed. machine_generated on a
  hit means the copy that matched was a translation, not the artisan's own
  words in this language -- the honest, backend-real "matched across
  languages" signal (see search-svc's HydrateSearchHits query).

  Zero results: search-svc's own sibling-craft fallback already substitutes
  adjacent-craft hits and returns did_you_mean names (domain.ApplyBusinessRules
  never runs an empty page) -- this renders the banner, not a second query.

  made-to-order listings are never sorted or filtered down for lacking
  stock: domain.ApplyBusinessRules on search-svc already handles that
  server-side (see services/search-svc/internal/search/service/search.go's
  point i), and ListingCard renders both stock badges at equal weight.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Input, Button, Chip, Dialog, VoiceInput, EmptyState, Skeleton, Switch } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { search, searchVoice, getListingSummary, suggest, type components } from '@kalakriti/api';
  import ListingCard from '$lib/ListingCard.svelte';

  type SearchHit = components['schemas']['SearchHit'];
  type ListingSummary = components['schemas']['ListingSummary'];

  const t = $derived(locale.t);

  let q = $state(page.url.searchParams.get('q') ?? page.url.searchParams.get('category') ?? '');
  let giOnly = $state(page.url.searchParams.get('gi_tagged') === 'true');
  let madeToOrder = $state(page.url.searchParams.get('made_to_order') === 'true');
  let material = $state(page.url.searchParams.get('material') ?? '');
  let limit = $state(20);

  let loading = $state(true);
  let hits = $state<SearchHit[]>([]);
  let summaries = $state<Record<string, ListingSummary>>({});
  let understood = $state<components['schemas']['StructuredFiltersUnderstood'] | undefined>(undefined);
  let didYouMean = $state<string[]>([]);
  let voiceOpen = $state(false);

  let suggestions = $state<string[]>([]);
  let showSuggestions = $state(false);

  $effect(() => {
    const trimmed = q.trim();
    if (trimmed.length < 2) {
      suggestions = [];
      showSuggestions = false;
      return;
    }
    const timer = setTimeout(async () => {
      try {
        const res = await suggest(trimmed);
        suggestions = res.suggestions ?? [];
        showSuggestions = suggestions.length > 0;
      } catch {
        suggestions = [];
        showSuggestions = false;
      }
    }, 200);
    return () => clearTimeout(timer);
  });

  function selectSuggestion(item: string): void {
    q = item;
    showSuggestions = false;
    limit = 20;
    const params = new URLSearchParams({ q });
    if (giOnly) params.set('gi_tagged', 'true');
    if (madeToOrder) params.set('made_to_order', 'true');
    if (material) params.set('material', material);
    void goto(`/search?${params.toString()}`, { keepFocus: true, noScroll: true });
    void runSearch();
  }

  async function runSearch(): Promise<void> {
    loading = true;
    try {
      const res = await search({
        q,
        gi_tagged: giOnly ? 'true' : undefined,
        made_to_order: madeToOrder ? 'true' : undefined,
        material: material ? [material] : undefined,
        limit: String(limit),
      });
      hits = res.results ?? [];
      understood = res.understood;
      didYouMean = res.did_you_mean ?? [];

      const missing = hits.filter((h) => h.listing_id && !summaries[h.listing_id]).map((h) => h.listing_id!);
      if (missing.length > 0) {
        const fetched = await Promise.allSettled(missing.map((id) => getListingSummary(id)));
        const next = { ...summaries };
        fetched.forEach((r, i) => {
          if (r.status === 'fulfilled') next[missing[i]] = r.value;
        });
        summaries = next;
      }
    } finally {
      loading = false;
    }
  }

  function onSubmit(event: SubmitEvent): void {
    event.preventDefault();
    limit = 20;
    const params = new URLSearchParams({ q });
    if (giOnly) params.set('gi_tagged', 'true');
    if (madeToOrder) params.set('made_to_order', 'true');
    if (material) params.set('material', material);
    void goto(`/search?${params.toString()}`, { keepFocus: true, noScroll: true });
    void runSearch();
  }

  function removeUnderstood(field: 'gi_only' | 'made_to_order' | 'material' | 'colour' | 'price', value?: string): void {
    if (field === 'gi_only') giOnly = false;
    if (field === 'made_to_order') madeToOrder = false;
    if (field === 'material' && value) {
      material = material === value ? '' : material;
      if (understood) understood = { ...understood, materials: (understood.materials ?? []).filter((item) => item !== value) };
    } else if (field === 'colour' && value && understood) {
      understood = { ...understood, colours: (understood.colours ?? []).filter((item) => item !== value) };
    } else if (field === 'price' && understood) {
      understood = { ...understood, min_price: undefined, max_price: undefined };
    } else {
      void runSearch();
    }
  }

  function loadMore(): void {
    limit += 20;
    void runSearch();
  }

  let sentinel: HTMLDivElement | undefined = $state();
  $effect(() => {
    if (!sentinel) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting && !loading) loadMore();
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  });

  async function onVoiceRecording(blob: Blob): Promise<void> {
    voiceOpen = false;
    loading = true;
    try {
      const res = await searchVoice(blob, locale.code);
      q = (res as { query?: string }).query ?? q;
      hits = res.results ?? [];
      understood = res.understood;
      didYouMean = res.did_you_mean ?? [];
      const ids = hits.map((h) => h.listing_id).filter((id): id is string => !!id);
      const fetched = await Promise.allSettled(ids.map((id) => getListingSummary(id)));
      const next: Record<string, ListingSummary> = {};
      fetched.forEach((r, i) => {
        if (r.status === 'fulfilled') next[ids[i]] = r.value;
      });
      summaries = next;
    } finally {
      loading = false;
    }
  }

  function highlight(text: string | undefined, terms: string[] | undefined): string {
    if (!text) return '';
    if (!terms || terms.length === 0) return text;
    const pattern = terms
      .filter(Boolean)
      .sort((a, b) => b.length - a.length)
      .map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
      .join('|');
    if (!pattern) return text;
    return text.replace(new RegExp(`(${pattern})`, 'gi'), '<mark>$1</mark>');
  }

  $effect(() => {
    const urlQ = page.url.searchParams.get('q') ?? page.url.searchParams.get('category') ?? '';
    const urlGi = page.url.searchParams.get('gi_tagged') === 'true';
    const urlMto = page.url.searchParams.get('made_to_order') === 'true';
    const urlMat = page.url.searchParams.get('material') ?? '';

    q = urlQ;
    giOnly = urlGi;
    madeToOrder = urlMto;
    material = urlMat;

    void runSearch();
  });
</script>

<svelte:head>
  <title>{t('search.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="search-container">
  <form class="search-bar" onsubmit={onSubmit}>
    <Input type="search" bind:value={q} placeholder={t('search.placeholder')} onfocus={() => { if (suggestions.length > 0) showSuggestions = true; }} />
    <button type="button" class="search-bar__voice" onclick={() => (voiceOpen = true)} aria-label={t('search.voice.label')}>
      <Icon name="microphone" />
    </button>
    <Button type="submit">{t('nav.search')}</Button>
  </form>

  {#if showSuggestions && suggestions.length > 0}
    <ul class="search-suggestions" role="listbox">
      {#each suggestions as item}
        <li>
          <button type="button" class="search-suggestions__item" onclick={() => selectSuggestion(item)}>
            <Icon name="search" />
            <span>{item}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

{#if understood}
  <div class="chips" role="list">
    {#if understood.gi_only}<Chip label={t('search.filters.giOnly')} onremove={() => removeUnderstood('gi_only')} />{/if}
    {#if understood.listing_type === 'MADE_TO_ORDER'}
      <Chip label={t('search.filters.madeToOrder')} onremove={() => removeUnderstood('made_to_order')} />
    {/if}
    {#each understood.colours ?? [] as colour (colour)}<Chip label={colour} onremove={() => removeUnderstood('colour', colour)} />{/each}
    {#each understood.materials ?? [] as m (m)}<Chip label={m} onremove={() => removeUnderstood('material', m)} />{/each}
    {#if understood.min_price || understood.max_price}
      <Chip label={t('search.filters.price')} onremove={() => removeUnderstood('price')} />
    {/if}
  </div>
{/if}

<div class="facets">
  <Switch checked={giOnly} onchange={(e: Event) => { giOnly = (e.currentTarget as HTMLInputElement).checked; void runSearch(); }}>
    {t('search.filters.giOnly')}
  </Switch>
  <Switch checked={madeToOrder} onchange={(e: Event) => { madeToOrder = (e.currentTarget as HTMLInputElement).checked; void runSearch(); }}>
    {t('search.filters.madeToOrder')}
  </Switch>
</div>

{#if didYouMean.length > 0}
  <p class="did-you-mean">
    {t('search.zeroResults.heading')}. {t('search.zeroResults.didYouMean')}
    {didYouMean.join(', ')}
  </p>
{/if}

{#if loading && hits.length === 0}
  <div class="results-grid">
    {#each Array(6) as _, i (i)}<Skeleton shape="card" height="16rem" />{/each}
  </div>
{:else if hits.length === 0}
  <EmptyState illustration="empty-no-search-results" heading={t('search.zeroResults.heading')} />
{:else}
  <div class="results-grid">
    {#each hits as hit (hit.listing_id)}
      {@const summary = hit.listing_id ? summaries[hit.listing_id] : undefined}
      {#if summary}
        <div class="result">
          <ListingCard listing={summary} href={`/listing/${hit.listing_id}`} crossLingual={!!hit.machine_generated} />
          {#if hit.matched_terms && hit.matched_terms.length > 0}
            <p class="result__matched">
              {@html highlight(summary.translations?.[0]?.title, hit.matched_terms)}
            </p>
          {/if}
          {#if hit.explanation}
            <details class="result__explanation">
              <summary>{t('search.explanation.show')}</summary>
              <p>{hit.explanation}</p>
            </details>
          {/if}
        </div>
      {/if}
    {/each}
  </div>

  <div bind:this={sentinel}></div>
  <div class="load-more">
    <Button variant="secondary" onclick={loadMore} loading={loading && hits.length > 0}>{t('search.loadMore')}</Button>
  </div>
{/if}

<Dialog bind:open={voiceOpen} title={t('search.voice.label')}>
  <VoiceInput onrecording={onVoiceRecording} />
</Dialog>

<style>
  .search-container {
    position: relative;
  }

  .search-bar {
    display: flex;
    gap: var(--k-space-2);
    align-items: center;
    margin-block: var(--k-space-4);
  }

  .search-suggestions {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    z-index: 10;
    background: var(--k-surface-elevated, #fff);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
    margin: 0;
    margin-block-start: -0.5rem;
    margin-block-end: var(--k-space-4);
    padding: var(--k-space-1);
    list-style: none;
  }

  .search-suggestions__item {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    width: 100%;
    padding: var(--k-space-2) var(--k-space-3);
    border: none;
    background: none;
    border-radius: var(--k-radius-sm);
    text-align: left;
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .search-suggestions__item:hover,
  .search-suggestions__item:focus-visible {
    background-color: var(--k-surface-sunken);
  }

  .search-bar :global(input) {
    flex: 1;
  }

  .search-bar__voice {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: none;
    cursor: pointer;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-4);
  }

  .facets {
    display: flex;
    gap: var(--k-space-4);
    margin-block-end: var(--k-space-4);
  }

  .did-you-mean {
    padding: var(--k-space-3);
    background-color: var(--k-surface-sunken);
    border-radius: var(--k-radius-md);
    margin-block-end: var(--k-space-4);
  }

  .results-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--k-space-5);
  }

  .result__matched {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-start: var(--k-space-1);
  }

  .result__matched :global(mark) {
    background-color: var(--k-accent-primary-bg, yellow);
    color: inherit;
  }

  .result__explanation {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .load-more {
    display: flex;
    justify-content: center;
    margin-block-start: var(--k-space-6);
  }

  @media (min-width: 48rem) {
    .results-grid {
      grid-template-columns: repeat(4, 1fr);
    }
  }
</style>
