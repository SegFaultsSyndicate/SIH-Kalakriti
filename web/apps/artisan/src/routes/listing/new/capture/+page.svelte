<!--
  apps/artisan/src/routes/listing/new/capture/+page.svelte

  Photo capture, step 1 of 7. A native <input type="file" capture> rather
  than getUserMedia: it works offline, needs no explicit camera permission
  dance in a PWA, and hands back the exact same Blob either way -- the
  simplest thing that gets a photo off the device. There is no synchronous
  photo-quality check -- nothing server-side can assess a photo before it
  reaches the bucket, so a fabricated blob-size heuristic used to stand in
  for one; it's gone. The real signal for a bad photo is asynchronous (the
  cataloguing pipeline's enhancement step failing and recording a reason on
  that media row) and isn't surfaced on this screen yet.
-->
<script lang="ts">
  import type { Component } from 'svelte';
  import type { SVGAttributes } from 'svelte/elements';
  import { liveQuery } from 'dexie';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { db, type MediaRecord } from '@kalakriti/offline';
  import ListingStep from '$lib/ListingStep.svelte';
  import SahayakTooltip from '$lib/SahayakTooltip.svelte';
  import { addCapturedMedia, removeCapturedMedia } from '$lib/listing-draft';
  import PhotographRaw from '@kalakriti/illustrations/src/onboard-photograph.svg';

  // Same cast as HandloomVerdict.svelte: a bare .svg import types as string.
  const Photograph = PhotographRaw as unknown as Component<SVGAttributes<SVGSVGElement>>;

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let fileInput: HTMLInputElement;
  let photos = $state<MediaRecord[]>([]);
  let photoUrls = $state<Record<string, string>>({});
  let locked = $state(false);

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (!draft) return [];
      const media = await db.media.bulkGet(draft.mediaIds);
      return media
        .filter((m): m is MediaRecord => !!m && m.kind === 'photo')
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
    }).subscribe((rows) => (photos = rows));
    return () => sub.unsubscribe();
  });

  // The listing.create outbox entry freezes its media list at enqueue time
  // (story step), and no later PATCH can add media (UpdateListingRequest has
  // no media field -- see ml_wiring.md). So once create is queued or landed,
  // photos are locked: adding one here would upload it and go nowhere.
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

  function openPicker(): void {
    fileInput.click();
  }

  async function onFileChosen(event: Event): Promise<void> {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file || !draftId) return;

    await addCapturedMedia(draftId, file, file.type || 'image/jpeg', 'photo', photos.length);
  }

  async function remove(id: string): Promise<void> {
    await removeCapturedMedia(draftId, id);
  }

  async function next(): Promise<void> {
    await goto(`/listing/new/studio?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.capture.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={0} heading={t('listing.capture.heading')} backHref="/">
  {#snippet children()}
    <input
      bind:this={fileInput}
      type="file"
      accept="image/*"
      capture="environment"
      class="k-visually-hidden"
      onchange={onFileChosen}
    />

    {#if photos.length > 0}
      <ul class="photo-grid" role="list">
        {#each photos as photo (photo.id)}
          <li class="photo-grid__item">
            <img src={photoUrls[photo.id]} alt="" />
            {#if !locked}
              <button
                type="button"
                class="photo-grid__remove"
                onclick={() => remove(photo.id)}
                aria-label={t('listing.capture.remove')}
              >
                <Icon name="close" />
              </button>
            {/if}
          </li>
        {/each}
        {#if !locked}
          <li>
            <button
              type="button"
              class="photo-grid__add"
              onclick={openPicker}
              aria-label={t('listing.capture.add')}
            >
              <Icon name="camera" size="1.75rem" />
            </button>
          </li>
        {/if}
      </ul>
    {/if}

    {#if locked}
      <p class="capture-status" role="status">{t('listing.capture.locked')}</p>
    {/if}

    {#if !locked}
      {#if photos.length === 0}
        <button type="button" class="dropzone" onclick={openPicker}>
          <Photograph class="dropzone__art" aria-hidden="true" focusable="false" />
          <span class="dropzone__label">
            <Icon name="camera" />
            {t('listing.capture.add')}
          </span>
          <span class="dropzone__hint">{t('listing.capture.hint')}</span>
        </button>
      {/if}
      <SahayakTooltip text={t('literacy.sahayak.tooltip.camera')} />
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={photos.length === 0} onclick={next} tooltip={tooltip('tooltip.next')}>{t('action.next')} →</Button>
  {/snippet}
</ListingStep>

<style>
  .photo-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-2);
  }

  .photo-grid__item {
    position: relative;
    aspect-ratio: 1;
    border-radius: var(--k-radius-md);
    overflow: hidden;
    border: var(--k-hairline) solid var(--k-border-hairline);
    box-shadow: 0 2px 6px rgb(0 0 0 / 0.08);
  }

  .photo-grid__item img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .photo-grid__remove {
    position: absolute;
    inset-block-start: var(--k-space-1);
    inset-inline-end: var(--k-space-1);
    display: flex;
    align-items: center;
    justify-content: center;
    min-block-size: 1.75rem;
    min-inline-size: 1.75rem;
    border: none;
    border-radius: var(--k-radius-pill);
    background-color: rgb(0 0 0 / 0.55);
    color: #fff;
    cursor: pointer;
  }

  .photo-grid__add,
  .dropzone {
    border: var(--k-rule) dashed var(--k-accent-primary-bg);
    background-color: color-mix(in srgb, var(--k-accent-primary-bg) 5%, var(--k-surface-raised));
    color: var(--k-accent-primary-text);
    cursor: pointer;
    transition: background-color 0.15s ease, border-style 0.15s ease;
  }

  .photo-grid__add:hover,
  .photo-grid__add:focus-visible,
  .dropzone:hover,
  .dropzone:focus-visible {
    border-style: solid;
    background-color: color-mix(in srgb, var(--k-accent-primary-bg) 11%, var(--k-surface-raised));
  }

  .photo-grid__add {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 100%;
    aspect-ratio: 1;
    border-radius: var(--k-radius-md);
  }

  .capture-status {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .dropzone {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-5) var(--k-space-4);
    border-radius: var(--k-radius-lg);
    text-align: center;
  }

  .dropzone :global(.dropzone__art) {
    inline-size: 9rem;
    block-size: auto;
    color: var(--k-text-secondary);
  }

  .dropzone__label {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-md);
    font-weight: var(--k-weight-semibold);
  }

  .dropzone__hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
