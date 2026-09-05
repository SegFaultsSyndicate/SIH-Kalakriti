// packages/offline/src/drain.ts
//
// The sync engine's core: drain the outbox, oldest first, in dependency
// order. Pure with respect to the network -- the caller injects `send`, so
// this file never imports @kalakriti/api and stays testable with a fake.
//
// What "drain" means here, in order:
//   1. recover rows a killed tab left stuck in `syncing` (see below)
//   2. walk pending/failed/needsAttention rows oldest-first
//   3. skip a row whose dependsOn isn't satisfied yet, or whose backoff
//      hasn't elapsed, and continue to the next
//   4. send it, and update its status from the result

import { db, type OutboxEntry, type OutboxStatus } from './db';
import { dependencySatisfied, discard } from './outbox';

export interface SendResult {
  ok: boolean;
  /** Only meaningful when ok is false. Defaults to true: assume transient. */
  retryable?: boolean;
  error?: string;
}

export type SendFn = (entry: OutboxEntry) => Promise<SendResult>;

export interface DrainOptions {
  maxAttempts?: number;
  baseBackoffMs?: number;
  maxBackoffMs?: number;
  now?: () => number;
}

const DEFAULTS = { maxAttempts: 6, baseBackoffMs: 2_000, maxBackoffMs: 5 * 60_000 };

export function backoffMs(
  attempts: number,
  base = DEFAULTS.baseBackoffMs,
  max = DEFAULTS.maxBackoffMs,
): number {
  return Math.min(base * 2 ** attempts, max);
}

/**
 * Rows left in `syncing` were mid-flight when the tab died -- there is no
 * response to know if the request landed. Returning them to `pending` and
 * resending is safe specifically because `idempotencyKey` was minted at
 * enqueue time and is reused unchanged: a duplicate send of the same key
 * cannot create a second listing. This is the check that makes "tab-kill
 * mid-sync does not duplicate a listing" true.
 */
async function recoverOrphanedSyncing(now: number): Promise<void> {
  await db.outbox
    .where('status')
    .equals('syncing' satisfies OutboxStatus)
    .modify({ status: 'pending', updatedAt: now });
}

let draining = false;

/** Drain once. Safe to call from multiple triggers; concurrent calls no-op. */
export async function drainOutbox(send: SendFn, options: DrainOptions = {}): Promise<void> {
  if (draining) return;
  draining = true;
  try {
    const now = options.now ?? Date.now;
    const base = options.baseBackoffMs ?? DEFAULTS.baseBackoffMs;
    const max = options.maxBackoffMs ?? DEFAULTS.maxBackoffMs;
    const maxAttempts = options.maxAttempts ?? DEFAULTS.maxAttempts;

    await recoverOrphanedSyncing(now());

    const candidates = await db.outbox
      .where('status')
      .anyOf('pending', 'failed', 'needsAttention')
      .sortBy('createdAt');

    for (const entry of candidates) {
      const t = now();
      if (entry.status !== 'pending' && t - entry.updatedAt < backoffMs(entry.attempts, base, max)) {
        continue; // backoff not elapsed yet
      }
      if (!(await dependencySatisfied(entry))) continue;

      await db.outbox.update(entry.id, { status: 'syncing', updatedAt: t });
      const result = await send(entry);

      if (result.ok) {
        await discard(entry.id);
        continue;
      }

      const attempts = entry.attempts + 1;
      const retryable = result.retryable ?? true;
      const nextStatus: OutboxStatus = !retryable
        ? 'blocked'
        : attempts >= maxAttempts
          ? 'needsAttention'
          : 'failed';
      await db.outbox.update(entry.id, {
        status: nextStatus,
        attempts,
        lastError: result.error,
        updatedAt: now(),
      });
    }
  } finally {
    draining = false;
  }
}
