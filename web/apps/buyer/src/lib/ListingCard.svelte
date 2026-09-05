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
  import { Money } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import type { components } from '@kalakriti/api';
  import { craftIcon } from './craft-icon';

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
</script>

<a class="listing-card" {href}>
  <span class="listing-card__media">
    {#if listing.image_url}
      <img src={listing.image_url} alt="" loading="lazy" width="400" height="400" />
    {:else}
      <span class="listing-card__placeholder" aria-hidden="true">
        <Icon name={craftIcon(listing.craft_slug ?? listing.craft_name ?? '')} />
      </span>
    {/if}
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
    {#if listing.artisan_name}<span class="listing-card__artisan">{listing.artisan_name}</span>{/if}
    <span class="listing-card__price"><Money paise={listing.price?.amount_paise ?? 0} /></span>
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
  }

  .listing-card__price {
    margin-block-start: var(--k-space-1);
    font-weight: var(--k-weight-semibold);
  }

  .listing-card__cross-lingual {
    margin-block-start: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-accent-secondary);
  }
</style>
