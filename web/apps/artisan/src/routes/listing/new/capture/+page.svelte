<!--
  apps/artisan/src/routes/listing/new/capture/+page.svelte

  Photo capture, step 1 of 7. A native <input type="file" capture> rather
  than getUserMedia: it works offline, needs no explicit camera permission
  dance in a PWA, and hands back the exact same Blob either way -- the
  simplest thing that gets a photo off the device. Each photo runs through
  the (mocked, see ml_wiring.md) quality gate before it is queued for
  upload, so a bad shot is caught before it ever reaches the outbox.
-->
<script lang="ts">
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
  import { assessPhotoQuality, type QualityIssue } from '$lib/ml-mock';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let fileInput: HTMLInputElement;
  let photos = $state<MediaRecord[]>([]);
  let photoUrls = $state<Record<string, string>>({});
  let checking = $state(false);
  let issue = $state<QualityIssue | undefined>(undefined);
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

    checking = true;
    issue = undefined;
    const assessment = await assessPhotoQuality(file);
    checking = false;

    if (!assessment.passed) {
      issue = assessment.issues[0];
      return;
    }
    await addCapturedMedia(draftId, file, file.type || 'image/jpeg', 'photo', photos.length);
  }

  function retake(): void {
    issue = undefined;
    openPicker();
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
      </ul>
    {/if}

    {#if locked}
      <p class="capture-status" role="status">{t('listing.capture.locked')}</p>
    {/if}

    {#if checking}
      <p class="capture-status" role="status">{t('listing.capture.checking')}</p>
    {/if}

    {#if issue}
      <div class="photo-issue" role="alert">
        <Icon name="warning" />
        <p>{t(issue.messageKey)}</p>
        <Button size="sm" onclick={retake} tooltip={tooltip('tooltip.retakePhoto')}>{t('listing.capture.retake')}</Button>
      </div>
    {/if}

    {#if !locked}
      <button type="button" class="add-photo" onclick={openPicker}>
        <Icon name="camera" />
        {t('listing.capture.add')}
      </button>

      <p class="capture-hint">{t('listing.capture.hint')}</p>
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
    min-block-size: 1.75rem;
    min-inline-size: 1.75rem;
    border: none;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .capture-status {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .photo-issue {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    align-items: start;
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-accent-danger);
    border-radius: var(--k-radius-md);
    color: var(--k-accent-danger);
  }

  .add-photo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    min-block-size: calc(var(--k-touch-min) * 1.2);
    border: var(--k-rule) dashed var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: none;
    color: var(--k-text-primary);
    font-size: var(--k-text-md);
    cursor: pointer;
  }

  .capture-hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
