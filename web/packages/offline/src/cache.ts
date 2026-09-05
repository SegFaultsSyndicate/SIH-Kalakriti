// packages/offline/src/cache.ts
//
// Read-only snapshots for offline reads (a listing, a search page, the craft
// ontology). Keyed generically -- what a key means is the caller's problem.

import { db } from './db';

export async function getCached<T = unknown>(key: string): Promise<T | undefined> {
  const row = await db.cache.get(key);
  return row?.data as T | undefined;
}

export async function setCached(key: string, data: unknown): Promise<void> {
  await db.cache.put({ key, data, fetchedAt: Date.now() });
}
