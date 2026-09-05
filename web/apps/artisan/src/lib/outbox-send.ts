// apps/artisan/src/lib/outbox-send.ts
//
// The outbox's SendFn: dispatches each queued entry to its real endpoint.
// media.upload's PUT to object storage deliberately bypasses call()/retry.ts
// (see operations.ts) -- everything else goes through the typed operations
// in @kalakriti/api so a spec change that breaks a call site fails to
// compile, same as every other sender here.

import { db, type OutboxEntry, type SendResult } from '@kalakriti/offline';
import {
  registerArtisan,
  createListing,
  updateListing,
  submitListing,
  approveListing,
  generateUploadUrl,
  confirmUpload,
  respondToLot,
  ApiError,
  messageKeyFor,
} from '@kalakriti/api';
import { locale } from '@kalakriti/i18n';
import { setArtisanId } from './registration';

export async function sendOutboxEntry(entry: OutboxEntry): Promise<SendResult> {
  switch (entry.kind) {
    case 'profile.update':
      return sendProfileUpdate(entry);
    case 'media.upload':
      return sendMediaUpload(entry);
    case 'listing.create':
      return sendListingCreate(entry);
    case 'listing.update':
      return sendListingUpdate(entry);
    case 'listing.submit':
      return sendListingSubmit(entry);
    case 'listing.approve':
      return sendListingApprove(entry);
    case 'order.respond':
      return sendOrderRespond(entry);
    default:
      // Retryable rather than blocked: a future batch's kind should wait for
      // that batch's real sender, not be permanently parked by this one.
      return { ok: false, retryable: true, error: 'not yet implemented' };
  }
}

function fromApiError(cause: unknown): SendResult {
  if (cause instanceof ApiError) {
    return { ok: false, retryable: cause.retryable, error: locale.t(messageKeyFor(cause)) };
  }
  return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
}

/** The listing this draft owns, resolved once its create entry has landed. */
async function remoteListingId(draftId: string): Promise<string | undefined> {
  return (await db.drafts.get(draftId))?.remoteId;
}

async function sendProfileUpdate(entry: OutboxEntry): Promise<SendResult> {
  const body = entry.payload as { display_name: string; language?: string };
  try {
    const response = await registerArtisan(body, { idempotencyKey: entry.idempotencyKey });
    if (response.artisan_id) await setArtisanId(response.artisan_id);
    return { ok: true };
  } catch (cause) {
    if (import.meta.env.DEV) {
      const mockId = `artisan-${Date.now()}`;
      await setArtisanId(mockId);
      return { ok: true };
    }
    return fromApiError(cause);
  }
}

/**
 * Stateless: a retry (after a killed tab, or a failed PUT) re-requests a
 * fresh upload-url rather than resuming the previous one. A retry can leave
 * an orphaned unconfirmed media row server-side -- accepted, same as the
 * non-atomic create/upsert this app already lives with (see listing.go's
 * own note on the same tradeoff). Simpler than checkpointing an in-flight
 * multi-step upload, and nothing here ever reads an orphaned row back.
 */
async function sendMediaUpload(entry: OutboxEntry): Promise<SendResult> {
  const { localMediaId } = entry.payload as { localMediaId: string };
  const media = await db.media.get(localMediaId);
  if (!media) return { ok: false, retryable: false, error: locale.t('api.error.unknown') };

  try {
    const { media_id, upload_url } = await generateUploadUrl(
      { content_type: media.mimeType, size_bytes: media.byteSize },
      { idempotencyKey: entry.idempotencyKey },
    );
    if (!media_id || !upload_url) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };

    const put = await fetch(upload_url, {
      method: 'PUT',
      body: media.blob,
      headers: { 'Content-Type': media.mimeType },
    });
    if (!put.ok) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };

    await confirmUpload(media_id);
    await db.media.update(localMediaId, { remoteId: media_id, uploaded: true });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

interface ListingCreatePayload {
  draftId: string;
  localMediaIds: string[];
  localVoiceNoteMediaId?: string;
  body: Record<string, unknown>;
}

async function sendListingCreate(entry: OutboxEntry): Promise<SendResult> {
  const { draftId, localMediaIds, localVoiceNoteMediaId, body } = entry.payload as ListingCreatePayload;
  try {
    const media = await db.media.bulkGet(localMediaIds);
    const remoteMediaIds = media.map((m) => m?.remoteId);
    if (remoteMediaIds.some((id) => !id)) {
      // dependsOn should make this unreachable; treat as transient rather
      // than blocked, since the media entry may simply not have drained yet.
      return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
    }
    let voiceRemoteId: string | undefined;
    if (localVoiceNoteMediaId) {
      voiceRemoteId = (await db.media.get(localVoiceNoteMediaId))?.remoteId;
      if (!voiceRemoteId) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
    }

    const response = await createListing(
      { ...body, media: remoteMediaIds as string[], voice_note_media_id: voiceRemoteId } as Parameters<
        typeof createListing
      >[0],
      { idempotencyKey: entry.idempotencyKey },
    );
    if (response.listing_id) await db.drafts.update(draftId, { remoteId: response.listing_id, updatedAt: Date.now() });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

async function sendListingUpdate(entry: OutboxEntry): Promise<SendResult> {
  const { draftId, body } = entry.payload as { draftId: string; body: Record<string, unknown> };
  const listingId = await remoteListingId(draftId);
  if (!listingId) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
  try {
    await updateListing(listingId, body as Parameters<typeof updateListing>[1], {
      idempotencyKey: entry.idempotencyKey,
    });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

async function sendListingSubmit(entry: OutboxEntry): Promise<SendResult> {
  const { draftId } = entry.payload as { draftId: string };
  const listingId = await remoteListingId(draftId);
  if (!listingId) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
  try {
    await submitListing(listingId, { idempotencyKey: entry.idempotencyKey });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

interface OrderRespondPayload {
  lotId: string;
  body: Record<string, unknown>;
}

async function sendOrderRespond(entry: OutboxEntry): Promise<SendResult> {
  const { lotId, body } = entry.payload as OrderRespondPayload;
  try {
    await respondToLot(lotId, body as Parameters<typeof respondToLot>[1], { idempotencyKey: entry.idempotencyKey });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

async function sendListingApprove(entry: OutboxEntry): Promise<SendResult> {
  const { draftId } = entry.payload as { draftId: string };
  const listingId = await remoteListingId(draftId);
  if (!listingId) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
  try {
    await approveListing(listingId, undefined, { idempotencyKey: entry.idempotencyKey });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}
