<!--
  apps/artisan/src/routes/listing/new/video/+page.svelte

  Optional product video, step 2 of 7. One video per listing -- recording a
  new one replaces whatever was captured before, same as retaking a photo.
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
  import { addCapturedMedia, removeCapturedMedia } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let fileInput: HTMLInputElement;
  let video = $state<MediaRecord | undefined>(undefined);
  let videoUrl = $state<string | undefined>(undefined);
  let locked = $state(false);

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (!draft) return undefined;
      const media = await db.media.bulkGet(draft.mediaIds);
      return media.find((m): m is MediaRecord => !!m && m.kind === 'video');
    }).subscribe((row) => (video = row));
    return () => sub.unsubscribe();
  });

  // Same freeze as the capture step -- listing.create locks in the media
  // list at story-step enqueue time, so a video added after cannot reach
  // the listing (see ml_wiring.md).
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
    if (!video) {
      videoUrl = undefined;
      return;
    }
    const url = URL.createObjectURL(video.blob);
    videoUrl = url;
    return () => URL.revokeObjectURL(url);
  });

  function openPicker(): void {
    fileInput.click();
  }

  async function onFileChosen(event: Event): Promise<void> {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file || !draftId) return;
    if (video) await removeCapturedMedia(draftId, video.id);
    await addCapturedMedia(draftId, file, file.type || 'video/mp4', 'video');
  }

  async function remove(): Promise<void> {
    if (!video) return;
    await removeCapturedMedia(draftId, video.id);
  }

  async function next(): Promise<void> {
    await goto(`/listing/new/story?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.video.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={2} heading={t('listing.video.heading')} backHref="/listing/new/studio?d={draftId}">
  {#snippet children()}
    <input
      bind:this={fileInput}
      type="file"
      accept="video/*"
      capture="environment"
      class="k-visually-hidden"
      onchange={onFileChosen}
    />

    {#if video && videoUrl}
      <div class="video-preview">
        <!-- svelte-ignore a11y_media_has_caption -->
        <video src={videoUrl} controls></video>
        {#if !locked}
          <Button size="sm" variant="ghost" onclick={remove} tooltip={tooltip('tooltip.removeVideo')}>{t('listing.video.remove')}</Button>
        {/if}
      </div>
    {:else if !locked}
      <button type="button" class="add-video" onclick={openPicker}>
        <Icon name="video" />
        {t('listing.video.add')}
      </button>
    {/if}

    {#if locked}
      <p class="video-hint" role="status">{t('listing.video.locked')}</p>
    {:else}
      <p class="video-hint">{t('listing.video.hint')}</p>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" onclick={next} tooltip={tooltip(video ? 'tooltip.next' : 'tooltip.skip')}>{video ? t('action.next') : t('action.skip')}</Button>
  {/snippet}
</ListingStep>

<style>
  .video-preview {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    align-items: start;
  }

  .video-preview video {
    inline-size: 100%;
    border-radius: var(--k-radius-md);
  }

  .add-video {
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

  .video-hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
