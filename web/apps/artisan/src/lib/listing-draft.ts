// apps/artisan/src/lib/listing-draft.ts
//
// The listing wizard's Dexie-backed draft: one row per in-progress listing,
// local field state across all 8 steps, plus the outbox entries that carry
// it to the server. Every step writes to the draft directly (no network) --
// media, create, update, submit and approve are the only things queued,
// each addressed by draftId rather than a remote id, because the remote
// listing (and even the remote media ids) may not exist yet when the
// artisan is still offline. outbox-send.ts resolves draftId -> remoteId at
// send time, once dependsOn guarantees the earlier step has landed.

import { liveQuery } from 'dexie';
import { db, enqueue, type DraftRecord, type MediaRecord } from '@kalakriti/offline';

export type ListingWizardStep =
  | 'capture'
  | 'studio'
  | 'video'
  | 'story'
  | 'processing'
  | 'review'
  | 'pricing'
  | 'terms'
  | 'done';

export interface StudioConfig {
  backgroundMode?: 'white' | 'transparent' | 'natural';
  autoLightingApplied?: boolean;
  brightnessOffset?: number;
  contrastOffset?: number;
}

export interface Dimensions {
  length_mm?: number;
  width_mm?: number;
  height_mm?: number;
  weight_g?: number;
}

export interface MadeToOrderTerms {
  lead_time_days?: number;
  capacity_per_month?: number;
  accepting_orders?: boolean;
  advance_pct?: number;
}

export interface ListingTranslation {
  language: string;
  title: string;
  description?: string;
  highlights?: string[];
  machine_generated?: boolean;
}

/** Everything a step can set on the draft. Free-form until submit, per DraftRecord.fields' own contract. */
export interface ListingDraftFields {
  step?: ListingWizardStep;
  craftId?: string;
  workingTitle?: string;
  dimensions?: Dimensions;
  type?: 'READY_STOCK' | 'MADE_TO_ORDER';
  priceAmountPaise?: number;
  stockQuantity?: number;
  minOrderQuantity?: number;
  madeToOrderTerms?: MadeToOrderTerms;
  packaging?: { fragile?: boolean; oversized?: boolean; requires_custom_crating?: boolean };
  translations?: ListingTranslation[];
  studioConfig?: StudioConfig;
  /**
   * Cached copy of GET /listings/{id}/attributes -- what the cataloguing
   * pipeline (triggered by ensureListingMediaAttachQueued below) actually
   * inferred, or what the artisan has since overridden server-side. Kept
   * locally too so the review screen has something to show offline right
   * after enrichment.
   */
  attributes?: {
    name: string;
    value: string;
    confidence: number;
    source: 'MODEL' | 'ARTISAN' | 'CURATOR';
  }[];
  /** Local selection used to keep the artisan's chosen primary image first. */
  primaryPhotoId?: string;
  reviewApproved?: boolean;
  termsAccepted?: boolean;
}

function fieldsOf(draft: DraftRecord): ListingDraftFields {
  return draft.fields as ListingDraftFields;
}

export async function createDraft(): Promise<DraftRecord> {
  const now = Date.now();
  const draft: DraftRecord = {
    id: crypto.randomUUID(),
    fields: { step: 'capture' } satisfies ListingDraftFields,
    mediaIds: [],
    createdAt: now,
    updatedAt: now,
  };
  await db.drafts.add(draft);
  return draft;
}

export function getDraft(id: string): Promise<DraftRecord | undefined> {
  return db.drafts.get(id);
}

/** Newest first, for the "resume a draft" list on the home screen. */
export function listDrafts(): Promise<DraftRecord[]> {
  return db.drafts.orderBy('updatedAt').reverse().toArray();
}

export function watchDraft(id: string, onChange: (draft: DraftRecord | undefined) => void): () => void {
  const sub = liveQuery(() => db.drafts.get(id)).subscribe(onChange);
  return () => sub.unsubscribe();
}

/** Merges into the draft's fields immediately -- the autosave every step's onchange calls. */
export async function patchFields(id: string, patch: Partial<ListingDraftFields>): Promise<void> {
  const draft = await db.drafts.get(id);
  if (!draft) return;
  await db.drafts.update(id, { fields: { ...fieldsOf(draft), ...patch }, updatedAt: Date.now() });
}

/** Removes the draft and any media nothing else (an in-flight outbox entry) still needs. */
export async function deleteDraft(id: string): Promise<void> {
  await db.transaction('rw', db.drafts, db.media, db.outbox, async () => {
    const draft = await db.drafts.get(id);
    if (!draft) return;
    await db.drafts.delete(id);
    for (const mediaId of draft.mediaIds) {
      const stillQueued = await db.outbox.filter((e) => e.mediaIds.includes(mediaId)).count();
      if (stillQueued === 0) await db.media.delete(mediaId);
    }
  });
}

/**
 * Records a captured photo/video/voice blob and queues its upload. Returns
 * immediately with a local id the capture screen can render via
 * `URL.createObjectURL` -- nothing here waits for a connection.
 */
export async function addCapturedMedia(
  draftId: string,
  blob: Blob,
  mimeType: string,
  kind: 'photo' | 'video' | 'voice',
  order?: number,
): Promise<MediaRecord> {
  const media: MediaRecord = {
    id: crypto.randomUUID(),
    blob,
    mimeType,
    byteSize: blob.size,
    uploaded: false,
    capturedAt: Date.now(),
    kind,
    order,
  };
  await db.media.add(media);
  const draft = await db.drafts.get(draftId);
  if (draft) await db.drafts.update(draftId, { mediaIds: [...draft.mediaIds, media.id], updatedAt: Date.now() });
  await enqueue({ kind: 'media.upload', draftId, mediaIds: [media.id], payload: { localMediaId: media.id } });
  return media;
}

/** Retake/remove: drops the media from the draft and cancels its upload if it hasn't sent yet. */
export async function removeCapturedMedia(draftId: string, mediaId: string): Promise<void> {
  const draft = await db.drafts.get(draftId);
  if (draft) {
    await db.drafts.update(draftId, {
      mediaIds: draft.mediaIds.filter((id) => id !== mediaId),
      updatedAt: Date.now(),
    });
  }

  const pending = await db.outbox
    .filter((e) => e.mediaIds.includes(mediaId) && e.status !== 'syncing')
    .toArray();
  await Promise.all(pending.map((e) => db.outbox.delete(e.id)));
  const stillQueued = await db.outbox.filter((e) => e.mediaIds.includes(mediaId)).count();
  if (stillQueued === 0) await db.media.delete(mediaId);
}

/** Reorders the captured photos and keeps the selected order in IndexedDB. */
export async function reorderPhotos(draftId: string, orderedPhotoIds: string[]): Promise<void> {
  await db.transaction('rw', db.drafts, db.media, async () => {
    const draft = await db.drafts.get(draftId);
    if (!draft) throw new Error('draft not found');
    const media = (await db.media.bulkGet(draft.mediaIds)).filter(
      (item): item is MediaRecord => !!item && item.kind === 'photo',
    );
    const existing = new Set(media.map((item) => item.id));
    if (
      orderedPhotoIds.length !== media.length ||
      orderedPhotoIds.some((id) => !existing.has(id)) ||
      new Set(orderedPhotoIds).size !== orderedPhotoIds.length
    ) {
      throw new Error('photo order must include every captured photo exactly once');
    }
    await Promise.all(orderedPhotoIds.map((id, order) => db.media.update(id, { order })));
    await db.drafts.update(draftId, {
      mediaIds: [...orderedPhotoIds, ...draft.mediaIds.filter((id) => !existing.has(id))],
      updatedAt: Date.now(),
    });
  });
}

/** Makes one captured photo primary by moving it to the first upload position. */
export async function setPrimaryPhoto(draftId: string, mediaId: string): Promise<void> {
  const draft = await db.drafts.get(draftId);
  if (!draft) throw new Error('draft not found');
  const photos = (await db.media.bulkGet(draft.mediaIds))
    .filter((item): item is MediaRecord => !!item && item.kind === 'photo')
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
  if (!photos.some((photo) => photo.id === mediaId)) throw new Error('primary photo must be captured on this draft');
  await reorderPhotos(draftId, [mediaId, ...photos.filter((photo) => photo.id !== mediaId).map((photo) => photo.id)]);
  await patchFields(draftId, { primaryPhotoId: mediaId });
}

async function pendingEntries(draftId: string, kind: string): Promise<{ id: string }[]> {
  return db.outbox.where('draftId').equals(draftId).and((e) => e.kind === kind).toArray();
}

/**
 * Queues listing creation once, from whatever the draft has so far (craft,
 * title, media). Price/type/translations are filled in later via
 * queueListingUpdate -- see openapi.json's CreateListingRequest, where only
 * craft_id is actually required server-side. A second call is a no-op: it's
 * called every time the wizard leaves the story step, but only the first
 * actually queues (checked by draft.remoteId and by an existing outbox row,
 * since remoteId isn't set until the first one lands).
 */
export async function ensureListingCreateQueued(draftId: string): Promise<void> {
  const draft = await db.drafts.get(draftId);
  if (!draft || draft.remoteId) return;
  if ((await pendingEntries(draftId, 'listing.create')).length > 0) return;

  const media = await db.media.bulkGet(draft.mediaIds);
  const nonVoice = media
    .filter((m): m is MediaRecord => !!m && m.kind !== 'voice')
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
  const voice = media.find((m): m is MediaRecord => !!m && m.kind === 'voice');
  const localMediaIds = nonVoice.map((m) => m.id);
  const mediaEntries = await pendingEntries(draftId, 'media.upload');

  const f = fieldsOf(draft);
  await enqueue({
    kind: 'listing.create',
    draftId,
    mediaIds: [...localMediaIds, ...(voice ? [voice.id] : [])],
    dependsOn: mediaEntries.map((e) => e.id),
    payload: {
      draftId,
      localMediaIds,
      localVoiceNoteMediaId: voice?.id,
      body: {
        craft_id: f.craftId ?? '',
        working_title: f.workingTitle,
        dimensions: f.dimensions,
        min_order_quantity: f.minOrderQuantity ?? 1,
      },
    },
  });
}

/**
 * Queues attaching this draft's captured photos (and process video, if any)
 * to the listing's own media set -- separate from the product-level media
 * CreateListing already sent, and what actually triggers the cataloguing
 * pipeline (enhance/extract attributes/generate description) server-side.
 * Depends on the create entry (needs the remote listing id) and every
 * media.upload (needs each photo's remote media id), same dependency shape
 * as ensureListingCreateQueued. A second call is a no-op, same guard.
 */
export async function ensureListingMediaAttachQueued(draftId: string): Promise<void> {
  const draft = await db.drafts.get(draftId);
  if (!draft) return;
  if ((await pendingEntries(draftId, 'listing.media.attach')).length > 0) return;

  const media = await db.media.bulkGet(draft.mediaIds);
  const photos = media
    .filter((m): m is MediaRecord => !!m && m.kind === 'photo')
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
  const video = media.find((m): m is MediaRecord => !!m && m.kind === 'video');
  if (photos.length === 0 && !video) return;

  const f = fieldsOf(draft);
  const items = [
    ...photos.map((p, i) => ({
      localMediaId: p.id,
      ordinal: i,
      role: p.id === f.primaryPhotoId ? ('PRIMARY_IMAGE' as const) : ('GALLERY' as const),
    })),
    ...(video ? [{ localMediaId: video.id, ordinal: photos.length, role: 'PROCESS_VIDEO' as const }] : []),
  ];

  const createEntry = (await pendingEntries(draftId, 'listing.create'))[0];
  const mediaEntries = await pendingEntries(draftId, 'media.upload');
  await enqueue({
    kind: 'listing.media.attach',
    draftId,
    mediaIds: items.map((i) => i.localMediaId),
    dependsOn: [...(createEntry ? [createEntry.id] : []), ...mediaEntries.map((e) => e.id)],
    payload: { draftId, items },
  });
}

/**
 * Queues a PATCH against the listing this draft becomes -- pricing and terms.
 * Depends on the create entry so it can never arrive first even if the
 * artisan finishes the whole wizard offline in one sitting.
 */
export async function queueListingUpdate(
  draftId: string,
  body: Record<string, unknown>,
): Promise<void> {
  const createEntry = (await pendingEntries(draftId, 'listing.create'))[0];
  await enqueue({
    kind: 'listing.update',
    draftId,
    dependsOn: createEntry ? [createEntry.id] : [],
    payload: { draftId, body },
  });
}

/**
 * The wizard's final action: submit for the artisan's own review, then
 * approve. Two real backend steps collapsed into one client action, since
 * this app never shows the DRAFT/PENDING split to anyone but the artisan
 * who both submits and approves it. Depends on every create/media-attach/
 * update queued so far -- in particular the media attach, since core-svc
 * refuses to submit a listing with no copy to review, and the copy only
 * exists once the pipeline the attach triggers has run.
 */
export async function publishListing(draftId: string): Promise<void> {
  const draft = await db.drafts.get(draftId);
  if (!draft || !(fieldsOf(draft).reviewApproved === true)) {
    throw new Error('listing requires explicit approval before publication');
  }
  const blockers = [
    ...(await pendingEntries(draftId, 'listing.create')),
    ...(await pendingEntries(draftId, 'listing.media.attach')),
    ...(await pendingEntries(draftId, 'listing.update')),
  ];
  const submitEntry = await enqueue({
    kind: 'listing.submit',
    draftId,
    dependsOn: blockers.map((e) => e.id),
    payload: { draftId },
  });
  await enqueue({
    kind: 'listing.approve',
    draftId,
    dependsOn: [submitEntry.id],
    payload: { draftId },
  });
}
