// apps/artisan/src/lib/provenance.ts
//
// Sealing a listing's provenance: freezing process-evidence media, the
// technique check, and (for textiles) the loom verdict into a permanent
// record. POST /listings/{id}/seal-provenance is real -- see ml_wiring.md's
// batch 9 section for what BFF wiring this batch added and why it wasn't
// there already. Called directly, online-only, same as pricing advisory:
// sealing is a rare, deliberate, one-time action, not something to queue
// silently for a later connection.
//
// MOCK: deriveVerdictOutcome's confidence threshold. The real model
// (services/ml-svc) only ever returns matches:boolean + confidence:float,
// with no INSUFFICIENT_EVIDENCE state of its own -- this function's 0.55
// cutoff is this app's own judgement call standing in for one. See
// ml_wiring.md before changing or removing it.

import { sealProvenance, generateUploadUrl, confirmUpload, type components } from '@kalakriti/api';

export type SealResponse = components['schemas']['ProvenanceRecord'];
export type TechniqueVerdict = components['schemas']['TechniqueVerdict'];
export type LoomVerdict = components['schemas']['LoomVerdict'];

export type VerdictOutcome = 'MATCH' | 'MISMATCH' | 'INSUFFICIENT_EVIDENCE';

/** Confidence below this reads as inconclusive regardless of `matches` -- see this file's header. */
const CONFIDENCE_FLOOR = 0.55;

export function deriveVerdictOutcome(verdict: TechniqueVerdict): VerdictOutcome {
  const confidence = verdict.confidence ?? 0;
  if (confidence < CONFIDENCE_FLOOR) return 'INSUFFICIENT_EVIDENCE';
  return verdict.matches ? 'MATCH' : 'MISMATCH';
}

/**
 * Uploads one piece of process evidence and returns its remote media id.
 * Mirrors outbox-send.ts's sendMediaUpload, but runs live rather than
 * through the outbox: sealing is online-only and immediate (see this file's
 * header), so there is no queued intent to resolve a media id for later.
 */
export async function uploadEvidence(blob: Blob, mimeType: string): Promise<string> {
  const { media_id, upload_url } = await generateUploadUrl({ content_type: mimeType, size_bytes: blob.size });
  if (!media_id || !upload_url) throw new Error('upload-url response missing media_id/upload_url');

  const put = await fetch(upload_url, { method: 'PUT', body: blob, headers: { 'Content-Type': mimeType } });
  if (!put.ok) throw new Error(`evidence upload failed: ${put.status}`);

  await confirmUpload(media_id);
  return media_id;
}

export interface SealInput {
  mediaIds: string[];
  claimedTechnique: string;
  skipLoomCheck?: boolean;
}

export async function seal(listingId: string, input: SealInput): Promise<SealResponse> {
  return sealProvenance(listingId, {
    media: input.mediaIds,
    claimed_technique: input.claimedTechnique,
    skip_loom_check: input.skipLoomCheck,
  });
}

/**
 * Renders the record's qr_code (a verification URL, not an image -- see the
 * ProvenanceRecord schema) as a scannable PNG data URL. Lazy-imported: this
 * is the only screen in the app that needs a QR encoder, and the wizard's
 * size budget (vite.config.ts's ARTISAN_JS_BUDGET_KB) shouldn't carry it on
 * every route.
 */
export async function buildQrDataUrl(verificationUrl: string): Promise<string> {
  const QRCode = await import('qrcode');
  return QRCode.toDataURL(verificationUrl, { margin: 0, width: 480 });
}
