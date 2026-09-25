<!--
  apps/artisan/src/routes/listing/new/story/+page.svelte

  Step 3 of 7: what it is (craft, working title -- the two facts
  CreateListingRequest actually needs) and the artisan's own spoken story
  about it. Leaving this step is what kicks off listing creation and the
  media uploads in the background (ensureListingCreateQueued) -- everything
  captured so far is enough to register the listing; pricing and terms
  arrive later as PATCHes.

  ensureListingMediaAttachQueued is also queued here now, not left for the
  processing screen to queue once media.upload finishes (its own previous
  behaviour). That gap let the outbox drain the whole media.upload ->
  listing.create chain -- both entries reference the local media blob by id,
  so as long as either is still in the outbox, discard()'s reference count
  (isMediaReferenced) keeps the blob -- before anything had queued the
  eventual listing.media.attach entry that would also need it. The instant
  both drained, the last reference vanished and discard() deleted the local
  blob outright; ensureListingMediaAttachQueued then found nothing to attach
  (db.media.bulkGet came back empty) and silently no-opped, so the
  cataloguing pipeline this attach call is what actually triggers never ran
  -- not a slow pipeline, a starved one, permanently, on essentially every
  fast/local drain. Queuing it in the same synchronous breath as
  ensureListingCreateQueued (no network call and thus no drain opportunity
  in between) keeps the blob referenced continuously through the whole
  chain. Confirmed live with a real upload+drain sequence, traced via debug
  logging in ensureListingMediaAttachQueued.
-->
<script lang="ts">
  import { liveQuery } from 'dexie';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, Select, Input, FieldGroup, VoiceInput } from '@kalakriti/ui';
  import { db, type MediaRecord } from '@kalakriti/offline';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields, addCapturedMedia, removeCapturedMedia, ensureListingCreateQueued, ensureListingMediaAttachQueued } from '$lib/listing-draft';
  import { loadCrafts, type Craft } from '$lib/ontology';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let craftId = $state('');
  let workingTitle = $state('');
  let voice = $state<MediaRecord | undefined>(undefined);
  let crafts = $state<Craft[]>([]);
  $effect(() => void loadCrafts().then((c) => (crafts = c)));

  const craftOptions = $derived([
    { value: '', label: t('listing.story.craftPlaceholder') },
    ...crafts.map((craft) => ({ value: craft.id, label: craft.displayName })),
  ]);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      craftId = draft.fields.craftId as string | undefined ?? '';
      workingTitle = (draft.fields.workingTitle as string | undefined) ?? '';
    });
  });

  // SIH Demo: once crafts load, auto-select "Weaving" (or first match) and
  // pre-fill the working title from the White Saree with Maroon Border listing.
  $effect(() => {
    if (crafts.length === 0 || craftId) return;
    const weaving = crafts.find((c) =>
      c.slug.includes('weav') || c.displayName.toLowerCase().includes('weav'),
    );
    if (weaving) {
      craftId = weaving.id;
      if (!workingTitle) workingTitle = 'White Saree with Maroon Border';
    }
  });

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (!draft) return undefined;
      const media = await db.media.bulkGet(draft.mediaIds);
      return media.find((m): m is MediaRecord => !!m && m.kind === 'voice');
    }).subscribe((row) => (voice = row));
    return () => sub.unsubscribe();
  });

  async function onRecording(blob: Blob): Promise<void> {
    if (!draftId) return;
    if (voice) await removeCapturedMedia(draftId, voice.id);
    await addCapturedMedia(draftId, blob, blob.type || 'audio/webm', 'voice');
  }

  async function removeVoice(): Promise<void> {
    if (!voice) return;
    await removeCapturedMedia(draftId, voice.id);
  }

  async function next(): Promise<void> {
    if (!draftId || craftId === '') return;
    await patchFields(draftId, { craftId, workingTitle: workingTitle.trim() || undefined });
    await ensureListingCreateQueued(draftId);
    await ensureListingMediaAttachQueued(draftId);
    await goto(`/listing/new/processing?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.story.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={3} heading={t('listing.story.heading')} backHref="/listing/new/video?d={draftId}">
  {#snippet children()}
    <FieldGroup label={t('listing.story.craftLabel')}>
      {#snippet children({ id })}
        <Select {id} bind:value={craftId} options={craftOptions} />
      {/snippet}
    </FieldGroup>

    <FieldGroup label={t('listing.story.titleLabel')} optional>
      {#snippet children({ id })}
        <Input {id} bind:value={workingTitle} placeholder={t('listing.story.titlePlaceholder')} />
      {/snippet}
    </FieldGroup>

    <div class="story-voice">
      <p class="story-voice__label">{t('listing.story.voiceLabel')}</p>
      {#if voice}
        <div class="story-voice__done">
          <Icon name="success" />
          <span>{t('listing.story.voiceRecorded')}</span>
          <Button size="sm" variant="ghost" onclick={removeVoice} tooltip={tooltip('tooltip.removeVoice')}>{t('listing.story.voiceRedo')}</Button>
        </div>
      {:else}
        <VoiceInput mode="hold" onrecording={onRecording} />
      {/if}
    </div>
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={craftId === ''} onclick={next} tooltip={tooltip('tooltip.next')}>{t('action.next')} →</Button>
  {/snippet}
</ListingStep>

<style>
  .story-voice {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .story-voice__label {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .story-voice__done {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }
</style>
