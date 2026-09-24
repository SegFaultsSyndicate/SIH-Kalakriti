<!--
  apps/buyer/src/routes/artisan/[slug]/+page.svelte

  Storefront: portrait, story, district, cluster, awardee status, follow,
  full catalogue, process-video feed. "Awardee status" has no backend field
  anywhere in the repo (grepped identity/catalog protos and SQL) -- not
  rendered, rather than inventing a badge. Follow uses the same
  POST/DELETE /artisans/{id}/follow the artisan app's own follower-count
  batch (10) already wired server-side; this batch only adds the buyer-side
  button.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Button, EmptyState, Skeleton, showToast, BadgeChip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    getArtisanStorefront,
    getFollowerCount,
    listListings,
    batchGetListingSummaries,
    getProcessFeed,
    followArtisan,
    unfollowArtisan,
    listArtisanBadges,
    type components,
  } from '@kalakriti/api';
  import { session } from '@kalakriti/api';
  import ListingCard from '$lib/ListingCard.svelte';
  import ListingCardSkeleton from '$lib/ListingCardSkeleton.svelte';
  import { demoStorefront } from '$lib/demo-catalog';

  type ArtisanStorefront = components['schemas']['ArtisanStorefront'];
  type ListingSummary = components['schemas']['ListingSummary'];
  type ProcessClip = components['schemas']['ProcessClip'];
  type ArtisanBadge = components['schemas']['ArtisanBadge'];

  const t = $derived(locale.t);
  const slug = $derived(page.params.slug ?? '');

  let loading = $state(true);
  let artisan = $state<ArtisanStorefront | undefined>(undefined);
  let followerCount = $state<number | undefined>(undefined);
  let listings = $state<ListingSummary[]>([]);
  let clips = $state<ProcessClip[]>([]);
  let badges = $state<ArtisanBadge[]>([]);
  // Follow state gap: The BFF exposes POST/DELETE /artisans/{id}/follow and
  // GET /artisans/{id}/follower-count, but has no GET /artisans/{id}/is-following
  // or caller-following query endpoint. Consequently, `following` initializes to false
  // on page load and reflects only actions taken during the current page session.
  let following = $state(false);
  let followBusy = $state(false);

  $effect(() => {
    void (async () => {
      loading = true;
      try {
        artisan = await getArtisanStorefront(slug);
        if (artisan?.id) {
          const [count, own, feed, badgeRes] = await Promise.all([
            getFollowerCount(artisan.id).catch(() => undefined),
            listListings({ artisan_id: artisan.id, state: 'PUBLISHED' }),
            getProcessFeed().catch(() => ({ clips: [] })),
            listArtisanBadges(artisan.id).catch(() => ({ artisan_badges: [] })),
          ]);
          followerCount = count?.count;
          badges = badgeRes?.artisan_badges ?? [];
          const ids = (own.listings ?? []).map((l) => l.id!).filter(Boolean);
          const { summaries } = ids.length
            ? await batchGetListingSummaries(ids)
            : { summaries: [] };
          listings = (summaries ?? []) as ListingSummary[];
          clips = (feed.clips ?? []).filter((c) => c.artisan_id === artisan!.id);
        }
      } catch {
        // Demo artisans (home/GI/search fallback pieces) have no backend record.
        const demo = demoStorefront(slug, t);
        artisan = demo?.artisan;
        listings = demo?.listings ?? [];
      } finally {
        loading = false;
      }
    })();
  });

  async function toggleFollow(): Promise<void> {
    if (!artisan?.id || followBusy) return;
    followBusy = true;
    try {
      if (following) {
        await unfollowArtisan(artisan.id);
        following = false;
        if (followerCount !== undefined) followerCount -= 1;
      } else {
        await followArtisan(artisan.id);
        following = true;
        followerCount = (followerCount ?? 0) + 1;
      }
    } catch {
      showToast({ variant: 'error', message: t('api.error.unknown') });
    } finally {
      followBusy = false;
    }
  }
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
  const artisanAvatar = $derived(artisan?.image_url || devAvatar);
  const fairParam = $derived(page.url.searchParams.get('fair'));
  const stallParam = $derived(page.url.searchParams.get('stall'));
</script>

<svelte:head>
  <title>{artisan?.display_name ?? t('search.heading')} — {t('app.name')}</title>
</svelte:head>

{#if loading}
  <div class="storefront-skeleton" aria-hidden="true">
    <header class="storefront-header">
      <Skeleton width="128px" height="128px" radius="50%" />
      <div class="storefront-skeleton__lines">
        <Skeleton shape="text" width="12rem" height="1.5rem" />
        <Skeleton shape="text" width="8rem" height="1rem" />
        <Skeleton shape="text" width="10rem" height="0.9rem" />
      </div>
    </header>
    <div class="storefront-catalog">
      {#each Array(4) as _, i (i)}<ListingCardSkeleton />{/each}
    </div>
  </div>
{:else if !artisan}
  <EmptyState illustration="empty-error" heading={t('artisan.notFound')} />
{:else}
  {#if fairParam || stallParam}
    <aside class="fair-welcome-banner">
      <div class="fair-welcome-banner__icon"><Icon name="calendar" size="1.5rem" /></div>
      <div class="fair-welcome-banner__content">
        <strong>
          {#if stallParam}
            {t('artisanStorefront.visitingStall', { stall: stallParam, fair: fairParam ? fairParam.replace(/-/g, ' ').toUpperCase() : t('artisanStorefront.fairFallback') })}
          {:else}
            {t('artisanStorefront.visitingFrom', { fair: fairParam ? fairParam.replace(/-/g, ' ').toUpperCase() : t('artisanStorefront.fairFallback') })}
          {/if}
          {t('artisanStorefront.welcome')}
        </strong>
        <p>{t('artisanStorefront.reorderDesc')}</p>
      </div>
      <a href="/card/{slug}" class="fair-welcome-banner__card-btn">
        <Icon name="verified-artisan" size="0.85rem" /> {t('artisanStorefront.visitingCard')}
      </a>
    </aside>
  {/if}

  <header class="storefront-header">
    {#if artisanAvatar}
      <img
        class="storefront-header__portrait"
        src={artisanAvatar}
        alt=""
        width="128"
        height="128"
        fetchpriority="high"
      />
    {:else}
      <div class="storefront-header__initial-avatar">
        <span>{(artisan.display_name || 'A').charAt(0).toUpperCase()}</span>
      </div>
    {/if}
    <div>
      <h1>{artisan.display_name}</h1>
      {#if artisan.craft_name}<p class="storefront-header__craft">{artisan.craft_name}</p>{/if}
      {#if artisan.district || artisan.state_code}
        <p class="storefront-header__location">
          <Icon name="location" />
          {[artisan.district, artisan.state_code].filter(Boolean).join(', ')}
        </p>
      {/if}
      {#if artisan.verified}
        <p class="storefront-header__verified"><Icon name="verified-artisan" />{t('artisan.verified')}</p>
      {/if}
      {#if badges.length > 0}
        <div class="storefront-badges">
          {#each badges as grant (grant.badge.id)}
            <BadgeChip badge={grant.badge} granted={true} grantedAt={grant.granted_at} />
          {/each}
        </div>
      {/if}
      {#if followerCount !== undefined}
        <p class="storefront-header__followers">{t('artisan.followerCount', { count: String(followerCount) })}</p>
      {/if}
      {#if session.status === 'authenticated'}
        <Button variant={following ? 'secondary' : 'primary'} onclick={toggleFollow} loading={followBusy} tooltip={tooltip(following ? 'tooltip.unfollow' : 'tooltip.follow')}>
          {following ? t('artisan.following') : t('artisan.follow')}
        </Button>
      {/if}
      {#if artisan.bio}<p class="storefront-header__bio">{artisan.bio}</p>{/if}
    </div>
  </header>

  <section>
    <h2>{t('artisan.catalog.heading')}</h2>
    {#if listings.length === 0}
      <p>{t('artisan.catalog.empty')}</p>
    {:else}
      <div class="storefront-catalog">
        {#each listings as listing (listing.id)}
          <ListingCard {listing} href={`/listing/${listing.id}`} />
        {/each}
      </div>
    {/if}
  </section>

  {#if clips.length > 0}
    <section>
      <h2>{t('artisan.processFeed.heading')}</h2>
      <div class="storefront-clips">
        {#each clips as clip (clip.listing_id)}
          <a class="storefront-clips__item" href={`/listing/${clip.listing_slug}`}>
            {#if clip.video_url}
              <video src={clip.video_url} muted playsinline preload="metadata"></video>
            {/if}
            <span>{clip.title}</span>
          </a>
        {/each}
      </div>
    </section>
  {/if}
{/if}

<style>
  .storefront-header {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-4);
    margin-block-end: var(--k-space-6);
  }

  @media (min-width: 36rem) {
    .storefront-header {
      grid-template-columns: 8rem 1fr;
      gap: var(--k-space-5);
    }
  }

  .storefront-header__portrait {
    inline-size: 100%;
    aspect-ratio: 1;
    object-fit: cover;
    border-radius: 50%;
  }

  .storefront-header__initial-avatar {
    inline-size: 100%;
    aspect-ratio: 1;
    border-radius: 50%;
    background-color: var(--k-terracotta-700, var(--k-accent-primary-bg));
    color: var(--k-text-on-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--k-text-3xl);
    font-weight: var(--k-weight-bold);
  }

  .storefront-header__craft {
    color: var(--k-text-secondary);
  }

  .storefront-header__location,
  .storefront-badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin: var(--k-space-2) 0;
  }

  .storefront-header__verified {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-sm);
  }

  .storefront-header__followers {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .storefront-header__bio {
    margin-block-start: var(--k-space-3);
    max-inline-size: 60ch;
  }

  .storefront-skeleton__lines {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .storefront-catalog {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(11rem, 1fr));
    gap: var(--k-space-5);
    margin-block-start: var(--k-space-4);
  }

  .storefront-clips {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-4);
  }

  .storefront-clips__item {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    color: inherit;
    text-decoration: none;
    font-size: var(--k-text-sm);
  }

  .storefront-clips__item video {
    inline-size: 100%;
    aspect-ratio: 3/4;
    object-fit: cover;
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
  }

  .fair-welcome-banner {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-4);
    background: linear-gradient(135deg, rgba(120, 53, 15, 0.1), rgba(180, 83, 9, 0.05));
    border: 1px solid var(--k-border-warning);
    border-radius: var(--k-radius-md);
    margin-block-end: var(--k-space-4);
  }

  .fair-welcome-banner__icon {
    font-size: 1.8rem;
    flex-shrink: 0;
  }

  .fair-welcome-banner__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .fair-welcome-banner__content strong {
    font-size: var(--k-text-sm);
    color: var(--k-accent-primary-text);
  }

  .fair-welcome-banner__content p {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    margin: 0;
  }

  .fair-welcome-banner__card-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: var(--k-text-xs);
    font-weight: 700;
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-pill);
    text-decoration: none;
    white-space: nowrap;
  }
</style>
