// packages/offline/src/outbox.ts
//
// Queue operations. Reads and writes; drain.ts drains the queue.

import { db, type OutboxEntry, type OutboxKind, type OutboxStatus } from './db';

/** Order shown in the UI and used by the drainer: oldest intent first. */
export function queuedEntries(): Promise<OutboxEntry[]> {
  return db.outbox.orderBy('createdAt').toArray();
}

export function entriesWithStatus(status: OutboxStatus): Promise<OutboxEntry[]> {
  return db.outbox.where('status').equals(status).sortBy('createdAt');
}

export interface OutboxCounts {
  total: number;
  pending: number;
  syncing: number;
  failed: number;
  needsAttention: number;
  blocked: number;
}

export async function outboxCounts(): Promise<OutboxCounts> {
  const entries = await db.outbox.toArray();
  const counts: OutboxCounts = {
    total: entries.length,
    pending: 0,
    syncing: 0,
    failed: 0,
    needsAttention: 0,
    blocked: 0,
  };
  for (const entry of entries) counts[entry.status] += 1;
  return counts;
}

export interface EnqueueInput {
  kind: OutboxKind;
  payload: unknown;
  draftId?: string;
  mediaIds?: string[];
  /** Outbox entry ids that must be delivered before this one can send. */
  dependsOn?: string[];
  /** Artisan a field agent is acting for; see OutboxEntry.onBehalfOf. */
  onBehalfOf?: string;
}

/**
 * Add an intent to the queue. The idempotency key is minted here, once, and
 * reused for every retry of this entry -- that is the whole point of it. A key
 * generated at send time would defeat itself.
 */
export async function enqueue(input: EnqueueInput): Promise<OutboxEntry> {
  const now = Date.now();
  const entry: OutboxEntry = {
    id: crypto.randomUUID(),
    kind: input.kind,
    payload: input.payload,
    status: 'pending',
    draftId: input.draftId,
    mediaIds: input.mediaIds ?? [],
    attempts: 0,
    createdAt: now,
    updatedAt: now,
    idempotencyKey: crypto.randomUUID(),
    dependsOn: input.dependsOn ?? [],
    onBehalfOf: input.onBehalfOf,
  };
  await db.outbox.add(entry);
  return entry;
}

/**
 * Move failed/needsAttention entries back to pending so the drainer picks
 * them up. `blocked` entries are deliberately left alone: they failed because
 * the server rejected their content, and retrying unchanged content just
 * fails again, more slowly.
 */
export async function retryFailed(): Promise<number> {
  return db.outbox
    .where('status')
    .anyOf('failed', 'needsAttention')
    .modify({ status: 'pending', updatedAt: Date.now(), lastError: undefined });
}

/** True if any outbox entry (any status) still references this media id. */
export async function isMediaReferenced(mediaId: string): Promise<boolean> {
  const count = await db.outbox.filter((o) => o.mediaIds.includes(mediaId)).count();
  return count > 0;
}

export async function markStatus(
  id: string,
  status: OutboxStatus,
  lastError?: string,
): Promise<void> {
  await db.outbox.update(id, { status, lastError, updatedAt: Date.now() });
}

/** Remove a delivered intent and any media blobs nothing else still needs. */
export async function discard(id: string): Promise<void> {
  await db.transaction('rw', db.outbox, db.media, async () => {
    const entry = await db.outbox.get(id);
    if (!entry) return;
    await db.outbox.delete(id);
    for (const mediaId of entry.mediaIds) {
      if (!(await isMediaReferenced(mediaId))) await db.media.delete(mediaId);
    }
  });
}

/**
 * True once every id in `dependsOn` has been delivered. A dependency is
 * delivered when its outbox row no longer exists -- `discard()` is the only
 * thing that removes a row, and it only runs after a successful send.
 */
export async function dependencySatisfied(entry: OutboxEntry): Promise<boolean> {
  if (entry.dependsOn.length === 0) return true;
  for (const depId of entry.dependsOn) {
    if ((await db.outbox.get(depId)) !== undefined) return false;
  }
  return true;
}
