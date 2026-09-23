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
  attachListingMedia,
  submitListing,
  approveListing,
  generateUploadUrl,
  confirmUpload,
  respondToLot,
  logOfflineSale,
  setIncomeBaseline,
  type SetIncomeBaselineBody,
  type LogOfflineSaleBody,
  ApiError,
  messageKeyFor,
  setAccessToken,
  setRefreshToken,
  session,
} from '@kalakriti/api';
import { locale } from '@kalakriti/i18n';
import { setArtisanId, type RegisterBody } from './registration';

export async function sendOutboxEntry(entry: OutboxEntry): Promise<SendResult> {
  switch (entry.kind) {
    case 'profile.update':
      return sendProfileUpdate(entry);
    case 'media.upload':
      return sendMediaUpload(entry);
    case 'listing.create':
      return sendListingCreate(entry);
    case 'listing.media.attach':
      return sendListingMediaAttach(entry);
    case 'listing.update':
      return sendListingUpdate(entry);
    case 'listing.submit':
      return sendListingSubmit(entry);
    case 'listing.approve':
      return sendListingApprove(entry);
    case 'order.respond':
      return sendOrderRespond(entry);
    case 'income.sale':
      return sendIncomeSale(entry);
    case 'income.baseline':
      return sendIncomeBaseline(entry);
    default:
      // Retryable rather than blocked: a future batch's kind should wait for
      // that batch's real sender, not be permanently parked by this one.
      return { ok: false, retryable: true, error: 'not yet implemented' };
  }
}

/**
 * Per-entry call options. onBehalfOf is the target frozen at enqueue (F14);
 * null, not undefined, so an artisan's own entry never picks up whatever
 * artisan an agent happens to be helping when it drains.
 */
function opts(entry: OutboxEntry) {
  return { idempotencyKey: entry.idempotencyKey, onBehalfOf: entry.onBehalfOf ?? null };
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
  const body = entry.payload as RegisterBody;
  try {
    const response = await registerArtisan(body, opts(entry));
    if (response.artisan_id) await setArtisanId(response.artisan_id);
    // The token held until now was minted at OTP-verify time, before this
    // profile existed -- it has no subject and can never authenticate
    // another call. Switch to the fresh, artisan-scoped pair the same way
    // completeOtpVerification does, or every request after this one 400s
    // with "artisan_id is not a valid uuid".
    if (response.access_token) {
      setAccessToken(response.access_token);
      setRefreshToken(response.refresh_token);
      session.establish(response.access_token);
    }
    return { ok: true };
  } catch (cause) {
    // No DEV fallback here (unlike other outbox senders): setArtisanId
    // writes the app's own identity, and every subsequent authenticated
    // request -- follower-count, artisans/me, listing creation -- carries
    // whatever id was last set here. A fabricated `artisan-${Date.now()}`
    // on a genuinely failed registration used to get treated as success
    // and stored as that identity, so every later real request the app
    // made was for an id no backend row would ever match: a fake success
    // that manufactures real, confusing failures several steps later. A
    // failed registration reports as failed, in every environment.
    //
    // VITE_USE_MOCKS=1 is the deliberate, opt-in exception: with no backend
    // reachable there is no real row to mismatch yet, so this stores an
    // obviously-fake `mock:<uuid>` id (not `artisan-${Date.now()}`, so it's
    // unmistakable in devtools) instead of rolling back to unregistered.
    // Clear IndexedDB/localStorage before pointing this app at a real
    // backend, or that fake id will 404/403 on every real authenticated call.
    if (import.meta.env.VITE_USE_MOCKS === '1') {
      console.warn('[mock fallback] registerArtisan:', cause);
      await setArtisanId(`mock:${crypto.randomUUID()}`);
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
      opts(entry),
    );
    if (!media_id || !upload_url) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };

    const put = await fetch(upload_url, {
      method: 'PUT',
      body: media.blob,
      headers: { 'Content-Type': media.mimeType },
    });
    if (!put.ok) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };

    await confirmUpload(media_id, { onBehalfOf: entry.onBehalfOf ?? null });
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
      opts(entry),
    );
    if (response.listing_id) await db.drafts.update(draftId, { remoteId: response.listing_id, updatedAt: Date.now() });
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

interface ListingMediaAttachPayload {
  draftId: string;
  items: { localMediaId: string; ordinal: number; role: 'PRIMARY_IMAGE' | 'GALLERY' | 'PROCESS_VIDEO' }[];
}

/** Triggers the cataloguing pipeline server-side -- see listing-draft.ts's ensureListingMediaAttachQueued. */
async function sendListingMediaAttach(entry: OutboxEntry): Promise<SendResult> {
  const { draftId, items } = entry.payload as ListingMediaAttachPayload;
  const listingId = await remoteListingId(draftId);
  if (!listingId) return { ok: false, retryable: true, error: locale.t('api.error.unknown') };

  try {
    const media = await db.media.bulkGet(items.map((i) => i.localMediaId));
    const remoteItems = items.map((item, i) => ({
      media_id: media[i]?.remoteId,
      ordinal: item.ordinal,
      role: item.role,
    }));
    if (remoteItems.some((i) => !i.media_id)) {
      // dependsOn should make this unreachable; treat as transient rather
      // than blocked, since the media entry may simply not have drained yet.
      return { ok: false, retryable: true, error: locale.t('api.error.unknown') };
    }

    await attachListingMedia(
      listingId,
      { items: remoteItems as Parameters<typeof attachListingMedia>[1]['items'] },
      opts(entry),
    );
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
    await updateListing(listingId, body as Parameters<typeof updateListing>[1], opts(entry));
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
    await submitListing(listingId, opts(entry));
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
    await respondToLot(lotId, body as Parameters<typeof respondToLot>[1], opts(entry));
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
    await approveListing(listingId, undefined, opts(entry));
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

/** F13: a sale logged offline; the body carries its own client_id so a replay is a no-op server-side too. */
async function sendIncomeSale(entry: OutboxEntry): Promise<SendResult> {
  try {
    await logOfflineSale(entry.payload as LogOfflineSaleBody, opts(entry));
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}

/** F13: the before-Kalakriti income from the register step; a PUT, so a replay just rewrites it. */
async function sendIncomeBaseline(entry: OutboxEntry): Promise<SendResult> {
  try {
    await setIncomeBaseline(entry.payload as SetIncomeBaselineBody, opts(entry));
    return { ok: true };
  } catch (cause) {
    return fromApiError(cause);
  }
}
