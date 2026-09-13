<!--
  apps/buyer/src/lib/ListingCard.svelte

  The product-forward card used by every listing grid on the buyer app:
  home sections, search results, craft pages. The photograph carries the
  card; chrome (craft name, price, badges) sits below it in one line each.
  made-to-order and ready-stock render as equal-weight badges -- neither is
  a downgrade of the other, per the batch 11 brief's "do not demote
  made-to-order" rule: same badge size, same position, no greyed styling.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Money, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import type { components } from '@kalakriti/api';
  import { craftIcon } from './craft-icon';
  import { wishlist } from './wishlist.svelte';

  type ListingSummary = components['schemas']['ListingSummary'];

  interface Props {
    listing: ListingSummary;
    href: string;
    crossLingual?: boolean;
  }

  let { listing, href, crossLingual = false }: Props = $props();

  const t = $derived(locale.t);
  const title = $derived(
    listing.translations?.find((tr) => tr.language === locale.code)?.title ??
      listing.translations?.[0]?.title ??
      '',
  );
  const madeToOrder = $derived(listing.type === 'MADE_TO_ORDER');
  const paise = $derived(listing.price?.amount_paise ?? 0);

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

  // The OpenAPI spec carries no artisan image on a listing summary, so this is
  // the locally-captured avatar only. Do not reintroduce an invented field.
  const artisanAvatar = $derived(devAvatar);

  let copied = $state(false);

  function handleShare(e: MouseEvent): void {
    e.preventDefault();
    e.stopPropagation();
    const fullUrl = typeof window !== 'undefined' ? `${window.location.origin}${href}` : href;
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard
        .writeText(fullUrl)
        .then(() => {
          copied = true;
          setTimeout(() => {
            copied = false;
          }, 2000);
          showToast({ message: 'Craft listing link copied to clipboard!', variant: 'success' });
        })
        .catch(() => {
          showToast({ message: 'Unable to copy link', variant: 'error' });
        });
    }
  }

  function handleWishlist(e: MouseEvent): void {
    e.preventDefault();
    e.stopPropagation();
    if (listing.id) {
      wishlist.toggle(listing.id, title || listing.craft_name);
    }
  }
</script>

<a class="listing-card" {href}>
  <span class="listing-card__media">
    {#if listing.image_url}
      <img
        src={listing.image_url}
        alt={title ? `${title} — Authentic handcrafted ${listing.craft_name ?? 'artisan piece'}` : 'Handcrafted artisan listing'}
        loading="lazy"
        width="400"
        height="400"
      />
    {:else}
      <span class="listing-card__placeholder" aria-hidden="true">
        <Icon name={craftIcon(listing.craft_slug ?? listing.craft_name ?? '')} />
      </span>
    {/if}

    {#if listing.gi_certified}
      <span class="listing-card__gi-pin" title="Registered Geographical Indication (GI) Certified">
        <svg width="22" height="26" viewBox="0 0 24 30" fill="none">
          <path d="M12 0C5.37 0 0 5.37 0 12c0 9 12 18 12 18s12-9 12-18c0-6.63-5.37-12-12-12z" fill="#ffffff" stroke="#c2b6a5" stroke-width="1"/>
          <circle cx="12" cy="11" r="7.5" fill="#fcfbf7"/>
          <path d="M7 8.5C8.5 7 15.5 7 17 8.5" stroke="#ff9933" stroke-width="2" stroke-linecap="round"/>
          <path d="M7 11C8.5 10 15.5 10 17 11" stroke="#dedede" stroke-width="2" stroke-linecap="round"/>
          <path d="M7 13.5C8.5 15 15.5 15 17 13.5" stroke="#138808" stroke-width="2" stroke-linecap="round"/>
          <circle cx="12" cy="11" r="1.5" fill="#000080"/>
        </svg>
      </span>
    {/if}

    <div class="listing-card__media-actions">
      <button
        type="button"
        class="listing-card__wishlist-btn"
        class:is-wishlisted={wishlist.has(listing.id ?? '')}
        onclick={handleWishlist}
        title={wishlist.has(listing.id ?? '') ? 'Saved to Wishlist' : 'Add to Wishlist'}
        aria-label="Wishlist"
      >
        <svg viewBox="0 0 24 24" width="15" height="15" fill={wishlist.has(listing.id ?? '') ? '#e11d48' : 'none'} stroke={wishlist.has(listing.id ?? '') ? '#e11d48' : '#ffffff'} stroke-width="2">
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
        </svg>
      </button>

      <button
        type="button"
        class="listing-card__share-btn"
        onclick={handleShare}
        title={copied ? 'Link copied!' : 'Share artisan piece'}
        aria-label={copied ? 'Link copied to clipboard' : 'Share artisan piece'}
      >
        <Icon name={copied ? 'check' : 'share'} size="0.9rem" />
      </button>
    </div>

    <span class="listing-card__badges">
      {#if listing.gi_certified}
        <span class="listing-card__badge"><Icon name="gi-tagged" />{t('search.giCertified.badge')}</span>
      {/if}
      <span class="listing-card__badge">
        <Icon name={madeToOrder ? 'made-to-order' : 'ready-stock'} />
        {madeToOrder ? t('search.madeToOrder.badge') : t('search.readyStock.badge')}
      </span>
    </span>
  </span>
  <span class="listing-card__body">
    {#if listing.craft_name}<span class="listing-card__craft">{listing.craft_name}</span>{/if}
    <span class="listing-card__title">{title}</span>
    {#if listing.artisan_name}
      <span class="listing-card__artisan">
        <span class="listing-card__artisan-badge">
          {#if artisanAvatar}
            <img
              src={artisanAvatar}
              alt={listing.artisan_name ? `Portrait of master craftsperson ${listing.artisan_name}` : 'Artisan portrait'}
              class="listing-card__artisan-avatar"
            />
          {:else}
            <span class="listing-card__artisan-initial">{listing.artisan_name.charAt(0).toUpperCase()}</span>
          {/if}
          <span class="listing-card__artisan-name">{listing.artisan_name}</span>
          <span class="listing-card__artisan-verified" title="Verified Artisan">
            <Icon name="verified-artisan" />
          </span>
        </span>
      </span>
    {/if}
    <span class="listing-card__price-row">
      <span class="listing-card__price"><Money {paise} /></span>
      {#if paise > 0}
        <span class="listing-card__mrp">₹{Math.round((paise * 1.25) / 100).toLocaleString('en-IN')}</span>
        <span class="listing-card__discount">(20% OFF)</span>
      {/if}
    </span>

    <!-- Action row: Order & Wishlist -->
    <span class="listing-card__action-row">
      <span class="listing-card__order-btn">
        <span>Order</span>
        <Icon name="arrow-right" size="0.75rem" />
      </span>
      <button
        type="button"
        class="listing-card__wishlist-cta"
        class:is-wishlisted={wishlist.has(listing.id ?? '')}
        onclick={handleWishlist}
        title={wishlist.has(listing.id ?? '') ? 'Remove from Wishlist' : 'Save to Wishlist'}
      >
        <svg viewBox="0 0 24 24" width="13" height="13" fill={wishlist.has(listing.id ?? '') ? '#e11d48' : 'none'} stroke={wishlist.has(listing.id ?? '') ? '#e11d48' : 'currentColor'} stroke-width="2">
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
        </svg>
        <span>{wishlist.has(listing.id ?? '') ? 'Saved' : 'Wishlist'}</span>
      </button>
    </span>

    {#if crossLingual}
      <span class="listing-card__cross-lingual">{t('search.crossLingual.badge')}</span>
    {/if}
  </span>
</a>

<style>
  .listing-card {
    display: flex;
    flex-direction: column;
    color: inherit;
    text-decoration: none;
  }

  .listing-card__media {
    position: relative;
    display: block;
    aspect-ratio: 4 / 5;
    background-color: var(--k-surface-sunken);
    border-radius: var(--k-radius-md);
    overflow: hidden;
  }

  .listing-card__media img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .listing-card__placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 100%;
    block-size: 100%;
    color: var(--k-text-secondary);
  }

  .listing-card__placeholder :global(svg) {
    inline-size: 2.5rem;
    block-size: 2.5rem;
  }

  .listing-card__badges {
    position: absolute;
    inset-block-start: var(--k-space-2);
    inset-inline-start: var(--k-space-2);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    pointer-events: none;
  }

  .listing-card__media-actions {
    position: absolute;
    inset-block-start: var(--k-space-2, 0.5rem);
    inset-inline-end: var(--k-space-2, 0.5rem);
    display: flex;
    align-items: center;
    gap: 0.35rem;
    z-index: 2;
  }

  .listing-card__share-btn,
  .listing-card__wishlist-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 1.95rem;
    block-size: 1.95rem;
    border-radius: var(--k-radius-full, 999px);
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    color: var(--k-text-secondary);
    cursor: pointer;
    transition: transform 0.15s ease, background-color 0.15s ease, color 0.15s ease;
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
  }

  .listing-card__share-btn:hover {
    background-color: #ffffff;
    color: var(--k-terracotta-700, #96381e);
    transform: scale(1.08);
  }

  .listing-card__wishlist-btn:hover {
    background-color: #ffffff;
    color: #e11d48;
    transform: scale(1.08);
  }

  .listing-card__wishlist-btn.is-wishlisted {
    background-color: #fff1f2;
    color: #e11d48;
    border-color: #fecdd3;
  }

  .listing-card__action-row {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    margin-block-start: 0.5rem;
  }

  .listing-card__order-btn {
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.3rem;
    padding: 0.35rem 0.6rem;
    border-radius: 4px;
    background-color: #f7f4ee;
    border: 1px solid #ded7cb;
    color: #7b1c1c;
    font-size: 0.72rem;
    font-weight: 700;
    transition: all 0.15s ease;
    text-decoration: none;
  }

  .listing-card:hover .listing-card__order-btn {
    background-color: #7b1c1c;
    color: #ffffff;
    border-color: #7b1c1c;
  }

  .listing-card__wishlist-cta {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.35rem 0.5rem;
    border-radius: 4px;
    border: 1px solid #ded7cb;
    background-color: #ffffff;
    color: #57534e;
    font-size: 0.7rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .listing-card__wishlist-cta:hover,
  .listing-card__wishlist-cta.is-wishlisted {
    border-color: #e11d48;
    color: #e11d48;
    background-color: #fff1f2;
  }

  .listing-card__badge {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: 0.2em var(--k-space-2);
    font-size: var(--k-text-xs);
    background-color: var(--k-surface-raised);
    border-radius: var(--k-radius-sm);
  }

  .listing-card__body {
    display: flex;
    flex-direction: column;
    padding-block-start: var(--k-space-2);
    gap: 0.1em;
  }

  .listing-card__craft {
    font-size: var(--k-text-xs);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
    color: var(--k-text-secondary);
  }

  .listing-card__title {
    font-weight: var(--k-weight-semibold);
  }

  .listing-card__artisan {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-start: 0.1rem;
  }

  .listing-card__artisan-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }

  .listing-card__artisan-avatar {
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 50%;
    object-fit: cover;
    border: 1px solid var(--k-stone-300, #d5cec5);
    flex-shrink: 0;
  }

  .listing-card__artisan-initial {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 50%;
    background-color: var(--k-khadi-200, #eedec8);
    color: var(--k-terracotta-800, #7a2010);
    font-size: 0.65rem;
    font-weight: var(--k-weight-bold, 700);
    flex-shrink: 0;
  }

  .listing-card__artisan-name {
    font-weight: var(--k-weight-medium, 500);
  }

  .listing-card__artisan-verified {
    display: inline-flex;
    align-items: center;
    color: var(--k-indigo-700, #364190);
    font-size: 0.85rem;
    flex-shrink: 0;
  }

  .listing-card__gi-pin {
    position: absolute;
    inset-inline-start: var(--k-space-2, 0.5rem);
    inset-block-start: var(--k-space-2, 0.5rem);
    z-index: 2;
    filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.25));
    transition: transform 0.15s ease;
  }

  .listing-card:hover .listing-card__gi-pin {
    transform: scale(1.1);
  }

  .listing-card__price-row {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 0.35rem;
    margin-block-start: var(--k-space-1);
  }

  .listing-card__price {
    font-weight: var(--k-weight-semibold, 600);
    color: var(--k-text-primary, #1e1915);
  }

  .listing-card__mrp {
    font-size: var(--k-text-xs, 0.75rem);
    color: var(--k-text-secondary, #7a7269);
    text-decoration: line-through;
  }

  .listing-card__discount {
    font-size: var(--k-text-2xs, 0.65rem);
    font-weight: 700;
    color: #2e7d32;
    background-color: #e8f5e9;
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
  }

  .listing-card__cross-lingual {
    margin-block-start: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-accent-secondary);
  }
</style>
