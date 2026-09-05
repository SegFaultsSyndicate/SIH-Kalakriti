// packages/offline/src/storage.ts
//
// Storage pressure handling. A phone with 4GB free and a day of unsynced
// photos will hit its IndexedDB quota; the browser then evicts the whole
// origin's storage unless it was asked to persist it.

import { db } from './db';
import { isMediaReferenced } from './outbox';

/** Ask the browser not to evict this origin's storage under pressure. */
export async function requestPersistentStorage(): Promise<boolean> {
  if (typeof navigator === 'undefined' || !navigator.storage?.persist) return false;
  return navigator.storage.persist();
}

export async function storageEstimate(): Promise<{ usage: number; quota: number } | null> {
  if (typeof navigator === 'undefined' || !navigator.storage?.estimate) return null;
  const { usage = 0, quota = 0 } = await navigator.storage.estimate();
  return { usage, quota };
}

/**
 * Drop already-uploaded media blobs, oldest first, until nothing evictable
 * remains or `limit` rows have been freed. "Evictable" means uploaded AND not
 * referenced by any live outbox entry -- a listing's own draft may still need
 * a media id it already uploaded (e.g. to redisplay a thumbnail while the
 * listing.submit entry is still queued), so upload status alone isn't enough.
 */
export async function evictOldestCachedMedia(limit = 5): Promise<number> {
  // IndexedDB keys can't be booleans, so `uploaded` can't be queried with
  // `.equals()`; filter after the index gives us capturedAt order instead.
  const uploaded = await db.media
    .orderBy('capturedAt')
    .filter((record) => record.uploaded)
    .toArray();
  let freed = 0;
  for (const record of uploaded) {
    if (freed >= limit) break;
    if (await isMediaReferenced(record.id)) continue;
    await db.media.delete(record.id);
    freed += 1;
  }
  return freed;
}

/** Call periodically (e.g. after a sync) to keep usage under `thresholdRatio` of quota. */
export async function evictIfNearQuota(thresholdRatio = 0.9): Promise<number> {
  const estimate = await storageEstimate();
  if (estimate === null || estimate.quota === 0) return 0;
  if (estimate.usage / estimate.quota < thresholdRatio) return 0;
  return evictOldestCachedMedia();
}
