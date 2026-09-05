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
  import { locale } from '@kalakriti/i18n';
  import type { DntTerm } from '@kalakriti/i18n';
  import { EmptyState, Skeleton, Money, AudioPlayback, CraftTerm } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { getListingSummary, type components } from '@kalakriti/api';
  import PurchaseForm from '$lib/PurchaseForm.svelte';

  type ListingSummary = components['schemas']['ListingSummary'];

  const t = $derived(locale.t);
  const listingId = $derived(page.params.slug ?? '');

  let loading = $state(true);
  let listing = $state<ListingSummary | undefined>(undefined);
  let activeMediaIndex = $state(0);
  let devAvatar = $state<string | undefined>(undefined);

  $effect(() => {
    try {
      if (typeof localStorage !== 'undefined') {
        const a = localStorage.getItem('kalakriti.artisan.avatar');
        if (a) devAvatar = a;
      }
    } catch {}
  });

  const artisanAvatar = $derived(listing?.artisan_image_url || devAvatar);

  $effect(() => {
    const id = listingId;
    void (async () => {
      loading = true;
      activeMediaIndex = 0;
      try {
        listing = await getListingSummary(id);
      } catch {
        listing = undefined;
      } finally {
        loading = false;
      }
    })();
  });

  const title = $derived(
    listing?.translations?.find((tr) => tr.language === locale.code)?.title ??
      listing?.translations?.[0]?.title ??
      '',
  );
  const description = $derived(
    listing?.translations?.find((tr) => tr.language === locale.code)?.description ??
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
  <title>{title ? `${title} — ${t('app.name')}` : t('app.name')}</title>
  {#if listing}
    {@html `<script type="application/ld+json">${jsonLd}<\/script>`}
  {/if}
</svelte:head>

{#if loading}
  <Skeleton shape="card" height="28rem" />
{:else if !listing}
  <EmptyState illustration="empty-error" heading={t('listing.notFound')} />
{:else}
  <div class="listing">
    <div class="listing__gallery">
      <div class="listing__stage">
        {#if activeMedia?.kind === 'VIDEO'}
          <video src={activeMedia.url} controls playsinline class="listing__stage-media">
            <track kind="captions" />
          </video>
        {:else if activeMedia?.url}
          <img src={activeMedia.url} alt={title} class="listing__stage-media" />
        {/if}
      </div>
      {#if media.length > 1}
        <div class="listing__thumbs" role="tablist" aria-label={t('listing.gallery.processVideo')}>
          {#each media as item, index (index)}
            <button
              type="button"
              role="tab"
              aria-selected={index === activeMediaIndex}
              class="listing__thumb"
              class:listing__thumb--active={index === activeMediaIndex}
              onclick={() => (activeMediaIndex = index)}
            >
              {#if item.kind === 'VIDEO'}
                <span class="listing__thumb-video"><Icon name="process-video" title={t('listing.gallery.processVideo')} /></span>
              {:else if item.url}
                <img src={item.url} alt="" />
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <div class="listing__info">
      <p class="listing__craft">
        {#if craftTerm}<CraftTerm term={craftTerm} />{:else}{listing.craft_name}{/if}
        {#if listing.gi_certified}
          <span class="listing__gi-badge"><Icon name="gi-tagged" />{t('listing.giBadge')}</span>
        {/if}
      </p>
      <h1>{title}</h1>
      {#if listing.artisan_name}
        <a href="/artisan/{encodeURIComponent(listing.artisan_name.toLowerCase())}" class="listing__artisan-badge" title="View artisan profile">
          <div class="listing__artisan-avatar">
            {#if artisanAvatar}
              <img src={artisanAvatar} alt={listing.artisan_name} class="listing__artisan-avatar-img" />
            {:else}
              <span class="listing__artisan-avatar-initial">{listing.artisan_name.charAt(0).toUpperCase()}</span>
            {/if}
            <span class="listing__artisan-verified" title="Govt & AI Verified Artisan">
              <Icon name="verified-artisan" />
            </span>
          </div>
          <div class="listing__artisan-info">
            <span class="listing__artisan-name">{t('listing.by', { name: listing.artisan_name })}</span>
            <span class="listing__artisan-sub">
              {#if listing.artisan_district}<span>{listing.artisan_district}</span> • {/if}
              <span class="listing__artisan-view">View artisan storefront →</span>
            </span>
          </div>
        </a>
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
    border-color: var(--k-accent-primary-border, #c45b37);
  }

  .listing__artisan-avatar {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-terracotta-700, #96381e);
    color: #fff;
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
    border: 1.5px solid var(--k-surface-base, #fff);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-neem-600, #2e7d32);
    color: #fff;
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
    color: var(--k-accent-primary-text, #96381e);
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
</style>
