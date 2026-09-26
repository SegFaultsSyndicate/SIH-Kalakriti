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
  import ProductDetailSections from '$lib/ProductDetailSections.svelte';
  import { wishlist } from '$lib/wishlist.svelte';
  import { demoListingById, demoSocial, productDetails, relatedListings } from '$lib/demo-catalog';
  import { ARTISAN_CRAFT_CATEGORIES } from '$lib/craft-categories';

  type ListingSummary = components['schemas']['ListingSummary'];

  const t = $derived(locale.t);
  const listingId = $derived(page.params.slug ?? '');

  let loading = $state(true);
  let fetched = $state<ListingSummary | undefined>(undefined);
  // Demo ids (home/GI/search fallbacks) never exist in the backend -- resolve
  // them locally. $derived, not assigned in the effect, so a language switch
  // retranslates the demo piece like any other UI text.
  const demo = $derived(fetched ? undefined : demoListingById(listingId, t));
  const listing = $derived(fetched ?? demo);
  const social = $derived(demo ? demoSocial(demo, t) : undefined);
  const details = $derived(listing ? productDetails(listing, t) : undefined);
  const related = $derived(listing ? relatedListings(listing, t) : undefined);
  const discountPct = $derived(
    social && listing?.price?.amount_paise
      ? Math.round((1 - listing.price.amount_paise / social.mrpPaise) * 100)
      : 0,
  );
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
      fetched = undefined;
      try {
        fetched = await getListingSummary(id);
        await setCached(listingCacheKey(id), fetched);
      } catch {
        fetched = await getCached<ListingSummary>(listingCacheKey(id));
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
  // Only show the listing's own media/image. Do not borrow other products' photos
  // into the gallery thumbnails, as single-photo artisan pieces should only display their own photo.
  const media = $derived.by(() => {
    if (listing?.media?.length) return listing.media;
    if (!listing?.image_url) return [];
    return [{ kind: 'IMAGE' as const, url: listing.image_url }];
  });
  const activeMedia = $derived(media[activeMediaIndex]);

  // Product zoom modal + carousel: click-to-expand full-screen overlay with
  // hover-to-zoom on the stage image and prev/next paging through `media`,
  // plus swipe gestures on touch devices. See the CSS block near
  // .listing__stage-trigger for the hover-scale mechanics.
  let zoomOpen = $state(false);
  let lensStyle = $state('');
  let zoomDialogEl: HTMLDialogElement | undefined = $state();
  let zoomPreviouslyFocused: HTMLElement | null = null;
  let zoomPreviousOverflow = '';

  function openZoom(): void {
    if (!activeMedia?.url) return;
    zoomOpen = true;
  }

  function closeZoom(): void {
    zoomOpen = false;
  }

  function showNext(): void {
    if (media.length === 0) return;
    activeMediaIndex = (activeMediaIndex + 1) % media.length;
  }

  function showPrev(): void {
    if (media.length === 0) return;
    activeMediaIndex = (activeMediaIndex - 1 + media.length) % media.length;
  }

  function onStageMouseMove(event: MouseEvent): void {
    if (activeMedia?.kind === 'VIDEO') return;
    const target = event.currentTarget as HTMLElement;
    const rect = target.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width) * 100;
    const y = ((event.clientY - rect.top) / rect.height) * 100;
    lensStyle = `--zx: ${x}%; --zy: ${y}%;`;
  }

  function onStageMouseLeave(): void {
    lensStyle = '';
  }

  let touchStartX = 0;
  function onGalleryTouchStart(event: TouchEvent): void {
    touchStartX = event.touches[0]?.clientX ?? 0;
  }

  function onGalleryTouchEnd(event: TouchEvent): void {
    const endX = event.changedTouches[0]?.clientX ?? touchStartX;
    const delta = endX - touchStartX;
    if (Math.abs(delta) > 40) {
      if (delta < 0) showNext();
      else showPrev();
    }
  }

  function onGalleryKeydown(event: KeyboardEvent): void {
    if (!zoomOpen) return;
    if (event.key === 'ArrowRight') showNext();
    else if (event.key === 'ArrowLeft') showPrev();
  }

  function onZoomDialogClose(): void {
    zoomOpen = false;
    document.documentElement.style.overflow = zoomPreviousOverflow;
    zoomPreviouslyFocused?.focus();
  }

  function onZoomBackdropClick(event: MouseEvent): void {
    if (event.target === zoomDialogEl) closeZoom();
  }

  $effect(() => {
    if (!zoomDialogEl) return;
    if (zoomOpen && !zoomDialogEl.open) {
      zoomPreviouslyFocused = document.activeElement as HTMLElement | null;
      zoomPreviousOverflow = document.documentElement.style.overflow;
      document.documentElement.style.overflow = 'hidden';
      zoomDialogEl.showModal();
    } else if (!zoomOpen && zoomDialogEl.open) {
      zoomDialogEl.close();
    }
  });
  const madeToOrder = $derived(listing?.type === 'MADE_TO_ORDER');
  const terms = $derived(listing?.made_to_order_terms);
  const provenance = $derived(listing?.provenance);
  // Matches the listing's craft against the 12 national craft categories to
  // surface a one-line "did you know" fact -- undefined (no box shown) if
  // the listing's craft name doesn't resolve to one of them.
  const craftFunFact = $derived.by(() => {
    const name = listing?.craft_name?.trim().toLowerCase();
    if (!name) return undefined;
    const match = ARTISAN_CRAFT_CATEGORIES.find(
      (c) => name === c.name.toLowerCase() || name.includes(c.id) || c.name.toLowerCase().includes(name),
    );
    return match ? t(match.funFactKey) : undefined;
  });

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
      // Demo pieces have no craft ontology record, so /craft/{slug} would 404.
      href: demo
        ? listingId.startsWith('gi-')
          ? '/gi-tagged'
          : `/search?q=${encodeURIComponent(listing?.craft_name ?? '')}`
        : listing?.craft_slug
          ? `/craft/${listing.craft_slug}`
          : '/catalog',
    },
    { label: title || 'Listing' },
  ]);

  // ponytail: delivery ETA is a flat lead-time estimate, not a courier quote --
  // swap for a real serviceability API once one exists.
  const PINCODE_KEY = 'kalakriti.buyer.pincode';
  const PINCODE_RE = /^[1-9]\d{5}$/;
  let pincode = $state('');
  let pincodeDraft = $state('');
  let pincodeError = $state('');
  const pincodeOk = $derived(PINCODE_RE.test(pincode));

  $effect(() => {
    try {
      pincode = localStorage.getItem(PINCODE_KEY) ?? '';
      pincodeDraft = pincode;
    } catch {}
  });

  function savePincode(): void {
    const v = pincodeDraft.trim();
    if (!PINCODE_RE.test(v)) {
      pincodeError = t('pdp.delivery.invalid');
      return;
    }
    pincodeError = '';
    pincode = v;
    try {
      localStorage.setItem(PINCODE_KEY, v);
    } catch {}
  }

  const deliveryLabel = $derived.by(() => {
    const days = madeToOrder ? (terms?.lead_time_days ?? 21) + 5 : 4 + (Number(pincode.slice(0, 1) || 0) % 3);
    const d = new Date();
    d.setDate(d.getDate() + days);
    return d.toLocaleDateString(locale.code, { weekday: 'short', day: 'numeric', month: 'short' });
  });

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
          showToast({ message: t('listing.share.copied'), variant: 'success' });
        })
        .catch(() => {
          showToast({ message: t('listing.share.copyFailed'), variant: 'error' });
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

<svelte:window onkeydown={onGalleryKeydown} />

{#if loading}
  <div class="listing" aria-hidden="true">
    <div class="listing__gallery">
      <div class="listing__stage">
        <Skeleton width="100%" height="100%" radius="0" />
      </div>
      <div class="listing__thumbs">
        {#each Array(4) as _, i (i)}
          <div class="listing__thumb"><Skeleton width="100%" height="100%" radius="0" /></div>
        {/each}
      </div>
    </div>
    <div class="listing__info listing__info--skeleton">
      <Skeleton shape="text" width="35%" height="0.9rem" />
      <Skeleton shape="text" width="80%" height="1.75rem" />
      <Skeleton shape="text" width="30%" height="1.5rem" />
      <Skeleton shape="text" width="100%" height="0.9rem" />
      <Skeleton shape="text" width="90%" height="0.9rem" />
      <Skeleton shape="text" width="60%" height="0.9rem" />
      <Skeleton width="100%" height="3rem" radius="var(--k-radius-md)" />
    </div>
  </div>
{:else if !listing}
  <EmptyState illustration="empty-error" heading={t('listing.notFound')} />
{:else}
  <div class="listing-page-wrap">
    <div class="listing-breadcrumbs-bar">
      <Breadcrumbs items={breadcrumbItems} />
    </div>

    <div class="listing">
      <div class="listing__gallery">
        <div
          class="listing__stage"
          onmousemove={onStageMouseMove}
          onmouseleave={onStageMouseLeave}
          ontouchstart={onGalleryTouchStart}
          ontouchend={onGalleryTouchEnd}
        >
          {#if activeMedia?.kind === 'VIDEO'}
            <video src={activeMedia.url} controls playsinline class="listing__stage-media">
              <track kind="captions" />
            </video>
          {:else if activeMedia?.url}
            <button type="button" class="listing__stage-trigger" onclick={openZoom} aria-label={t('pdp.gallery.zoom')}>
              <img
                src={activeMedia.url}
                alt={title ? `${title} — Authentic craft photo` : 'Craft piece view'}
                class="listing__stage-media"
                style={lensStyle}
              />
              <span class="listing__zoom-hint"><Icon name="search" size="0.85rem" />{t('pdp.gallery.zoom')}</span>
            </button>
          {/if}

          {#if media.length > 1}
            <Tooltip text={tooltip('tooltip.selectImage')}>
              {#snippet trigger(tp)}
                <button type="button" class="listing__stage-nav listing__stage-nav--prev" onclick={showPrev} aria-label={t('pdp.gallery.prev')} {...tp}>
                  <Icon name="chevron-left" />
                </button>
              {/snippet}
            </Tooltip>
            <Tooltip text={tooltip('tooltip.selectImage')}>
              {#snippet trigger(tp)}
                <button type="button" class="listing__stage-nav listing__stage-nav--next" onclick={showNext} aria-label={t('pdp.gallery.next')} {...tp}>
                  <Icon name="chevron-right" />
                </button>
              {/snippet}
            </Tooltip>
            <span class="listing__stage-counter">{t('pdp.gallery.counter', { current: String(activeMediaIndex + 1), total: String(media.length) })}</span>
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

      {#if social}
        <a class="pdp-rating" href="#reviews">
          <span class="pdp-rating__value">{social.rating.toFixed(1)}</span>
          <span class="pdp-stars" style="--pct: {(social.rating / 5) * 100}%" role="img" aria-label={t('pdp.starsAria', { rating: social.rating })}></span>
          <span class="pdp-rating__count">{t('pdp.ratingsCount', { count: social.ratingCount.toLocaleString(locale.code) })}</span>
        </a>
      {/if}

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
      <div class="pdp-price">
        {#if discountPct > 0}<span class="pdp-price__off">-{discountPct}%</span>{/if}
        <p class="listing__price"><Money paise={listing.price?.amount_paise ?? 0} /></p>
      </div>
      {#if social && discountPct > 0}
        <p class="pdp-mrp">{t('pdp.mrp')} <s><Money paise={social.mrpPaise} /></s></p>
      {/if}
      <p class="pdp-tax">{t('pdp.inclTax')}</p>

      <ul class="pdp-trust" aria-label={t('pdp.trust.aria')}>
        <li><Icon name="fair-price" /><span>{t('pdp.trust.direct')}</span></li>
        <li><Icon name="ready-stock" /><span>{t('pdp.trust.freeDelivery')}</span></li>
        {#if provenance}<li><Icon name="provenance" /><span>{t('pdp.trust.sealed')}</span></li>{/if}
        {#if !madeToOrder}<li><Icon name="verified-artisan" /><span>{t('pdp.trust.returns')}</span></li>{/if}
        {#if listing.gi_certified}<li><Icon name="gi-tagged" /><span>{t('pdp.trust.gi')}</span></li>{/if}
      </ul>

      <div class="pdp-sustainability">
        <ul class="pdp-sustainability__badges" aria-label={t('pdp.sustainability.heading')}>
          <li><Icon name="handmade-certified" size="1rem" /><span>{t('pdp.sustainability.natural')}</span></li>
          <li><Icon name="verified-artisan" size="1rem" /><span>{t('pdp.sustainability.artisanMade')}</span></li>
          <li><Icon name="shg" size="1rem" /><span>{t('pdp.sustainability.lowWaste')}</span></li>
        </ul>
        {#if craftFunFact}
          <p class="pdp-trivia"><Icon name="info" size="0.95rem" /><strong>{t('pdp.trivia.heading')}</strong> {craftFunFact}</p>
        {/if}
      </div>

      {#if details}
        <section class="pdp-about" aria-labelledby="about-heading">
          <h2 id="about-heading">{t('pdp.about.heading')}</h2>
          <ul>
            {#each details.bullets as b (b)}<li>{b}</li>{/each}
          </ul>
        </section>
      {/if}

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

      <div class="pdp-delivery">
        <p class="pdp-delivery__eta">
          <Icon name="ready-stock" />
          <span>{pincodeOk ? t('pdp.delivery.etaTo', { date: deliveryLabel, pincode }) : t('pdp.delivery.eta', { date: deliveryLabel })}</span>
        </p>
        <form class="pdp-delivery__form" onsubmit={(e) => { e.preventDefault(); savePincode(); }}>
          <label class="visually-hidden" for="pdp-pincode">{t('pdp.delivery.pincodeLabel')}</label>
          <input id="pdp-pincode" inputmode="numeric" maxlength="6" placeholder={t('pdp.delivery.pincodePlaceholder')} bind:value={pincodeDraft} />
          <button type="submit">{t('pdp.delivery.check')}</button>
        </form>
        {#if pincodeError}<p class="pdp-delivery__error" role="alert">{pincodeError}</p>{/if}
        {#if social?.stockLeft !== undefined}
          <p class="pdp-stock" class:pdp-stock--low={social.stockLeft <= 3}>
            {social.stockLeft <= 3 ? t('pdp.stock.low', { count: social.stockLeft }) : t('pdp.stock.in')}
          </p>
        {/if}
        <p class="pdp-seller">
          {t('pdp.soldBy')}
          <a href="/artisan/{encodeURIComponent((listing.artisan_name ?? '').toLowerCase())}">{listing.artisan_name ?? t('pdp.theArtisan')}</a>
        </p>
      </div>

      <button
        type="button"
        class="pdp-wishlist"
        aria-pressed={wishlist.has(listing.id ?? '')}
        onclick={() => wishlist.toggle(listing.id ?? '', title)}
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill={wishlist.has(listing.id ?? '') ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
        </svg>
        <span>{wishlist.has(listing.id ?? '') ? t('listingCard.savedToWishlist') : t('listingCard.addToWishlist')}</span>
      </button>

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

  {#if details && related}
    <ProductDetailSections {listing} {details} {social} {related} />
  {/if}

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

  <!-- Product zoom modal: full-screen overlay + carousel paging -->
  <dialog
    bind:this={zoomDialogEl}
    class="listing__zoom-dialog"
    aria-label={title || t('pdp.gallery.zoom')}
    onclose={onZoomDialogClose}
    onclick={onZoomBackdropClick}
    ontouchstart={onGalleryTouchStart}
    ontouchend={onGalleryTouchEnd}
  >
    {#if activeMedia}
      <button type="button" class="listing__zoom-close" onclick={closeZoom} aria-label={t('ui.dialog.close')}>
        <Icon name="close" size="1.4rem" />
      </button>
      {#if media.length > 1}
        <button type="button" class="listing__zoom-nav listing__zoom-nav--prev" onclick={showPrev} aria-label={t('pdp.gallery.prev')}>
          <Icon name="chevron-left" size="1.6rem" />
        </button>
        <button type="button" class="listing__zoom-nav listing__zoom-nav--next" onclick={showNext} aria-label={t('pdp.gallery.next')}>
          <Icon name="chevron-right" size="1.6rem" />
        </button>
      {/if}
      <div class="listing__zoom-stage">
        {#if activeMedia.kind === 'VIDEO'}
          <video src={activeMedia.url} controls playsinline class="listing__zoom-media">
            <track kind="captions" />
          </video>
        {:else if activeMedia.url}
          <img src={activeMedia.url} alt={title ? `${title} — Authentic craft photo` : 'Craft piece view'} class="listing__zoom-media" />
        {/if}
      </div>
      {#if media.length > 1}
        <p class="listing__zoom-counter">{t('pdp.gallery.counter', { current: String(activeMediaIndex + 1), total: String(media.length) })}</p>
      {/if}
    {/if}
  </dialog>
</div>
{/if}

<style>
  .listing {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-6);
  }

  .listing__info--skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .listing__stage {
    position: relative;
    aspect-ratio: 4/5;
    background-color: var(--k-surface-sunken);
    border-radius: var(--k-radius-lg);
    overflow: hidden;
  }

  .listing__stage-trigger {
    position: absolute;
    inset: 0;
    inline-size: 100%;
    block-size: 100%;
    padding: 0;
    border: none;
    background: none;
    cursor: zoom-in;
    overflow: hidden;
  }

  .listing__stage-media {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
    transition: transform 0.2s ease;
  }

  @media (hover: hover) {
    .listing__stage-trigger:hover .listing__stage-media,
    .listing__stage-trigger:focus-visible .listing__stage-media {
      transform: scale(1.8);
      transform-origin: var(--zx, 50%) var(--zy, 50%);
    }
  }

  .listing__zoom-hint {
    position: absolute;
    inset-block-end: 0.6rem;
    inset-inline-end: 0.6rem;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.3rem 0.6rem;
    border-radius: var(--k-radius-pill);
    background: rgba(0, 0, 0, 0.55);
    color: #fff;
    font-size: var(--k-text-xs);
    pointer-events: none;
  }

  .listing__stage-nav {
    position: absolute;
    inset-block-start: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.25rem;
    block-size: 2.25rem;
    border-radius: var(--k-radius-pill);
    border: none;
    background: rgba(255, 255, 255, 0.85);
    color: var(--k-text-primary);
    cursor: pointer;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
  }

  .listing__stage-nav--prev {
    inset-inline-start: 0.6rem;
  }

  .listing__stage-nav--next {
    inset-inline-end: 0.6rem;
  }

  .listing__stage-counter {
    position: absolute;
    inset-block-start: 0.6rem;
    inset-inline-end: 0.6rem;
    padding: 0.2rem 0.55rem;
    border-radius: var(--k-radius-pill);
    background: rgba(0, 0, 0, 0.55);
    color: #fff;
    font-size: var(--k-text-xs);
  }

  /* Full-screen zoom modal */
  .listing__zoom-dialog {
    position: fixed;
    inset: 0;
    inline-size: 100vw;
    block-size: 100vh;
    max-inline-size: 100vw;
    max-block-size: 100vh;
    margin: 0;
    padding: 0;
    border: none;
    background: transparent;
  }

  .listing__zoom-dialog::backdrop {
    background: rgba(10, 8, 6, 0.86);
  }

  .listing__zoom-dialog[open] {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .listing__zoom-stage {
    max-inline-size: min(92vw, 60rem);
    max-block-size: 88vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .listing__zoom-media {
    max-inline-size: 100%;
    max-block-size: 88vh;
    object-fit: contain;
    border-radius: var(--k-radius-md);
  }

  .listing__zoom-close {
    position: fixed;
    inset-block-start: 1rem;
    inset-inline-end: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border-radius: var(--k-radius-pill);
    border: none;
    background: rgba(255, 255, 255, 0.92);
    color: var(--k-text-primary);
    cursor: pointer;
    z-index: 1;
  }

  .listing__zoom-nav {
    position: fixed;
    inset-block-start: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 3rem;
    block-size: 3rem;
    border-radius: var(--k-radius-pill);
    border: none;
    background: rgba(255, 255, 255, 0.92);
    color: var(--k-text-primary);
    cursor: pointer;
    z-index: 1;
  }

  .listing__zoom-nav--prev {
    inset-inline-start: 1rem;
  }

  .listing__zoom-nav--next {
    inset-inline-end: 1rem;
  }

  .listing__zoom-counter {
    position: fixed;
    inset-block-end: 1.25rem;
    inset-inline: 0;
    text-align: center;
    color: #fff;
    font-size: var(--k-text-sm);
    margin: 0;
  }

  @media (max-width: 30rem) {
    .listing__zoom-nav {
      inline-size: 2.5rem;
      block-size: 2.5rem;
    }
  }

  /* Sustainability badges + craft trivia */
  .pdp-sustainability {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-4);
  }

  .pdp-sustainability__badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .pdp-sustainability__badges li {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: 0.3rem 0.6rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-khadi-100, var(--k-surface-sunken));
    border: var(--k-hairline) solid var(--k-border-hairline);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .pdp-sustainability__badges :global(svg) {
    color: var(--k-neem-600, var(--k-accent-success-muted));
  }

  .pdp-trivia {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
    margin: 0;
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    line-height: 1.5;
  }

  .pdp-trivia :global(svg) {
    flex: none;
    margin-block-start: 0.15rem;
    color: var(--k-accent-primary-text);
  }

  .pdp-trivia strong {
    color: var(--k-text-primary);
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
    margin: 0;
  }

  /* Amazon-style buy box pieces */
  .pdp-rating {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-3);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    text-decoration: none;
  }

  .pdp-rating:hover .pdp-rating__count {
    text-decoration: underline;
  }

  .pdp-rating__value {
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .pdp-price {
    display: flex;
    align-items: baseline;
    gap: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-3);
  }

  .pdp-price__off {
    font-size: var(--k-text-xl);
    color: var(--k-accent-danger-muted, #b12704);
  }

  .pdp-mrp,
  .pdp-tax {
    margin: var(--k-space-1) 0 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .pdp-trust {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(5.5rem, 1fr));
    gap: var(--k-space-3);
    list-style: none;
    padding: var(--k-space-4) 0;
    margin: var(--k-space-3) 0;
    border-block: var(--k-hairline) solid var(--k-border-hairline);
    text-align: center;
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .pdp-trust li {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-1);
  }

  .pdp-trust :global(svg) {
    inline-size: 1.5rem;
    block-size: 1.5rem;
    color: var(--k-accent-primary-text);
  }

  .pdp-about h2 {
    font-size: var(--k-text-md);
    margin: 0 0 var(--k-space-2);
  }

  .pdp-about ul {
    margin: 0 0 var(--k-space-4);
    padding-inline-start: 1.1rem;
    display: grid;
    gap: var(--k-space-1);
    font-size: var(--k-text-sm);
    line-height: 1.5;
  }

  .pdp-delivery {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3) var(--k-space-4);
    margin-block: var(--k-space-3);
    font-size: var(--k-text-sm);
  }

  .pdp-delivery p {
    margin: 0 0 var(--k-space-2);
  }

  .pdp-delivery__eta {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .pdp-delivery__form {
    display: flex;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-2);
  }

  .pdp-delivery__form input {
    inline-size: 9rem;
    padding: var(--k-space-1) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    font: inherit;
    background: var(--k-surface-base);
    color: var(--k-text-primary);
  }

  .pdp-delivery__form button,
  .pdp-wishlist {
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
    cursor: pointer;
  }

  .pdp-delivery__error {
    color: var(--k-accent-danger-muted, #b12704);
  }

  .pdp-stock {
    font-weight: var(--k-weight-semibold);
    color: var(--k-accent-success-muted, #007600);
  }

  .pdp-stock--low {
    color: var(--k-accent-danger-muted, #b12704);
  }

  .pdp-seller {
    color: var(--k-text-secondary);
  }

  .pdp-wishlist {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-4);
    font-size: var(--k-text-sm);
  }

  .pdp-wishlist[aria-pressed='true'] {
    color: #e11d48;
    border-color: #e11d48;
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
