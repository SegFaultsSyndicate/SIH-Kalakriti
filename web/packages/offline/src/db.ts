// packages/offline/src/db.ts
//
// The on-device database. IndexedDB via Dexie, because the artisan app has to
// be fully usable with no network for hours at a stretch and everything the
// artisan records has to survive the tab being killed mid-capture.
//
// Five tables, and the split matters:
//
//   drafts   -- work in progress the artisan owns. Never deleted by sync.
//   outbox   -- intents waiting to reach the server, in order.
//   media    -- captured photo/video blobs, referenced by an outbox entry.
//              (the batch spec calls this "mediaQueue"; kept as `media` since
//              apps/artisan already reads this package's exports and renaming
//              a table name isn't free -- nothing in the public API exposes
//              the table name itself, so the spec's name and this name are
//              the same thing under two labels.)
//   cache    -- read-only snapshots (listings, search, craft ontology) for
//              offline reads. Keyed generically; the sync engine and screens
//              decide what a key means.
//   prefs    -- small device-local settings (theme, text size, voice). NOT
//              language: @kalakriti/i18n already persists locale to
//              localStorage under `kalakriti.locale` and owns that until a
//              later batch reconciles the two stores.
//
// Media is separate from outbox because a 4MB photo blob must not be rewritten
// every time the outbox row's attempt counter changes, and because eviction
// pressure on a 16GB phone should be able to drop an already-uploaded blob
// without touching the intent that referenced it.
//
// This file is schema and access only; drain.ts drains the outbox.

import Dexie, { type EntityTable } from 'dexie';

/**
 * Where a queued intent has got to. Never inferred from a boolean pair.
 *
 * `failed` and `needsAttention` are both retried by `retryFailed()`; `blocked`
 * is not. The difference: `failed`/`needsAttention` mean the transport or
 * gateway failed and trying the exact same bytes again later may work.
 * `blocked` means the server looked at the content and rejected it -- retrying
 * unchanged content just fails again, more slowly. Collapsing these into one
 * status would make "try sending again" stop retrying the artisan's normal
 * bad-2G failures, which is the common case, not the rare one.
 */
export type OutboxStatus =
  | 'pending' // waiting for a connection, or for a dependency to clear
  | 'syncing' // in flight right now
  | 'failed' // transport failed, will be retried with backoff
  | 'needsAttention' // retries exhausted; surfaced to the artisan, not dropped
  | 'blocked'; // server rejected the content; needs the artisan to change something

/**
 * The kinds of intent the outbox can carry. A closed union rather than a
 * string: the sync engine dispatches on it, and a typo must not become a
 * silently undeliverable row.
 */
export type OutboxKind =
  | 'listing.create'
  | 'listing.update'
  | 'listing.submit'
  | 'listing.approve'
  | 'media.upload'
  | 'order.respond'
  | 'profile.update';

export interface OutboxEntry {
  id: string;
  kind: OutboxKind;
  /**
   * The intent's payload, shaped by `kind`. Typed as unknown rather than a
   * union of request bodies because those types are GENERATED from the
   * OpenAPI spec, and this package must not hand-write them. The sync engine
   * narrows per kind at the point it builds the request.
   */
  payload: unknown;
  status: OutboxStatus;
  /** Local id of the draft this intent belongs to, for grouping in the UI. */
  draftId?: string;
  /** Ids into `media`, uploaded before the intent itself can be sent. */
  mediaIds: string[];
  attempts: number;
  createdAt: number;
  updatedAt: number;
  /** Last transport or server error, shown to the artisan as a plain reason. */
  lastError?: string;
  /**
   * Idempotency key sent with the request, so a retry after an ambiguous
   * failure cannot create a second listing.
   */
  idempotencyKey: string;
  /**
   * Ids of other outbox entries that must be delivered (and discarded) first.
   * Explicit rather than implicit ordering: e.g. a `listing.submit` entry
   * depends on that draft's `media.upload` entries, so the drainer holds it
   * back until those ids no longer exist in `outbox` (a discarded row means
   * delivered; a row still present, in any status, means not yet).
   */
  dependsOn: string[];
}

export interface DraftRecord {
  id: string;
  /** Set once the server has assigned a real listing id. */
  remoteId?: string;
  title?: string;
  /** Free-form while in progress; validated at submit, not at every keystroke. */
  fields: Record<string, unknown>;
  mediaIds: string[];
  createdAt: number;
  updatedAt: number;
  /**
   * The version the artisan last saw from the server, for conflict detection
   * against a concurrent edit elsewhere. Nothing populates this yet: `Listing`
   * (GET /listings/{id}, added Batch 8) carries no version field, so the
   * comparison in conflict.ts is real; the input to it is not, until core-svc
   * exposes one.
   */
  baseVersion?: number;
}

export interface CacheRecord {
  /** e.g. "listing:abc123", "search:q=silk", "ontology:crafts" -- caller-defined. */
  key: string;
  data: unknown;
  fetchedAt: number;
}

export interface PrefRecord {
  key: string;
  value: unknown;
  updatedAt: number;
}

export interface MediaRecord {
  id: string;
  blob: Blob;
  mimeType: string;
  byteSize: number;
  /** Set after POST /media/{id}/confirm, so eviction knows it is safe to drop. */
  uploaded: boolean;
  capturedAt: number;
  /**
   * The server's media_id, set once the media.upload outbox entry for this
   * record succeeds. A retry of that entry re-requests a fresh upload-url
   * rather than resuming this one (see outbox-send.ts), so this is written
   * once, at success, not checkpointed mid-attempt.
   */
  remoteId?: string;
  /** 0-based position among a listing's photos; the artisan-designated primary is 0. */
  order?: number;
  /** 'photo' | 'video' | 'voice', so a listing.create sender can split media[] from voice_note_media_id without re-deriving it from mimeType. */
  kind?: 'photo' | 'video' | 'voice';
}

export class KalakritiDatabase extends Dexie {
  drafts!: EntityTable<DraftRecord, 'id'>;
  outbox!: EntityTable<OutboxEntry, 'id'>;
  media!: EntityTable<MediaRecord, 'id'>;
  cache!: EntityTable<CacheRecord, 'key'>;
  prefs!: EntityTable<PrefRecord, 'key'>;

  constructor(name = 'kalakriti') {
    super(name);
    // Indexes are the queries the offline UI actually runs: the queue ordered
    // oldest-first, and counts per status. Nothing speculative.
    this.version(1).stores({
      drafts: 'id, updatedAt, remoteId',
      outbox: 'id, status, createdAt, draftId, [status+createdAt]',
      media: 'id, uploaded, capturedAt',
    });
    // v2: cache + prefs tables, dependsOn on outbox entries, baseVersion on
    // drafts, and the needsAttention status. Existing `blocked` rows are
    // untouched -- the split between "content rejected" and "retries
    // exhausted" only starts mattering for entries created after this
    // upgrade, and old blocked rows were already correctly parked.
    this.version(2)
      .stores({
        drafts: 'id, updatedAt, remoteId',
        outbox: 'id, status, createdAt, draftId, [status+createdAt]',
        media: 'id, uploaded, capturedAt',
        cache: 'key, fetchedAt',
        prefs: 'key',
      })
      .upgrade(async (tx) => {
        // v1 rows predate `dependsOn`; backfill so the field is never missing.
        await tx
          .table('outbox')
          .toCollection()
          .modify((entry: OutboxEntry) => {
            entry.dependsOn ??= [];
          });
      });
  }
}

export const db = new KalakritiDatabase();
