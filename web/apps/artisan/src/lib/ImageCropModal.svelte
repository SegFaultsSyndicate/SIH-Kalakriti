<!--
  apps/artisan/src/lib/ImageCropModal.svelte

  Artisan Profile Picture Cropper & Resizer:
  - Interactive circular/square viewport with 3x3 alignment grid
  - Pan & drag positioning (pointer/touch support)
  - Zoom slider & stepper controls (1.0x to 3.0x)
  - 90-degree image rotation & reset centering
  - High-resolution HTML5 Canvas render export
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';

  interface Props {
    imageSrc: string;
    open: boolean;
    oncrop: (resultDataUrl: string) => void;
    oncancel: () => void;
  }

  let { imageSrc, open = false, oncrop, oncancel }: Props = $props();

  const t = $derived(locale.t);

  let zoom = $state(1.0);
  let rotation = $state(0); // in degrees: 0, 90, 180, 270
  let panX = $state(0);
  let panY = $state(0);

  let isDragging = $state(false);
  let dragStartX = $state(0);
  let dragStartY = $state(0);
  let startPanX = $state(0);
  let startPanY = $state(0);

  const CROP_SIZE = 280; // Size of circular viewport in px

  function onImageLoad(): void {
    resetPosition();
  }

  function resetPosition(): void {
    zoom = 1.0;
    rotation = 0;
    panX = 0;
    panY = 0;
  }

  function handlePointerDown(e: PointerEvent): void {
    e.preventDefault();
    isDragging = true;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    startPanX = panX;
    startPanY = panY;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function handlePointerMove(e: PointerEvent): void {
    if (!isDragging) return;
    e.preventDefault();
    const dx = e.clientX - dragStartX;
    const dy = e.clientY - dragStartY;
    panX = startPanX + dx;
    panY = startPanY + dy;
  }

  function handlePointerUp(e: PointerEvent): void {
    if (isDragging) {
      isDragging = false;
      try {
        (e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
      } catch {}
    }
  }

  function handleWheel(e: WheelEvent): void {
    e.preventDefault();
    const delta = e.deltaY * -0.0015;
    zoom = Math.min(3.0, Math.max(0.8, zoom + delta));
  }

  function rotateClockwise(): void {
    rotation = (rotation + 90) % 360;
  }

  function zoomIn(): void {
    zoom = Math.min(3.0, Number((zoom + 0.15).toFixed(2)));
  }

  function zoomOut(): void {
    zoom = Math.max(0.8, Number((zoom - 0.15).toFixed(2)));
  }

  function applyCrop(): void {
    if (typeof document === 'undefined') return;

    const canvas = document.createElement('canvas');
    const OUTPUT_SIZE = 480;
    canvas.width = OUTPUT_SIZE;
    canvas.height = OUTPUT_SIZE;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      ctx.clearRect(0, 0, OUTPUT_SIZE, OUTPUT_SIZE);

      // Smooth rendering
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';

      // Move context to center of target canvas
      ctx.translate(OUTPUT_SIZE / 2, OUTPUT_SIZE / 2);

      // Apply rotation
      ctx.rotate((rotation * Math.PI) / 180);

      // Scale factor mapping viewport crop size to high-res canvas
      const scaleMultiplier = OUTPUT_SIZE / CROP_SIZE;

      // Base scale to fit image into viewport (cover)
      const baseScale = Math.max(
        CROP_SIZE / img.naturalWidth,
        CROP_SIZE / img.naturalHeight,
      );

      const effectiveScale = baseScale * zoom * scaleMultiplier;

      // When rotated by 90 or 270, pan coordinates transform
      const rad = (-rotation * Math.PI) / 180;
      const unrotatedPanX = panX * Math.cos(rad) - panY * Math.sin(rad);
      const unrotatedPanY = panX * Math.sin(rad) + panY * Math.cos(rad);

      ctx.drawImage(
        img,
        -((img.naturalWidth * effectiveScale) / 2) + unrotatedPanX * scaleMultiplier,
        -((img.naturalHeight * effectiveScale) / 2) + unrotatedPanY * scaleMultiplier,
        img.naturalWidth * effectiveScale,
        img.naturalHeight * effectiveScale,
      );

      const croppedDataUrl = canvas.toDataURL('image/jpeg', 0.92);
      oncrop(croppedDataUrl);
    };
    img.src = imageSrc;
  }
</script>

{#if open}
  <div
    class="crop-backdrop"
    onclick={oncancel}
    onkeydown={(e) => (e.key === 'Escape' || e.key === 'Enter') && oncancel()}
    role="button"
    tabindex="0"
    aria-label="Close crop modal"
  >
    <div
      class="crop-dialog"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="crop-title"
      tabindex="-1"
    >
      <div class="crop-header">
        <div class="crop-header__text">
          <h2 id="crop-title" class="crop-title">
            <Icon name="camera" size="1.15rem" />
            {t('profile.crop.title')}
          </h2>
          <p class="crop-subtitle">{t('profile.crop.instruction')}</p>
        </div>
        <button type="button" class="crop-close-btn" onclick={oncancel} title="Close">
          <Icon name="close" size="1.1rem" />
        </button>
      </div>

      <!-- Interactive Viewport -->
      <div
        class="crop-viewport"
        style="width: {CROP_SIZE}px; height: {CROP_SIZE}px;"
        onpointerdown={handlePointerDown}
        onpointermove={handlePointerMove}
        onpointerup={handlePointerUp}
        onpointercancel={handlePointerUp}
        onwheel={handleWheel}
        role="application"
        aria-label="Drag to reposition photo"
      >
        <!-- The scaled, translated, rotated image -->
        <img
          src={imageSrc}
          alt="Preview to crop"
          class="crop-img"
          class:is-dragging={isDragging}
          style="
            transform: translate(calc(-50% + {panX}px), calc(-50% + {panY}px)) rotate({rotation}deg) scale({zoom});
          "
          onload={onImageLoad}
          draggable="false"
        />

        <!-- Circular Aperture Mask & Grid Lines -->
        <div class="crop-overlay">
          <!-- Circular cut-out -->
          <div class="crop-circle-frame">
            <!-- 3x3 Grid Guidelines for Alignment -->
            <div class="crop-grid-line crop-grid-line--h1"></div>
            <div class="crop-grid-line crop-grid-line--h2"></div>
            <div class="crop-grid-line crop-grid-line--v1"></div>
            <div class="crop-grid-line crop-grid-line--v2"></div>
          </div>
        </div>

        <div class="crop-viewport-badge">
          <span>{Math.round(zoom * 100)}%</span>
        </div>
      </div>

      <!-- Zoom & Adjustment Controls -->
      <div class="crop-controls">
        <div class="crop-zoom-bar">
          <button type="button" class="crop-tool-btn" onclick={zoomOut} title="Zoom Out">
            <Icon name="chevron-down" size="1rem" />
          </button>
          <input
            type="range"
            min="0.8"
            max="3.0"
            step="0.02"
            bind:value={zoom}
            class="crop-slider"
            aria-label={t('profile.crop.zoom')}
          />
          <button type="button" class="crop-tool-btn" onclick={zoomIn} title="Zoom In">
            <Icon name="chevron-up" size="1rem" />
          </button>
        </div>

        <div class="crop-tool-actions">
          <button type="button" class="crop-action-btn" onclick={rotateClockwise} title="Rotate 90°">
            <Icon name="refresh" size="0.9rem" />
            {t('profile.crop.rotate')}
          </button>
          <button type="button" class="crop-action-btn" onclick={resetPosition} title="Reset position">
            <Icon name="refresh" size="0.9rem" />
            {t('profile.crop.reset')}
          </button>
        </div>
      </div>

      <!-- Footer Buttons -->
      <div class="crop-footer">
        <Button variant="secondary" size="md" onclick={oncancel}>
          {t('profile.crop.cancel')}
        </Button>
        <Button variant="primary" size="md" onclick={applyCrop}>
          <Icon name="check" size="1rem" />
          {t('profile.crop.apply')}
        </Button>
      </div>
    </div>
  </div>
{/if}

<style>
  .crop-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1200;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: rgba(18, 14, 10, 0.72);
    backdrop-filter: blur(4px);
    padding: var(--k-space-3);
  }

  .crop-dialog {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-3);
    width: 100%;
    max-width: 22rem;
    padding: var(--k-space-4);
    background-color: var(--k-khadi-50, var(--k-surface-base));
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-lg, 12px);
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.28);
    user-select: none;
    animation: crop-pop 180ms ease-out;
  }

  @keyframes crop-pop {
    from {
      opacity: 0;
      transform: scale(0.96) translateY(8px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .crop-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    width: 100%;
    padding-block-end: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-stone-200, var(--k-border-subtle));
  }

  .crop-header__text {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .crop-title {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin: 0;
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-semibold);
    color: var(--k-terracotta-900, var(--k-stone-800));
  }

  .crop-subtitle {
    margin: 0;
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary, var(--k-stone-600));
    line-height: 1.35;
  }

  .crop-close-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.85rem;
    height: 1.85rem;
    background: transparent;
    border: none;
    color: var(--k-text-secondary);
    font-size: 1.1rem;
    cursor: pointer;
    border-radius: var(--k-radius-pill);
    transition: background 150ms ease;
  }

  .crop-close-btn:hover {
    background-color: var(--k-stone-200);
  }

  /* Viewport & Mask */
  .crop-viewport {
    position: relative;
    overflow: hidden;
    margin-block: var(--k-space-2);
    border-radius: var(--k-radius-md, 8px);
    background-color: var(--k-ink-900);
    cursor: grab;
    touch-action: none;
  }

  .crop-viewport:active {
    cursor: grabbing;
  }

  .crop-img {
    position: absolute;
    top: 50%;
    left: 50%;
    max-width: none;
    pointer-events: none;
    transform-origin: center center;
    transition: transform 30ms linear;
  }

  .crop-overlay {
    position: absolute;
    inset: 0;
    pointer-events: none;
    box-shadow: 0 0 0 9999px rgba(18, 14, 10, 0.65);
    border-radius: 50%;
  }

  .crop-circle-frame {
    position: absolute;
    inset: 0;
    border: 2px solid rgba(255, 255, 255, 0.92);
    border-radius: 50%;
    overflow: hidden;
  }

  /* 3x3 Grid Overlay */
  .crop-grid-line {
    position: absolute;
    background-color: rgba(255, 255, 255, 0.4);
  }

  .crop-grid-line--h1 {
    top: 33.33%;
    left: 0;
    right: 0;
    height: 1px;
    border-top: 1px dashed rgba(255, 255, 255, 0.6);
  }

  .crop-grid-line--h2 {
    top: 66.66%;
    left: 0;
    right: 0;
    height: 1px;
    border-top: 1px dashed rgba(255, 255, 255, 0.6);
  }

  .crop-grid-line--v1 {
    left: 33.33%;
    top: 0;
    bottom: 0;
    width: 1px;
    border-left: 1px dashed rgba(255, 255, 255, 0.6);
  }

  .crop-grid-line--v2 {
    left: 66.66%;
    top: 0;
    bottom: 0;
    width: 1px;
    border-left: 1px dashed rgba(255, 255, 255, 0.6);
  }

  .crop-viewport-badge {
    position: absolute;
    bottom: 0.5rem;
    right: 0.5rem;
    padding: 0.15rem 0.4rem;
    background: rgba(0, 0, 0, 0.65);
    color: var(--k-text-on-accent);
    font-size: 0.65rem;
    font-weight: var(--k-weight-medium);
    border-radius: var(--k-radius-xs, 4px);
    pointer-events: none;
  }

  /* Controls */
  .crop-controls {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    width: 100%;
  }

  .crop-zoom-bar {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    width: 100%;
    padding: var(--k-space-1) var(--k-space-2);
    background-color: var(--k-khadi-100, var(--k-surface-raised));
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-pill, 999px);
  }

  .crop-tool-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    background: transparent;
    border: none;
    color: var(--k-text-primary);
    cursor: pointer;
    border-radius: 50%;
  }

  .crop-tool-btn:hover {
    background-color: var(--k-stone-200);
  }

  .crop-slider {
    flex: 1;
    height: 4px;
    accent-color: var(--k-terracotta-700, var(--k-terracotta-800));
    cursor: pointer;
  }

  .crop-tool-actions {
    display: flex;
    justify-content: center;
    gap: var(--k-space-3);
    width: 100%;
  }

  .crop-action-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    background: transparent;
    border: var(--k-hairline) solid var(--k-stone-300);
    padding: var(--k-space-1) var(--k-space-2);
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    cursor: pointer;
    transition: all 150ms ease;
  }

  .crop-action-btn:hover {
    background-color: var(--k-surface-pressed);
    color: var(--k-text-primary);
    border-color: var(--k-terracotta-600);
  }

  /* Footer */
  .crop-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--k-space-2);
    width: 100%;
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }
</style>
