// apps/artisan/src/lib/listing-draft.test.ts
import { beforeEach, describe, expect, it } from 'vitest';
import { db } from '@kalakriti/offline';
import {
  createDraft,
  patchFields,
  addCapturedMedia,
  removeCapturedMedia,
  setPrimaryPhoto,
  reorderPhotos,
  ensureListingCreateQueued,
  queueListingUpdate,
  publishListing,
} from './listing-draft';

async function resetDb(): Promise<void> {
  await db.drafts.clear();
  await db.outbox.clear();
  await db.media.clear();
}
beforeEach(resetDb);

describe('addCapturedMedia / removeCapturedMedia', () => {
  it('records the media, links it to the draft, and queues an upload', async () => {
    const draft = await createDraft();
    const media = await addCapturedMedia(draft.id, new Blob(['x']), 'image/jpeg', 'photo', 0);

    const stored = await db.drafts.get(draft.id);
    expect(stored?.mediaIds).toEqual([media.id]);

    const uploadEntries = await db.outbox.where('draftId').equals(draft.id).toArray();
    expect(uploadEntries).toHaveLength(1);
    expect(uploadEntries[0].kind).toBe('media.upload');
    expect(uploadEntries[0].payload).toEqual({ localMediaId: media.id });
  });

  describe('photo ordering', () => {
    it('persists the selected primary photo as the first uploaded image', async () => {
      const draft = await createDraft();
      const first = await addCapturedMedia(draft.id, new Blob(['first']), 'image/jpeg', 'photo', 0);
      const second = await addCapturedMedia(draft.id, new Blob(['second']), 'image/jpeg', 'photo', 1);

      await setPrimaryPhoto(draft.id, second.id);

      const stored = await db.drafts.get(draft.id);
      expect((stored?.fields as { primaryPhotoId?: string }).primaryPhotoId).toBe(second.id);
      const media = await db.media.bulkGet([first.id, second.id]);
      expect(media.find((item) => item?.id === second.id)?.order).toBe(0);
      expect(media.find((item) => item?.id === first.id)?.order).toBe(1);
    });

    it('reorders only the draft photos and rejects an incomplete order', async () => {
      const draft = await createDraft();
      const first = await addCapturedMedia(draft.id, new Blob(['first']), 'image/jpeg', 'photo', 0);
      const second = await addCapturedMedia(draft.id, new Blob(['second']), 'image/jpeg', 'photo', 1);

      await expect(reorderPhotos(draft.id, [second.id, first.id])).resolves.toBeUndefined();
      expect((await db.media.bulkGet([first.id, second.id])).map((item) => item?.order)).toEqual([1, 0]);
      await expect(reorderPhotos(draft.id, [first.id])).rejects.toThrow('photo order');
    });
  });

  it('removing media before its upload sends cancels the queued upload and deletes the blob', async () => {
    const draft = await createDraft();
    const media = await addCapturedMedia(draft.id, new Blob(['x']), 'image/jpeg', 'photo', 0);

    await removeCapturedMedia(draft.id, media.id);

    expect(await db.media.get(media.id)).toBeUndefined();
    expect(await db.outbox.where('draftId').equals(draft.id).count()).toBe(0);
    expect((await db.drafts.get(draft.id))?.mediaIds).toEqual([]);
  });
});

describe('ensureListingCreateQueued', () => {
  it('depends on every pending media.upload entry for the draft, and is a no-op the second time', async () => {
    const draft = await createDraft();
    await patchFields(draft.id, { craftId: 'weaving', workingTitle: 'Silk stole' });
    const photo = await addCapturedMedia(draft.id, new Blob(['p']), 'image/jpeg', 'photo', 0);
    const voice = await addCapturedMedia(draft.id, new Blob(['v']), 'audio/webm', 'voice');

    await ensureListingCreateQueued(draft.id);

    const entries = await db.outbox.where('draftId').equals(draft.id).toArray();
    const createEntry = entries.find((e) => e.kind === 'listing.create');
    const uploadEntries = entries.filter((e) => e.kind === 'media.upload');

    expect(createEntry).toBeDefined();
    expect(createEntry?.dependsOn.sort()).toEqual(uploadEntries.map((e) => e.id).sort());
    expect(createEntry?.payload).toMatchObject({
      draftId: draft.id,
      localMediaIds: [photo.id],
      localVoiceNoteMediaId: voice.id,
      body: { craft_id: 'weaving', working_title: 'Silk stole', min_order_quantity: 1 },
    });

    // A second call before the first has drained must not queue a duplicate.
    await ensureListingCreateQueued(draft.id);
    const createEntriesAfter = (await db.outbox.where('draftId').equals(draft.id).toArray()).filter(
      (e) => e.kind === 'listing.create',
    );
    expect(createEntriesAfter).toHaveLength(1);
  });

  it('does nothing once the draft already has a remote id', async () => {
    const draft = await createDraft();
    await db.drafts.update(draft.id, { remoteId: 'listing-1' });
    await ensureListingCreateQueued(draft.id);
    expect(await db.outbox.where('draftId').equals(draft.id).count()).toBe(0);
  });
});

describe('queueListingUpdate / publishListing', () => {
  it('refuses to queue publication until the draft has been explicitly approved', async () => {
    const draft = await createDraft();
    await patchFields(draft.id, { craftId: 'weaving' });
    await expect(publishListing(draft.id)).rejects.toThrow('explicit approval');
    expect(await db.outbox.where('draftId').equals(draft.id).count()).toBe(0);
  });

  it('an update queued before create resolves depends on the create entry', async () => {
    const draft = await createDraft();
    await patchFields(draft.id, { craftId: 'weaving', reviewApproved: true });
    await ensureListingCreateQueued(draft.id);
    const createEntry = (await db.outbox.where('draftId').equals(draft.id).toArray()).find(
      (e) => e.kind === 'listing.create',
    );

    await queueListingUpdate(draft.id, { price: { amount_paise: 50_000 } });

    const updateEntry = (await db.outbox.where('draftId').equals(draft.id).toArray()).find(
      (e) => e.kind === 'listing.update',
    );
    expect(updateEntry?.dependsOn).toEqual([createEntry?.id]);
  });

  it('publish chains submit after every create/update, and approve after submit', async () => {
    const draft = await createDraft();
    await patchFields(draft.id, { craftId: 'weaving', reviewApproved: true });
    await ensureListingCreateQueued(draft.id);
    await queueListingUpdate(draft.id, { price: { amount_paise: 50_000 } });

    await publishListing(draft.id);

    const entries = await db.outbox.where('draftId').equals(draft.id).toArray();
    const create = entries.find((e) => e.kind === 'listing.create')!;
    const update = entries.find((e) => e.kind === 'listing.update')!;
    const submit = entries.find((e) => e.kind === 'listing.submit')!;
    const approve = entries.find((e) => e.kind === 'listing.approve')!;

    expect(submit.dependsOn.sort()).toEqual([create.id, update.id].sort());
    expect(approve.dependsOn).toEqual([submit.id]);
  });
});
