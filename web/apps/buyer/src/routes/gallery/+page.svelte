<!--
  apps/buyer/src/routes/gallery/+page.svelte

  Media & Craft Gallery (footer). Every craft photo shipped in
  static/craft-images, grouped by the 12 craft categories, each group
  linking on to that category's search results.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { ARTISAN_CRAFT_CATEGORIES } from '$lib/craft-categories';
  import { GALLERY_IMAGES } from '$lib/gallery-images';

  const t = $derived(locale.t);

  const seenImages = new Set<string>();
  const groups = ARTISAN_CRAFT_CATEGORIES.map((c) => {
    const folder = c.sampleImage.split('/')[2];
    const images = (GALLERY_IMAGES[folder] ?? [])
      .map((f) => `/craft-images/${folder}/${f}`)
      .filter((src) => {
        if (seenImages.has(src)) return false;
        seenImages.add(src);
        return true;
      });
    return { category: c, images };
  }).filter((g) => g.images.length > 0);

  let open = $state<{ src: string; label: string } | null>(null);
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    if (open) dialog?.showModal();
    else dialog?.close();
  });
</script>

<svelte:head>
  <title>{t('gallery.title')} — {t('app.name')}</title>
  <meta name="description" content={t('gallery.subtitle')} />
</svelte:head>

<div class="gallery">
  <header class="gallery__header">
    <h1>{t('gallery.title')}</h1>
    <p>{t('gallery.subtitle')}</p>
  </header>

  {#each groups as { category, images } (category.id)}
    <section class="gallery__group" aria-labelledby="gallery-{category.id}">
      <div class="gallery__group-head">
        <h2 id="gallery-{category.id}">{t(category.nameKey)}</h2>
        <a href="/search?category={encodeURIComponent(category.query)}">{t('gallery.shopCategory')}</a>
      </div>
      <ul class="gallery__grid" role="list">
        {#each images as src (src)}
          <li>
            <button type="button" onclick={() => (open = { src, label: t(category.nameKey) })}>
              <img {src} alt={t(category.nameKey)} loading="lazy" />
            </button>
          </li>
        {/each}
      </ul>
    </section>
  {/each}
</div>

<dialog class="gallery__lightbox" bind:this={dialog} onclose={() => (open = null)} aria-label={open?.label}>
  {#if open}
    <img src={open.src} alt={open.label} />
    <form method="dialog">
      <button class="k-button k-button--secondary">{t('ui.dialog.close')}</button>
    </form>
  {/if}
</dialog>

<style>
  .gallery {
    max-inline-size: 72rem;
    margin-inline: auto;
    padding: var(--k-space-6) var(--k-space-4) var(--k-space-12);
  }

  .gallery__header h1 {
    font-family: var(--k-font-display, serif);
    font-size: clamp(1.6rem, 4vw, 2.2rem);
    margin: 0 0 var(--k-space-2);
  }

  .gallery__header p {
    color: var(--k-text-secondary);
    max-inline-size: 44rem;
    margin: 0 0 var(--k-space-6);
  }

  .gallery__group {
    margin-block-end: var(--k-space-7);
  }

  .gallery__group-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-3);
    margin-block-end: var(--k-space-3);
  }

  .gallery__group-head h2 {
    font-size: var(--k-text-lg);
    margin: 0;
  }

  .gallery__group-head a {
    font-size: var(--k-text-sm);
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  .gallery__grid {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(11rem, 1fr));
    gap: var(--k-space-3);
  }

  .gallery__grid button {
    display: block;
    inline-size: 100%;
    padding: 0;
    border: 0;
    border-radius: var(--k-radius-md);
    overflow: hidden;
    cursor: zoom-in;
    background: var(--k-surface-raised, var(--k-surface-base));
  }

  .gallery__grid img {
    display: block;
    inline-size: 100%;
    aspect-ratio: 1;
    object-fit: cover;
    transition: transform 0.2s ease;
  }

  .gallery__grid button:hover img,
  .gallery__grid button:focus-visible img {
    transform: scale(1.04);
  }

  .gallery__lightbox {
    max-inline-size: min(64rem, 94vw);
    padding: var(--k-space-3);
    border: 0;
    border-radius: var(--k-radius-lg);
    background: var(--k-surface-base);
  }

  .gallery__lightbox::backdrop {
    background: rgb(0 0 0 / 0.75);
  }

  .gallery__lightbox img {
    display: block;
    max-inline-size: 100%;
    max-block-size: 80vh;
    margin-inline: auto;
  }

  .gallery__lightbox form {
    display: flex;
    justify-content: flex-end;
    margin-block-start: var(--k-space-3);
  }
</style>
