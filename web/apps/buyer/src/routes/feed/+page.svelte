<!--
  apps/buyer/src/routes/feed/+page.svelte

  The process-provenance feed: a vertical feed of craft-in-progress clips,
  each one a listing with its making visible, not a social feed. Every clip
  is real product video (GET /feed/process, derived from published
  listings' own product media -- see client.Catalog.ListProcessClips; there
  is no dedicated feed store). Autoplay is muted and only runs when
  prefers-reduced-motion is not set and the connection is not Save-Data or
  2G -- otherwise a tap plays it, same as any video element.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { EmptyState, SkeletonDetail } from '@kalakriti/ui';
  import { getProcessFeed, type components } from '@kalakriti/api';
  import { shouldConserveData } from '@kalakriti/offline';

  type ProcessClip = components['schemas']['ProcessClip'];

  const t = $derived(locale.t);

  let loading = $state(true);
  let clips = $state<ProcessClip[]>([]);
  let autoplay = $state(false);

  $effect(() => {
    autoplay =
      !window.matchMedia('(prefers-reduced-motion: reduce)').matches && !shouldConserveData();
  });

  $effect(() => {
    void (async () => {
      loading = true;
      try {
        const res = await getProcessFeed();
        clips = res.clips ?? [];
      } finally {
        loading = false;
      }
    })();
  });

  function onIntersect(entries: IntersectionObserverEntry[]): void {
    if (!autoplay) return;
    for (const entry of entries) {
      const video = entry.target as HTMLVideoElement;
      if (entry.isIntersecting) void video.play().catch(() => {});
      else video.pause();
    }
  }

  function observe(node: HTMLVideoElement): { destroy(): void } {
    const observer = new IntersectionObserver(onIntersect, { threshold: 0.6 });
    observer.observe(node);
    return { destroy: () => observer.disconnect() };
  }
</script>

<svelte:head>
  <title>{t('feed.heading')} — {t('app.name')}</title>
</svelte:head>

<h1>{t('feed.heading')}</h1>

{#if !autoplay}
  <p class="feed-note">{t('feed.reducedMotion')}</p>
{/if}

{#if loading}
  <SkeletonDetail mediaHeight="24rem" lines={0} actions={false} />
{:else if clips.length === 0}
  <EmptyState illustration="empty-error" heading={t('feed.empty')} />
{:else}
  <div class="feed">
    {#each clips as clip (clip.listing_id)}
      <article class="feed__item">
        {#if clip.video_url}
          <video
            use:observe
            src={clip.video_url}
            muted
            loop
            playsinline
            preload="metadata"
            controls={!autoplay}
          ></video>
        {/if}
        <div class="feed__caption">
          <p class="feed__title">{clip.title}</p>
          {#if clip.artisan_name}<p class="feed__artisan">{clip.artisan_name}</p>{/if}
          <a class="feed__link" href={`/listing/${clip.listing_slug}`}>{t('feed.viewListing')}</a>
        </div>
      </article>
    {/each}
  </div>
{/if}

<style>
  .feed-note {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-3);
  }

  .feed {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-6);
    max-inline-size: 28rem;
    margin-inline: auto;
  }

  .feed__item video {
    inline-size: 100%;
    aspect-ratio: 9/16;
    object-fit: cover;
    border-radius: var(--k-radius-lg);
    background-color: var(--k-surface-sunken);
  }

  .feed__caption {
    padding: var(--k-space-3) 0;
  }

  .feed__title {
    font-weight: var(--k-weight-semibold);
  }

  .feed__artisan {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .feed__link {
    display: inline-block;
    margin-block-start: var(--k-space-2);
    color: inherit;
    text-decoration: underline;
  }
</style>
