<!--
  apps/buyer/src/lib/VoicesReelCarousel.svelte

  "Voices from the Looms" - Authentic artisan process clips.
  Connects buyers directly to the hands, rhythmic soundscapes, and regional
  languages of master weaving families. Adheres to the Kalakriti design law:
  hairline boundaries, khadi surfaces, semantic tokens, and zero AI gloss.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Dialog, Button, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { getProcessFeed, type components } from '@kalakriti/api';

  type ProcessClip = components['schemas']['ProcessClip'];

  const t = $derived(locale.t);

  let apiClips = $state<ProcessClip[] | null>(null);
  let activeIndex = $state<number | null>(null);
  let isStoryOpen = $state(false);
  let isAudioMuted = $state(true);
  let videoEl = $state<HTMLVideoElement | null>(null);

  interface StoryItem extends ProcessClip {
    thumbnail_url?: string;
    craft_discipline?: string;
    cluster_origin?: string;
  }

  interface FallbackStory {
    listing_id: string;
    listing_slug: string;
    titleKey: MessageKey;
    artisanNameKey: MessageKey;
    thumbnail_url: string;
    craftDisciplineKey: MessageKey;
    cluster_origin: string;
    video_url: string;
  }

  const FALLBACK_STORIES: FallbackStory[] = [
    {
      listing_id: 'story-1',
      listing_slug: 'ajrakh-indigo-stole',
      titleKey: 'home.voices.story.1.title',
      artisanNameKey: 'home.voices.story.1.artisanName',
      thumbnail_url: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
      craftDisciplineKey: 'home.voices.story.1.craftDiscipline',
      cluster_origin: 'Dhamadka, Kutch',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4',
    },
    {
      listing_id: 'story-2',
      listing_slug: 'banarasi-kadwa-silk',
      titleKey: 'home.voices.story.2.title',
      artisanNameKey: 'home.voices.story.2.artisanName',
      thumbnail_url: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      craftDisciplineKey: 'home.voices.story.2.craftDiscipline',
      cluster_origin: 'Varanasi, UP',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4',
    },
    {
      listing_id: 'story-3',
      listing_slug: 'kashmir-pashmina-shawl',
      titleKey: 'home.voices.story.3.title',
      artisanNameKey: 'home.voices.story.3.artisanName',
      thumbnail_url: '/craft-images/embroidery/kashmir_pashmina_sozni_01.jpeg',
      craftDisciplineKey: 'home.voices.story.3.craftDiscipline',
      cluster_origin: 'Srinagar, J&K',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4',
    },
    {
      listing_id: 'story-4',
      listing_slug: 'dhokra-brass-figurine',
      titleKey: 'home.voices.story.4.title',
      artisanNameKey: 'home.voices.story.4.artisanName',
      thumbnail_url: '/craft-images/metalwork/dhokra-casting.jpg',
      craftDisciplineKey: 'home.voices.story.4.craftDiscipline',
      cluster_origin: 'Bastar, CG',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyBlazes.mp4',
    },
    {
      listing_id: 'story-5',
      listing_slug: 'pochampally-double-ikat',
      titleKey: 'home.voices.story.5.title',
      artisanNameKey: 'home.voices.story.5.artisanName',
      thumbnail_url: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      craftDisciplineKey: 'home.voices.story.5.craftDiscipline',
      cluster_origin: 'Pochampally, TG',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerMeltdowns.mp4',
    },
    {
      listing_id: 'story-6',
      listing_slug: 'nizamabad-black-pottery',
      titleKey: 'home.voices.story.6.title',
      artisanNameKey: 'home.voices.story.6.artisanName',
      thumbnail_url: '/craft-images/pottery/nizamabad-black-pottery.jpg',
      craftDisciplineKey: 'home.voices.story.6.craftDiscipline',
      cluster_origin: 'Nizamabad, UP',
      video_url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/Sintel.mp4',
    },
  ];

  function toStoryItem(raw: FallbackStory): StoryItem {
    return {
      listing_id: raw.listing_id,
      listing_slug: raw.listing_slug,
      title: t(raw.titleKey),
      artisan_name: t(raw.artisanNameKey),
      thumbnail_url: raw.thumbnail_url,
      craft_discipline: t(raw.craftDisciplineKey),
      cluster_origin: raw.cluster_origin,
      video_url: raw.video_url,
    };
  }

  const clips = $derived(
    apiClips && apiClips.length > 0 ? apiClips : FALLBACK_STORIES.map(toStoryItem)
  );

  onMount(async () => {
    try {
      const res = await getProcessFeed();
      apiClips = res.clips ?? [];
    } catch {
      apiClips = [];
    }
  });

  function openStory(index: number) {
    activeIndex = index;
    isStoryOpen = true;
    isAudioMuted = true;
    if (videoEl) {
      videoEl.currentTime = 0;
    }
  }

  function nextStory() {
    if (activeIndex !== null && activeIndex < clips.length - 1) {
      activeIndex += 1;
    } else {
      isStoryOpen = false;
    }
  }

  function prevStory() {
    if (activeIndex !== null && activeIndex > 0) {
      activeIndex -= 1;
    }
  }

  const activeClip = $derived(
    activeIndex !== null && clips[activeIndex] ? clips[activeIndex] : null
  );
</script>

<div class="voices-strip-container">
  <div class="reels-scroller" role="region" aria-label={t('home.voices.heading')}>
    <ul class="reels-list" role="list">
      {#each clips as clip, i (clip.listing_id || i)}
        {@const thumb = (clip as StoryItem).thumbnail_url}
        {@const discipline = (clip as StoryItem).craft_discipline || (clip.title ? clip.title.split(' ')[0] : t('home.voices.craftFallback'))}
        <li>
          <Tooltip text={tooltip('tooltip.play')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="reel-card"
                onclick={() => openStory(i)}
                aria-label={`${t('home.voices.playStory')}: ${clip.artisan_name ?? clip.title}`}
                {...tp}
              >
            <div class="reel-frame">
              {#if thumb}
                <img src={thumb} alt="" class="reel-thumb" loading="lazy" />
              {:else}
                <span class="reel-icon"><Icon name="process-video" size="1.25rem" /></span>
                <span class="reel-initial">{(clip.artisan_name || clip.title || 'K').slice(0, 1).toUpperCase()}</span>
              {/if}
              <span class="reel-play-tag" aria-hidden="true">
                <Icon name="play" size="0.65rem" />
              </span>
            </div>
            <div class="reel-meta">
              <span class="artisan-name">{clip.artisan_name || t('home.voices.masterArtisanFallback')}</span>
              <span class="craft-discipline">{discipline}</span>
            </div>
          </button>
            {/snippet}
          </Tooltip>
        </li>
      {/each}
    </ul>
  </div>
</div>

<!-- Accessible Dialog for Process Video -->
{#if activeClip}
  <Dialog bind:open={isStoryOpen} title={activeClip.title ?? t('home.voices.artisanStoryFallback')}>
    <div class="story-dialog-body">
      <div class="video-wrapper">
        <video
          bind:this={videoEl}
          src={activeClip.video_url}
          autoplay
          loop
          playsinline
          muted={isAudioMuted}
          controls
          class="story-player"
        ></video>
      </div>

      <div class="story-artisan-details">
        <div class="artisan-headline">
          <div>
            <p class="artisan-title">{activeClip.artisan_name}</p>
            <p class="artisan-badge-tag"><Icon name="verified-artisan" size="0.9rem" /> {t('home.voices.verifiedMasterGuildLoom')}</p>
          </div>
          <Tooltip text={tooltip('tooltip.toggleAudio')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="audio-toggle-btn"
                onclick={() => (isAudioMuted = !isAudioMuted)}
                {...tp}
              >
                <Icon name={isAudioMuted ? 'speaker' : 'volume'} size="1rem" />
                <span>{isAudioMuted ? t('home.voices.listenAudio') : t('home.voices.muteVoice')}</span>
              </button>
            {/snippet}
          </Tooltip>
        </div>

        <div class="dialog-actions">
          {#if activeIndex !== null && activeIndex > 0}
            <Button variant="secondary" onclick={prevStory} tooltip={tooltip('tooltip.prev')}>
              <Icon name="chevron-left" />
              <span>{t('action.previous')}</span>
            </Button>
          {/if}

          {#if activeClip.listing_slug}
            <a class="k-button k-button--primary" href={`/listing/${activeClip.listing_slug}`}>
              <span>{t('home.voices.viewFullListing')}</span>
              <Icon name="arrow-right" />
            </a>
          {/if}

          {#if activeIndex !== null && activeIndex < clips.length - 1}
            <Button variant="secondary" onclick={nextStory} tooltip={tooltip('tooltip.next')}>
              <span>{t('action.next')}</span>
              <Icon name="chevron-right" />
            </Button>
          {/if}
        </div>
      </div>
    </div>
  </Dialog>
{/if}

<style>
  .voices-strip-container {
    padding-block: var(--k-space-3);
  }

  .reels-scroller {
    overflow-x: auto;
    padding-block: var(--k-space-2);
    -webkit-overflow-scrolling: touch;
    scrollbar-width: thin;
  }

  .reels-list {
    display: flex;
    gap: var(--k-space-4);
    list-style: none;
    padding: 0;
    margin: 0;
    inline-size: max-content;
  }

  .reel-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    background: none;
    border: none;
    padding: var(--k-space-2);
    cursor: pointer;
    font-family: inherit;
    text-align: center;
    border-radius: var(--k-radius-sm);
    transition: background-color 0.15s ease;
  }

  .reel-card:hover {
    background-color: var(--k-surface-raised);
  }

  .reel-card:focus-visible {
    outline: 2px solid var(--k-focus-ring);
    outline-offset: 2px;
  }

  .reel-frame {
    inline-size: 4.5rem;
    block-size: 4.5rem;
    border-radius: var(--k-radius-full, 999px);
    border: 2px solid var(--k-border-accent);
    background-color: var(--k-surface-base);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: var(--k-accent-primary-text);
    position: relative;
    box-sizing: border-box;
    overflow: hidden;
    padding: 2px;
    transition: transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1), border-color 0.2s ease;
  }

  .reel-card:hover .reel-frame {
    transform: scale(1.05);
    border-color: var(--k-terracotta-800);
  }

  .reel-thumb {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: var(--k-radius-full, 999px);
    display: block;
  }

  .reel-play-tag {
    position: absolute;
    bottom: 2px;
    right: 2px;
    width: 1.15rem;
    height: 1.15rem;
    border-radius: 50%;
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1.5px solid var(--k-surface-base);
    z-index: 1;
  }

  .reel-icon {
    opacity: 0.8;
  }

  .reel-initial {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
  }

  .reel-meta {
    display: flex;
    flex-direction: column;
    max-inline-size: 6rem;
  }

  .artisan-name {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .craft-discipline {
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  /* Story Modal Layout */
  .story-dialog-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .video-wrapper {
    inline-size: 100%;
    aspect-ratio: 16 / 9;
    background-color: var(--k-surface-inverse);
    border-radius: var(--k-radius-sm);
    overflow: hidden;
  }

  .story-player {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .story-artisan-details {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .artisan-headline {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: var(--k-space-2);
  }

  .artisan-title {
    font-family: var(--k-font-display);
    font-size: var(--k-text-lg);
    color: var(--k-text-primary);
    margin: 0;
  }

  .artisan-badge-tag {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-accent-primary-text);
    margin: var(--k-space-1) 0 0 0;
  }

  .audio-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-base);
    color: var(--k-text-primary);
    font-size: var(--k-text-xs);
    cursor: pointer;
  }

  .dialog-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    align-items: center;
    gap: var(--k-space-2);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .k-button {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-4);
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    text-decoration: none;
  }

  .k-button--primary {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
  }
</style>
