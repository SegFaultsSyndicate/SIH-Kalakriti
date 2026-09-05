// packages/offline/src/optimistic.ts
//
// Apply a local change immediately, enqueue the intent, and roll back with a
// toast only if that intent eventually lands in needsAttention or blocked.
// A discarded row (removed on success) means nothing to roll back.

import { liveQuery } from 'dexie';
import { db, type OutboxEntry } from './db';

export interface OptimisticOptions {
  /** Mutate local state immediately. */
  apply: () => void;
  /** Undo it, called only on terminal failure. */
  rollback: () => void;
  /** Perform the enqueue; returns the row to watch. */
  enqueue: () => Promise<OutboxEntry>;
  onTerminalFailure?: (entry: OutboxEntry) => void;
}

/** Returns a disposer that stops watching without rolling back. */
export async function applyOptimistic(options: OptimisticOptions): Promise<() => void> {
  options.apply();
  const entry = await options.enqueue();

  const sub = liveQuery(() => db.outbox.get(entry.id)).subscribe((row) => {
    if (row === undefined) {
      sub.unsubscribe(); // discarded: delivered successfully, nothing to undo
      return;
    }
    if (row.status === 'needsAttention' || row.status === 'blocked') {
      options.rollback();
      options.onTerminalFailure?.(row);
      sub.unsubscribe();
    }
  });

  return () => sub.unsubscribe();
}
