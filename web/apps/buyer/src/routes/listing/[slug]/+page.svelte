<!--
  apps/buyer/src/routes/listing/[slug]/+page.svelte

  The product page. `slug` here is the bare listing id -- every internal
  link in this app (ListingCard, search, feed) already points at
  /listing/{listing_id}, matching services/bff/internal/bff/client/slug.go's
  parseSlugID fallback (a separator-less string is treated as the id
  outright), so there's no client-side slug parsing to duplicate.

  GET /listings/{id}/summary (batch 12 extended it) carries everything this
  page needs in one call: media, made_to_order_terms, provenance (technique
  + loom verdicts, sealed_at, qr_code as the public verify URL),
  story_audio_url and the artisan's bio/district -- all real fields, see
  services/bff/internal/bff/client/listing.go's GetListingSummary.

  JSON-LD: mirrors the shape services/bff/internal/bff/handler/seo.go's
  Go-rendered /listing/{slug} emits (name/description/image/brand/offers),
  so a crawler sees the same Product data whichever variant it hits.

  Made-to-order is framed as what it is buying: a lead time, a capacity, an
  advance split and customisation options, never "out of stock" wording or
  styling -- see PurchaseForm below, which renders MTO's own panel before
  the shared quantity/notes fields, not after a stock check.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, matchesLocale, tooltip, type DntTerm } from '@kalakriti/i18n';
  import { EmptyState, Skeleton, Money, AudioPlayback, CraftTerm, Breadcrumbs, type BreadcrumbItem, showToast, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { getListingSummary, type components } from '@kalakriti/api';
  import { getCached, setCached } from '@kalakriti/offline';
  import PurchaseForm from '$lib/PurchaseForm.svelte';

  type ListingSummary = components['schemas']['ListingSummary'];

  const t = $derived(locale.t);
  const listingId = $derived(page.params.slug ?? '');

  let loading = $state(true);
  let listing = $state<ListingSummary | undefined>(undefined);
  let activeMediaIndex = $state(0);
  let devAvatar = $state<string | undefined>(undefined);

  $effect(() => {
    function syncAvatar(): void {
      try {
        if (typeof localStorage !== 'undefined') {
          const a = localStorage.getItem('kalakriti.artisan.avatar');
          if (a) devAvatar = a;
        }
      } catch {}
    }
    syncAvatar();
    if (typeof window !== 'undefined') {
      window.addEventListener('storage', syncAvatar);
      return () => window.removeEventListener('storage', syncAvatar);
    }
    return undefined;
  });

  const artisanAvatar = $derived((listing as Record<string, unknown> | undefined)?.artisan_image_url as string | undefined || devAvatar);

  // Caches the full response -- every language's translations[], not just the
  // one active at fetch time -- so a buyer who switches language offline on a
  // listing they've already viewed still sees it correctly translated instead
  // of falling back to whatever locale was active when it was cached. Same
  // network -> cache pattern as apps/artisan/src/lib/ontology.ts's loadCrafts.
  function listingCacheKey(id: string): string {
    return `listing.summary.${id}`;
  }

  $effect(() => {
    const id = listingId;
    void (async () => {
      loading = true;
      activeMediaIndex = 0;
      try {
        listing = await getListingSummary(id);
        await setCached(listingCacheKey(id), listing);
      } catch {
        listing = await getCached<ListingSummary>(listingCacheKey(id));
      } finally {
        loading = false;
      }
    })();
  });

  const title = $derived(
    listing?.translations?.find((tr) => matchesLocale(tr.language, locale.code))?.title ??
      listing?.translations?.[0]?.title ??
      '',
  );
  const description = $derived(
    listing?.translations?.find((tr) => matchesLocale(tr.language, locale.code))?.description ??
      listing?.translations?.[0]?.description ??
      '',
  );
  const media = $derived(listing?.media ?? []);
  const activeMedia = $derived(media[activeMediaIndex]);
  const madeToOrder = $derived(listing?.type === 'MADE_TO_ORDER');
  const terms = $derived(listing?.made_to_order_terms);
  const provenance = $derived(listing?.provenance);
  const craftTerm = $derived<DntTerm | undefined>(
    listing?.craft_name
      ? {
          term: listing.craft_name,
          canonicalName: listing.craft_name,
          giStatus: listing.craft_gi_registration_no ? t('listing.giBadge') : undefined,
        }
      : undefined,
  );

  const breadcrumbItems = $derived<BreadcrumbItem[]>([
    { label: t('nav.home') || 'Home', href: '/' },
    {
      label: listing?.craft_name ?? 'Craft Corridor',
      href: listing?.craft_slug ? `/craft/${listing.craft_slug}` : '/catalog',
    },
    { label: title || 'Listing' },
  ]);

  let copied = $state(false);

  function handleShare(): void {
    const fullUrl = typeof window !== 'undefined' ? window.location.href : page.url.href;
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard
        .writeText(fullUrl)
        .then(() => {
          copied = true;
          setTimeout(() => {
            copied = false;
          }, 2000);
          showToast({ message: 'Artisan piece link copied to clipboard!', variant: 'success' });
        })
        .catch(() => {
          showToast({ message: 'Failed to copy link', variant: 'error' });
        });
    }
  }

  const jsonLd = $derived(
    listing
      ? JSON.stringify({
          '@context': 'https://schema.org/',
          '@type': 'Product',
          name: title,
          description,
          image: listing.image_url,
          brand: { '@type': 'Brand', name: listing.artisan_name },
          offers: {
            '@type': 'Offer',
            price: (listing.price?.amount_paise ?? 0) / 100,
            priceCurrency: listing.price?.currency_code ?? 'INR',
            availability:
              listing.type === 'READY_STOCK' && (listing.stock_quantity ?? 0) <= 0
                ? 'https://schema.org/OutOfStock'
                : 'https://schema.org/InStock',
            url: page.url.href,
          },
        })
      : '',
  );
</script>

<svelte:head>
  <title>{title ? `${title} — Handcrafted by ${listing?.artisan_name ?? 'Master Artisan'} | Kalakriti` : `Authentic Heritage Craft | ${t('app.name')}`}</title>
  <meta name="description" content={description ? description.slice(0, 160) : `Discover ${title || 'handcrafted artisan works'} by ${listing?.artisan_name ?? 'certified artisans'} on Kalakriti.`} />
  <meta property="og:title" content={title ? `${title} — Handcrafted by ${listing?.artisan_name ?? 'Master Artisan'}` : 'Authentic Handcrafted Heritage'} />
  <meta property="og:description" content={description ? description.slice(0, 200) : 'Authentic GI-certified handcrafted piece direct from verified artisan looms.'} />
  <meta property="og:image" content={listing?.image_url ?? 'https://kalakriti.gov.in/og-craft.jpg'} />
  <meta property="og:type" content="product" />
  <meta property="og:url" content={page.url.href} />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content={title ? `${title} — Handcrafted by ${listing?.artisan_name ?? 'Master Artisan'}` : 'Authentic Handcrafted Heritage'} />
  <meta name="twitter:description" content={description ? description.slice(0, 200) : 'Authentic GI-certified handcrafted piece direct from verified artisan looms.'} />
  <meta name="twitter:image" content={listing?.image_url ?? 'https://kalakriti.gov.in/og-craft.jpg'} />
  {#if listing}
    {@html `<script type="application/ld+json">${jsonLd}<\/script>`}
  {/if}
</svelte:head>

{#if loading}
  <Skeleton shape="card" height="28rem" />
{:else if !listing}
  <EmptyState illustration="empty-error" heading={t('listing.notFound')} />
{:else}
  <div class="listing-page-wrap">
    <div class="listing-breadcrumbs-bar">
      <Breadcrumbs items={breadcrumbItems} />
    </div>

    <div class="listing">
      <div class="listing__gallery">
        <div class="listing__stage">
          {#if activeMedia?.kind === 'VIDEO'}
            <video src={activeMedia.url} controls playsinline class="listing__stage-media">
              <track kind="captions" />
            </video>
          {:else if activeMedia?.url}
            <img src={activeMedia.url} alt={title ? `${title} — Authentic craft photo` : 'Craft piece view'} class="listing__stage-media" />
          {/if}
        </div>
      {#if media.length > 1}
        <div class="listing__thumbs" role="tablist" aria-label={t('listing.gallery.processVideo')}>
          {#each media as item, index (index)}
            <Tooltip text={tooltip('tooltip.selectImage')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                role="tab"
                aria-selected={index === activeMediaIndex}
                class="listing__thumb"
                class:listing__thumb--active={index === activeMediaIndex}
                onclick={() => (activeMediaIndex = index)}
                {...tp}
              >
                {#if item.kind === 'VIDEO'}
                  <span class="listing__thumb-video"><Icon name="process-video" title={t('listing.gallery.processVideo')} /></span>
                {:else if item.url}
                  <img src={item.url} alt={title ? `${title} photo ${index + 1}` : `Photo ${index + 1}`} />
                {/if}
              </button>
            {/snippet}
          </Tooltip>
          {/each}
        </div>
      {/if}
    </div>

    <div class="listing__info">
      <div class="listing__top-row">
        <p class="listing__craft">
          {#if craftTerm}<CraftTerm term={craftTerm} />{:else}{listing.craft_name}{/if}
          {#if listing.gi_certified}
            <span class="listing__gi-badge"><Icon name="gi-tagged" />{t('listing.giBadge')}</span>
          {/if}
        </p>

        <Tooltip text={tooltip('tooltip.share')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="listing__share-btn"
              onclick={handleShare}
              title={t('listing.shareTitle')}
              aria-label={t('listing.shareAriaLabel')}
              {...tp}
            >
              <Icon name={copied ? 'check' : 'share'} size="0.95rem" />
              <span>{copied ? t('listing.shareCopied') : t('listing.shareButton')}</span>
            </button>
          {/snippet}
        </Tooltip>
      </div>

      <h1>{title}</h1>

      <!-- 24-Hour Response Time Promise -->
      <div class="listing__promise-card">
        <Icon name="verified-artisan" size="1.25rem" />
        <div class="listing__promise-copy">
          <strong>{t('listing.responseGuarantee')}</strong>
          <span>{t('listing.responseGuaranteeSub')}</span>
        </div>
      </div>
      {#if listing.artisan_name}
        {@const artisanName = listing.artisan_name}
        {@const artisanDistrict = listing.artisan_district}
        <Tooltip text={tooltip('tooltip.viewArtisan')}>
        {#snippet trigger(tp)}
          <a href="/artisan/{encodeURIComponent(artisanName.toLowerCase())}" class="listing__artisan-badge" title={t('listing.viewArtisanProfileTitle')} {...tp}>
            <div class="listing__artisan-avatar">
              {#if artisanAvatar}
                <img src={artisanAvatar} alt={artisanName} class="listing__artisan-avatar-img" />
              {:else}
                <span class="listing__artisan-avatar-initial">{artisanName.charAt(0).toUpperCase()}</span>
              {/if}
              <span class="listing__artisan-verified" title={t('profile.verifiedBadgeTitle')}>
                <Icon name="verified-artisan" />
              </span>
            </div>
            <div class="listing__artisan-info">
              <span class="listing__artisan-name">{t('listing.by', { name: artisanName })}</span>
              <span class="listing__artisan-sub">
                {#if artisanDistrict}<span>{artisanDistrict}</span> • {/if}
                <span class="listing__artisan-view">{t('listing.viewArtisanStorefront')}</span>
              </span>
            </div>
          </a>
        {/snippet}
      </Tooltip>
      {/if}
      <p class="listing__price"><Money paise={listing.price?.amount_paise ?? 0} /></p>

      {#if madeToOrder && terms}
        <section class="listing__mto" aria-labelledby="mto-heading">
          <h2 id="mto-heading" class="listing__mto-heading">
            <Icon name="made-to-order" />
            {t('listing.mto.leadTime', { days: String(terms.lead_time_days ?? ''), artisan: listing.artisan_name ?? '' })}
          </h2>
          <ul class="listing__mto-facts">
            <li>{t('listing.mto.capacity', { count: String(terms.capacity_per_month ?? '') })}</li>
            <li>{t('listing.mto.advance', { pct: String(terms.advance_pct ?? 0) })}</li>
            {#if terms.accepting_orders === false}
              <li class="listing__mto-paused">{t('listing.mto.notAccepting')}</li>
            {/if}
          </ul>
        </section>
      {:else if listing.type === 'READY_STOCK'}
        <p class="listing__ready"><Icon name="ready-stock" />{t('listing.readyStock')}</p>
      {/if}

      <PurchaseForm {listing} />

      {#if listing.artisan_bio || listing.story_audio_url}
        <section class="listing__story" aria-labelledby="story-heading">
          <h2 id="story-heading">{t('listing.artisanStory.heading')}</h2>
          {#if listing.artisan_bio}<p>{listing.artisan_bio}</p>{/if}
          {#if listing.story_audio_url}
            <AudioPlayback src={listing.story_audio_url} label={t('listing.story.listen')} />
          {/if}
        </section>
      {/if}

      <section class="listing__provenance" aria-labelledby="provenance-heading">
        <h2 id="provenance-heading"><Icon name="provenance" />{t('listing.provenance.heading')}</h2>
        {#if provenance}
          <dl class="listing__provenance-fields">
            <div>
              <dt>{t('listing.provenance.technique')}</dt>
              <dd>
                {#if provenance.technique_verdict?.matches}
                  <Icon name="handmade-certified" />{t('listing.provenance.techniqueMatch')}
                {:else}
                  {t('listing.provenance.techniqueMismatch')}
                {/if}
              </dd>
            </div>
            {#if provenance.loom_verdict}
              <div>
                <dt>{t('listing.provenance.handloom')}</dt>
                <dd>
                  <Icon name="handloom-verified" />
                  {provenance.loom_verdict.is_handloom ? t('listing.provenance.handloomYes') : t('listing.provenance.handloomNo')}
                </dd>
              </div>
            {/if}
            <div>
              <dt>{t('listing.provenance.sealedLabel')}</dt>
              <dd>{new Date(provenance.sealed_at ?? '').toLocaleDateString(locale.code)}</dd>
            </div>
          </dl>
          {#if provenance.qr_code}
            <a class="listing__verify-link" href={provenance.qr_code}>{t('listing.provenance.verifyLink')}</a>
          {/if}
        {:else}
          <p class="listing__provenance-empty">{t('listing.provenance.unsealed')}</p>
        {/if}
      </section>
    </div>
  </div>

  <!-- Sticky Mobile CTA Dock -->
  <aside class="sticky-mobile-dock" aria-label={t('listing.quickOrderDockAriaLabel')}>
    <div class="sticky-mobile-dock__price">
      <span class="dock-label">{madeToOrder ? 'Advance Split' : 'Direct Price'}</span>
      <span class="dock-amount"><Money paise={listing.price?.amount_paise ?? 0} /></span>
    </div>
    <Tooltip text={tooltip('tooltip.next')}>
      {#snippet trigger(tp)}
        <button
          type="button"
          class="sticky-mobile-dock__action"
          onclick={() => {
            const form = document.querySelector('.listing__info');
            form?.scrollIntoView({ behavior: 'smooth' });
          }}
          {...tp}
        >
          <Icon name={madeToOrder ? 'made-to-order' : 'ready-stock'} size="1rem" />
          <span>{madeToOrder ? 'Commission Piece' : 'Acquire Now'}</span>
        </button>
      {/snippet}
    </Tooltip>
  </aside>
</div>
{/if}

<style>
  .listing {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-6);
  }

  .listing__stage {
    aspect-ratio: 4/5;
    background-color: var(--k-surface-sunken);
    border-radius: var(--k-radius-lg);
    overflow: hidden;
  }

  .listing__stage-media {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .listing__thumbs {
    display: flex;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-2);
    overflow-x: auto;
  }

  .listing__thumb {
    flex: 0 0 auto;
    inline-size: 3.5rem;
    block-size: 3.5rem;
    border-radius: var(--k-radius-md);
    overflow: hidden;
    border: var(--k-hairline) solid transparent;
    padding: 0;
    background: var(--k-surface-sunken);
    cursor: pointer;
  }

  .listing__thumb--active {
    border-color: var(--k-border-interactive);
  }

  .listing__thumb img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .listing__thumb-video {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 100%;
    block-size: 100%;
  }

  .listing__craft {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
    color: var(--k-text-secondary);
  }

  .listing__gi-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-accent-primary-text);
  }

  .listing__artisan-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-2) var(--k-space-3);
    margin-block-end: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
    text-decoration: none;
    color: inherit;
    transition: background-color var(--k-duration-fast) var(--k-ease-standard),
                border-color var(--k-duration-fast) var(--k-ease-standard);
  }

  .listing__artisan-badge:hover {
    background-color: var(--k-surface-raised);
    border-color: var(--k-accent-primary-border, var(--k-border-accent));
  }

  .listing__artisan-avatar {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-terracotta-700, var(--k-accent-primary-bg));
    color: var(--k-text-on-accent);
    font-weight: var(--k-weight-bold);
    font-size: var(--k-text-md);
    flex-shrink: 0;
  }

  .listing__artisan-avatar-img {
    inline-size: 100%;
    block-size: 100%;
    border-radius: var(--k-radius-pill);
    object-fit: cover;
  }

  .listing__artisan-verified {
    position: absolute;
    inset-inline-end: -4px;
    inset-block-end: -4px;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 1.2rem;
    block-size: 1.2rem;
    border: 1.5px solid var(--k-surface-base, var(--k-khadi-50));
    border-radius: var(--k-radius-pill);
    background-color: var(--k-neem-600, var(--k-accent-success-bg));
    color: var(--k-text-on-accent);
  }

  .listing__artisan-info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    text-align: start;
  }

  .listing__artisan-name {
    font-weight: var(--k-weight-semibold);
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .listing__artisan-sub {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .listing__artisan-view {
    color: var(--k-accent-primary-text, var(--k-accent-primary-text));
  }

  .listing__price {
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-semibold);
    margin-block-end: var(--k-space-4);
  }

  .listing__mto {
    border-block: var(--k-hairline) solid var(--k-border-hairline);
    padding-block: var(--k-space-3);
    margin-block-end: var(--k-space-4);
  }

  .listing__mto-heading {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-md);
    margin: 0 0 var(--k-space-2);
  }

  .listing__mto-facts {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .listing__mto-paused {
    color: var(--k-accent-danger);
  }

  .listing__ready {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-accent-success);
    margin-block-end: var(--k-space-4);
  }

  .listing__story,
  .listing__provenance {
    margin-block-start: var(--k-space-6);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-4);
  }

  .listing__provenance h2 {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .listing__provenance-fields {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin: var(--k-space-3) 0;
  }

  .listing__provenance-fields > div {
    display: flex;
    justify-content: space-between;
    gap: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-end: var(--k-space-2);
  }

  .listing__provenance-fields dt {
    color: var(--k-text-secondary);
  }

  .listing__provenance-fields dd {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    margin: 0;
  }

  .listing__provenance-empty {
    color: var(--k-text-secondary);
  }

  .listing__verify-link {
    color: var(--k-accent-secondary);
  }

  @media (min-width: 60rem) {
    .listing {
      grid-template-columns: 1fr 1fr;
    }
  }

  .listing-page-wrap {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .listing-breadcrumbs-bar {
    margin-block-end: var(--k-space-2);
  }

  .listing__top-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
  }

  .listing__share-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    padding: var(--k-space-1) var(--k-space-2);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    color: var(--k-text-secondary);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .listing__share-btn:hover {
    color: var(--k-terracotta-700, var(--k-accent-primary-text));
    border-color: var(--k-terracotta-500, var(--k-border-accent));
  }

  .listing__promise-card {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-100, var(--k-surface-base));
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    margin-block: var(--k-space-2) var(--k-space-3);
    color: var(--k-text-primary);
  }

  .listing__promise-copy {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    font-size: var(--k-text-xs);
  }

  .listing__promise-copy strong {
    color: var(--k-terracotta-800, var(--k-terracotta-800));
    font-weight: var(--k-weight-semibold);
  }

  .listing__promise-copy span {
    color: var(--k-text-secondary);
  }

  /* Sticky Mobile CTA Dock */
  .sticky-mobile-dock {
    display: none;
  }

  @media (max-width: 48rem) {
    .listing-page-wrap {
      padding-block-end: calc(5rem + env(safe-area-inset-bottom, 0px));
    }

    .sticky-mobile-dock {
      position: fixed;
      inset-block-end: 0;
      inset-inline: 0;
      z-index: 50;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--k-space-3);
      padding: var(--k-space-3) var(--k-space-4);
      padding-block-end: calc(var(--k-space-3) + env(safe-area-inset-bottom, 0px));
      background-color: rgba(255, 255, 255, 0.95);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      border-block-start: var(--k-hairline) solid var(--k-border-hairline);
      box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.08);
    }

    .sticky-mobile-dock__price {
      display: flex;
      flex-direction: column;
    }

    .dock-label {
      font-size: var(--k-text-xs);
      color: var(--k-text-secondary);
      text-transform: uppercase;
      letter-spacing: var(--k-tracking-wide);
    }

    .dock-amount {
      font-size: var(--k-text-lg);
      font-weight: var(--k-weight-bold);
      color: var(--k-text-primary);
    }

    .sticky-mobile-dock__action {
      display: inline-flex;
      align-items: center;
      gap: var(--k-space-2);
      padding: var(--k-space-2) var(--k-space-4);
      border-radius: var(--k-radius-md);
      background-color: var(--k-terracotta-700, var(--k-accent-primary-bg));
      color: var(--k-text-on-accent);
      font-weight: var(--k-weight-semibold);
      font-size: var(--k-text-sm);
      border: none;
      cursor: pointer;
      box-shadow: 0 2px 6px rgba(150, 56, 30, 0.3);
    }

    .sticky-mobile-dock__action:active {
      transform: scale(0.98);
    }
  }

  @media (max-width: 360px) {
    .sticky-mobile-dock {
      padding: var(--k-space-2) var(--k-space-3);
      padding-block-end: calc(var(--k-space-2) + env(safe-area-inset-bottom, 0px));
    }
    .sticky-mobile-dock__action {
      padding: var(--k-space-2) var(--k-space-3);
      font-size: var(--k-text-xs);
    }
    .dock-amount {
      font-size: var(--k-text-base);
    }
  }
</style>
