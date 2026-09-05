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
import type { MessageKey } from '@kalakriti/i18n';

export type ListingWizardStep =
  | 'capture'
  | 'video'
  | 'story'
  | 'processing'
  | 'review'
  | 'pricing'
  | 'terms'
  | 'done';

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
  /** MOCK, from ml-mock.ts's pipeline result -- local-only, never sent. See ml_wiring.md. */
  attributes?: { key: string; labelKey: MessageKey; value: string; source: 'MODEL' | 'ARTISAN' }[];
  /** MOCK, from ml-mock.ts -- sentence-to-attribute pairs for the review screen's tap-highlight. */
  claims?: { sentenceIndex: number; attributeKey: string }[];
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
 * Queues a PATCH against the listing this draft becomes -- pricing, terms,
 * and (from the mock ML pipeline) translations. Depends on the create entry
 * so it can never arrive first even if the artisan finishes the whole
 * wizard offline in one sitting.
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
 * who both submits and approves it. Depends on every create/update queued
 * so far -- in particular the translations update the mock pipeline queued,
 * since core-svc refuses to submit a listing with no copy to review.
 */
export async function publishListing(draftId: string): Promise<void> {
  const blockers = [
    ...(await pendingEntries(draftId, 'listing.create')),
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
