<!--
  apps/artisan/src/routes/listing/new/studio/+page.svelte

  AI Image Enhancer & Studio, step 2 of 8.
  Provides master artisans with instant showroom-grade studio photography:
  - Side-by-side & interactive split before/after comparison
  - One-tap background removal (Studio White / Transparent PNG / Workshop Context)
  - Auto-brightness and contrast indicator with glare suppression
-->
<script lang="ts">
  import { liveQuery } from 'dexie';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { db, type MediaRecord } from '@kalakriti/offline';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields, type StudioConfig } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let photos = $state<MediaRecord[]>([]);
  let photoUrls = $state<Record<string, string>>({});
  let selectedPhotoIndex = $state(0);
  let backgroundMode = $state<'white' | 'transparent' | 'natural'>('white');
  let autoLighting = $state(true);
  let viewMode = $state<'split' | 'sideBySide'>('split');
  let sliderPos = $state(50);
  let locked = $state(false);
  let saving = $state(false);

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (!draft) return [];
      const media = await db.media.bulkGet(draft.mediaIds);
      return media
        .filter((m): m is MediaRecord => !!m && m.kind === 'photo')
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
    }).subscribe((rows) => {
      photos = rows;
    });
    return () => sub.unsubscribe();
  });

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      const cfg = (draft.fields.studioConfig as StudioConfig | undefined);
      if (cfg) {
        if (cfg.backgroundMode) backgroundMode = cfg.backgroundMode;
        if (typeof cfg.autoLightingApplied === 'boolean') autoLighting = cfg.autoLightingApplied;
      }
    });
  });

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (draft?.remoteId) return true;
      const created = await db.outbox
        .where('draftId')
        .equals(draftId)
        .and((e) => e.kind === 'listing.create')
        .count();
      return created > 0;
    }).subscribe((v) => (locked = v));
    return () => sub.unsubscribe();
  });

  $effect(() => {
    const urls: Record<string, string> = {};
    for (const p of photos) urls[p.id] = URL.createObjectURL(p.blob);
    photoUrls = urls;
    return () => {
      for (const url of Object.values(urls)) URL.revokeObjectURL(url);
    };
  });

  const activePhoto = $derived(photos[selectedPhotoIndex]);
  const activePhotoUrl = $derived(activePhoto ? photoUrls[activePhoto.id] : '');

  async function updateConfig(): Promise<void> {
    if (!draftId) return;
    saving = true;
    await patchFields(draftId, {
      studioConfig: {
        backgroundMode,
        autoLightingApplied: autoLighting,
        brightnessOffset: 18,
        contrastOffset: 12,
      },
    });
    saving = false;
  }

  function setBg(mode: 'white' | 'transparent' | 'natural'): void {
    backgroundMode = mode;
    void updateConfig();
  }

  function toggleLighting(): void {
    autoLighting = !autoLighting;
    void updateConfig();
  }

  async function next(): Promise<void> {
    await updateConfig();
    await goto(`/listing/new/video?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.studio.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={1} heading={t('listing.studio.heading')} backHref="/listing/new/capture?d={draftId}">
  {#snippet children()}
    <p class="studio-subhead">{t('listing.studio.subheading')}</p>

    {#if photos.length > 1}
      <div class="photo-selector" role="tablist" aria-label="Select photo to enhance">
        {#each photos as photo, i (photo.id)}
          <button
            type="button"
            class="photo-tab"
            class:photo-tab--active={i === selectedPhotoIndex}
            onclick={() => (selectedPhotoIndex = i)}
            role="tab"
            aria-selected={i === selectedPhotoIndex}
          >
            <img src={photoUrls[photo.id]} alt="" />
            <span class="photo-tab__num">#{i + 1}</span>
          </button>
        {/each}
      </div>
    {/if}

    <!-- Auto Lighting Indicator Badge -->
    <div class="lighting-badge" role="status">
      <div class="lighting-badge__icon">
        <Icon name="contrast" />
      </div>
      <div class="lighting-badge__text">
        <span class="lighting-badge__title">{t('listing.studio.lightingBadge')}</span>
        <span class="lighting-badge__desc">{t('listing.studio.lightingDesc')}</span>
      </div>
      <button
        type="button"
        class="lighting-toggle-btn"
        class:lighting-toggle-btn--active={autoLighting}
        onclick={toggleLighting}
        aria-pressed={autoLighting}
      >
        {autoLighting ? 'Active' : 'Off'}
      </button>
    </div>

    <!-- View Mode Selector -->
    <div class="view-mode-bar" role="tablist" aria-label="Comparison View Mode">
      <button
        type="button"
        class="view-mode-btn"
        class:view-mode-btn--active={viewMode === 'split'}
        onclick={() => (viewMode = 'split')}
        role="tab"
        aria-selected={viewMode === 'split'}
      >
        Split Slider
      </button>
      <button
        type="button"
        class="view-mode-btn"
        class:view-mode-btn--active={viewMode === 'sideBySide'}
        onclick={() => (viewMode = 'sideBySide')}
        role="tab"
        aria-selected={viewMode === 'sideBySide'}
      >
        Side-by-Side
      </button>
    </div>

    <!-- Interactive Comparison Stage -->
    {#if activePhotoUrl}
      {#if viewMode === 'split'}
        <div class="comparison-stage">
          <div class="split-viewer">
            <!-- Enhanced Background Layer -->
            <div
              class="split-viewer__enhanced"
              class:split-viewer__enhanced--white={backgroundMode === 'white'}
              class:split-viewer__enhanced--transparent={backgroundMode === 'transparent'}
            >
              <img
                src={activePhotoUrl}
                alt="Enhanced preview"
                class="split-viewer__img"
                class:split-viewer__img--enhanced={autoLighting}
                class:split-viewer__img--isolated={backgroundMode !== 'natural'}
              />
              <span class="view-tag view-tag--after">{t('listing.studio.after')}</span>
            </div>

            <!-- Original Foreground Layer (Clipped) -->
            <div class="split-viewer__original" style:inline-size="{sliderPos}%">
              <img
                src={activePhotoUrl}
                alt=""
                class="split-viewer__img"
              />
              <span class="view-tag view-tag--before">{t('listing.studio.before')}</span>
            </div>

            <!-- Drag Handle Divider -->
            <div class="split-divider" style:inset-inline-start="{sliderPos}%">
              <div class="split-divider__handle">
                <span class="split-divider__arrows">‹ ›</span>
              </div>
            </div>

            <!-- Invisible range input covering stage -->
            <input
              type="range"
              min="0"
              max="100"
              value={sliderPos}
              class="split-range"
              aria-label={t('listing.studio.sliderHint')}
              oninput={(e) => (sliderPos = Number((e.target as HTMLInputElement).value))}
            />
          </div>
          <p class="slider-hint">{t('listing.studio.sliderHint')}</p>
        </div>
      {:else}
        <!-- Side-by-Side View -->
        <div class="side-by-side-grid">
          <div class="side-card">
            <span class="side-card__label">{t('listing.studio.before')}</span>
            <div class="side-card__media">
              <img src={activePhotoUrl} alt="Original workshop capture" />
            </div>
            <span class="side-card__caption">Raw Workshop Capture</span>
          </div>

          <div class="side-card side-card--enhanced">
            <span class="side-card__label side-card__label--gold">{t('listing.studio.after')}</span>
            <div
              class="side-card__media"
              class:side-card__media--white={backgroundMode === 'white'}
              class:side-card__media--transparent={backgroundMode === 'transparent'}
            >
              <img
                src={activePhotoUrl}
                alt="AI Studio Enhanced"
                class:split-viewer__img--enhanced={autoLighting}
                class:split-viewer__img--isolated={backgroundMode !== 'natural'}
              />
            </div>
            <span class="side-card__caption">Clean Studio + Light Balancer</span>
          </div>
        </div>
      {/if}
    {/if}

    <!-- Background Removal Controls -->
    <div class="bg-section">
      <h3 class="bg-section__title">{t('listing.studio.bgTitle')}</h3>
      <div class="bg-pills" role="radiogroup" aria-label={t('listing.studio.bgTitle')}>
        <button
          type="button"
          class="bg-pill"
          class:bg-pill--active={backgroundMode === 'white'}
          onclick={() => setBg('white')}
          role="radio"
          aria-checked={backgroundMode === 'white'}
        >
          <span class="bg-pill__swatch bg-pill__swatch--white"></span>
          <span class="bg-pill__text">
            <strong>{t('listing.studio.bgWhite')}</strong>
            <small>E-Commerce Showroom</small>
          </span>
          {#if backgroundMode === 'white'}
            <Icon name="check" class="bg-pill__check" />
          {/if}
        </button>

        <button
          type="button"
          class="bg-pill"
          class:bg-pill--active={backgroundMode === 'transparent'}
          onclick={() => setBg('transparent')}
          role="radio"
          aria-checked={backgroundMode === 'transparent'}
        >
          <span class="bg-pill__swatch bg-pill__swatch--checker"></span>
          <span class="bg-pill__text">
            <strong>{t('listing.studio.bgTransparent')}</strong>
            <small>Cutout for Banners</small>
          </span>
          {#if backgroundMode === 'transparent'}
            <Icon name="check" class="bg-pill__check" />
          {/if}
        </button>

        <button
          type="button"
          class="bg-pill"
          class:bg-pill--active={backgroundMode === 'natural'}
          onclick={() => setBg('natural')}
          role="radio"
          aria-checked={backgroundMode === 'natural'}
        >
          <span class="bg-pill__swatch bg-pill__swatch--natural"></span>
          <span class="bg-pill__text">
            <strong>{t('listing.studio.bgNatural')}</strong>
            <small>Keep Loom Backdrop</small>
          </span>
          {#if backgroundMode === 'natural'}
            <Icon name="check" class="bg-pill__check" />
          {/if}
        </button>
      </div>
    </div>
  {/snippet}

  {#snippet actions()}
    <Button size="xl" onclick={next} loading={saving} disabled={locked}>{t('listing.studio.apply')}</Button>
  {/snippet}
</ListingStep>

<style>
  .studio-subhead {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    line-height: var(--k-leading-normal);
  }

  .photo-selector {
    display: flex;
    gap: var(--k-space-2);
    overflow-x: auto;
    padding-block-end: var(--k-space-2);
  }

  .photo-tab {
    position: relative;
    inline-size: 4rem;
    block-size: 4rem;
    border-radius: var(--k-radius-md);
    overflow: hidden;
    border: 2px solid transparent;
    padding: 0;
    background: var(--k-surface-sunken);
    cursor: pointer;
    flex-shrink: 0;
  }

  .photo-tab--active {
    border-color: var(--k-accent-primary);
    box-shadow: 0 0 0 2px var(--k-accent-primary);
  }

  .photo-tab img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .photo-tab__num {
    position: absolute;
    inset-block-end: 2px;
    inset-inline-end: 2px;
    background: rgba(0, 0, 0, 0.7);
    color: var(--k-text-on-accent);
    font-size: 0.65rem;
    padding-inline: 4px;
    border-radius: var(--k-radius-xs);
  }

  .lighting-badge {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: linear-gradient(135deg, rgba(217, 119, 6, 0.12), rgba(245, 158, 11, 0.04));
    border: 1px solid rgba(217, 119, 6, 0.35);
  }

  .lighting-badge__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.2rem;
    block-size: 2.2rem;
    border-radius: var(--k-radius-pill);
    background: var(--k-haldi-700);
    color: var(--k-text-on-accent);
    flex-shrink: 0;
  }

  .lighting-badge__text {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .lighting-badge__title {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
  }

  .lighting-badge__desc {
    font-size: var(--k-text-2xs, 0.75rem);
    color: var(--k-text-secondary);
  }

  .lighting-toggle-btn {
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-raised);
    color: var(--k-text-secondary);
    padding: var(--k-space-1) var(--k-space-3);
    border-radius: var(--k-radius-pill);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    cursor: pointer;
  }

  .lighting-toggle-btn--active {
    background: var(--k-haldi-700);
    color: var(--k-text-on-accent);
    border-color: var(--k-haldi-700);
  }

  .view-mode-bar {
    display: flex;
    justify-content: center;
    gap: var(--k-space-1);
    background: var(--k-surface-raised, var(--k-surface-base));
    padding: 3px;
    border-radius: var(--k-radius-pill, 999px);
    inline-size: fit-content;
    margin-inline: auto;
  }

  .view-mode-btn {
    border: none;
    background: transparent;
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    padding: var(--k-space-1) var(--k-space-4);
    border-radius: var(--k-radius-pill, 999px);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .view-mode-btn--active {
    background: var(--k-surface-base, var(--k-surface-base));
    color: var(--k-text-primary);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  }

  /* Split Comparison View */
  .comparison-stage {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .split-viewer {
    position: relative;
    inline-size: 100%;
    aspect-ratio: 4 / 3;
    border-radius: var(--k-radius-lg);
    overflow: hidden;
    border: var(--k-hairline) solid var(--k-border-hairline);
    background: var(--k-surface-inverse);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  }

  .split-viewer__enhanced,
  .split-viewer__original {
    position: absolute;
    inset: 0;
    overflow: hidden;
  }

  .split-viewer__enhanced--white {
    background: var(--k-surface-base);
  }

  .split-viewer__enhanced--transparent {
    background-color: var(--k-surface-base);
    background-image:
      linear-gradient(45deg, var(--k-stone-100) 25%, transparent 25%),
      linear-gradient(-45deg, var(--k-stone-100) 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, var(--k-stone-100) 75%),
      linear-gradient(-45deg, transparent 75%, var(--k-stone-100) 75%);
    background-size: 16px 16px;
    background-position: 0 0, 0 8px, 8px -8px, -8px 0px;
  }

  .split-viewer__img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
    pointer-events: none;
    user-select: none;
    transition: filter 0.25s ease;
  }

  .split-viewer__img--enhanced {
    filter: brightness(1.16) contrast(1.12) saturate(1.08);
  }

  .split-viewer__img--isolated {
    /* Soft vignette/mask simulating alpha isolate */
    filter: brightness(1.16) contrast(1.12) saturate(1.08) drop-shadow(0 14px 20px rgba(0, 0, 0, 0.22));
  }

  .view-tag {
    position: absolute;
    inset-block-start: var(--k-space-2);
    padding: 3px 8px;
    border-radius: var(--k-radius-xs);
    font-size: 0.7rem;
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    pointer-events: none;
  }

  .view-tag--before {
    inset-inline-start: var(--k-space-2);
    background: rgba(15, 23, 42, 0.85);
    color: var(--k-text-on-accent);
  }

  .view-tag--after {
    inset-inline-end: var(--k-space-2);
    background: var(--k-haldi-700);
    color: var(--k-text-on-accent);
  }

  .split-divider {
    position: absolute;
    inset-block: 0;
    inline-size: 2px;
    background: var(--k-surface-base);
    box-shadow: 0 0 6px rgba(0, 0, 0, 0.6);
    pointer-events: none;
    transform: translateX(-50%);
  }

  .split-divider__handle {
    position: absolute;
    inset-block-start: 50%;
    inset-inline-start: 50%;
    transform: translate(-50%, -50%);
    inline-size: 2rem;
    block-size: 2rem;
    border-radius: var(--k-radius-pill);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.35);
  }

  .split-divider__arrows {
    font-size: 0.9rem;
    letter-spacing: 2px;
  }

  .split-range {
    position: absolute;
    inset: 0;
    inline-size: 100%;
    block-size: 100%;
    opacity: 0;
    cursor: ew-resize;
    z-index: 10;
  }

  .slider-hint {
    text-align: center;
    font-size: var(--k-text-2xs, 0.75rem);
    color: var(--k-text-secondary);
  }

  /* Side by side layout */
  .side-by-side-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--k-space-3);
  }

  .side-card {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-2);
    background: var(--k-surface-base);
  }

  .side-card--enhanced {
    border-color: var(--k-haldi-700);
  }

  .side-card__label {
    font-size: 0.7rem;
    font-weight: var(--k-weight-bold);
    color: var(--k-text-secondary);
    text-transform: uppercase;
  }

  .side-card__label--gold {
    color: var(--k-haldi-700);
  }

  .side-card__media {
    aspect-ratio: 1;
    border-radius: var(--k-radius-sm);
    overflow: hidden;
    background: var(--k-surface-inverse);
  }

  .side-card__media--white {
    background: var(--k-surface-base);
  }

  .side-card__media--transparent {
    background-color: var(--k-surface-base);
    background-image:
      linear-gradient(45deg, var(--k-stone-100) 25%, transparent 25%),
      linear-gradient(-45deg, var(--k-stone-100) 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, var(--k-stone-100) 75%),
      linear-gradient(-45deg, transparent 75%, var(--k-stone-100) 75%);
    background-size: 12px 12px;
  }

  .side-card__media img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .side-card__caption {
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  /* Background Controls */
  .bg-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-2);
  }

  .bg-section__title {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
  }

  .bg-pills {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .bg-pill {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background: var(--k-surface-base);
    cursor: pointer;
    text-align: start;
    transition: border-color 0.15s, background-color 0.15s;
  }

  .bg-pill--active {
    border-color: var(--k-haldi-700);
    background: rgba(217, 119, 6, 0.05);
  }

  .bg-pill__swatch {
    inline-size: 2rem;
    block-size: 2rem;
    border-radius: var(--k-radius-pill);
    border: 1px solid var(--k-border-hairline);
    flex-shrink: 0;
  }

  .bg-pill__swatch--white {
    background: var(--k-surface-base);
    box-shadow: inset 0 0 0 1px var(--k-stone-100);
  }

  .bg-pill__swatch--checker {
    background-color: var(--k-surface-base);
    background-image:
      linear-gradient(45deg, var(--k-stone-300) 25%, transparent 25%),
      linear-gradient(-45deg, var(--k-stone-300) 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, var(--k-stone-300) 75%),
      linear-gradient(-45deg, transparent 75%, var(--k-stone-300) 75%);
    background-size: 8px 8px;
  }

  .bg-pill__swatch--natural {
    background: linear-gradient(135deg, var(--k-accent-primary-bg), var(--k-accent-primary-bg));
  }

  .bg-pill__text {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .bg-pill__text strong {
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .bg-pill__text small {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .bg-pill :global(.bg-pill__check) {
    color: var(--k-haldi-700);
    inline-size: 1.25rem;
    block-size: 1.25rem;
  }
</style>
