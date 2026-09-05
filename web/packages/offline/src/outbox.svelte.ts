// packages/offline/src/outbox.svelte.ts
//
// A live, rune-backed view of the outbox.
//
// Dexie's liveQuery already re-emits whenever the underlying tables change, so
// there is nothing to poll and nothing to invalidate by hand. This wraps that
// observable in $state so components can read it like any other value, and
// hands back a disposer so the subscription dies with the component.
//
// $effect appears here for the reason $effect exists: a subscription with a
// teardown. Nothing in this file computes a value in an effect.

import { liveQuery, type Subscription } from 'dexie';
import { db, type OutboxEntry } from './db';
import type { OutboxCounts } from './outbox';

export class OutboxView {
  entries = $state<OutboxEntry[]>([]);
  /** Distinguishes "no queued work" from "have not looked yet". */
  loaded = $state(false);
  error = $state<Error | null>(null);

  counts: OutboxCounts = $derived.by(() => {
    const counts: OutboxCounts = {
      total: this.entries.length,
      pending: 0,
      syncing: 0,
      failed: 0,
      needsAttention: 0,
      blocked: 0,
    };
    for (const entry of this.entries) counts[entry.status] += 1;
    return counts;
  });

  /** True when something is queued that the artisan may be waiting on. */
  hasWork: boolean = $derived(this.counts.total > 0);

  #subscription: Subscription | null = null;

  /**
   * Begin observing. Returns a disposer; call it from the $effect that started
   * this, or the subscription outlives the screen that opened it.
   */
  start(): () => void {
    this.#subscription?.unsubscribe();
    this.#subscription = liveQuery(() => db.outbox.orderBy('createdAt').toArray()).subscribe({
      next: (entries) => {
        this.entries = entries;
        this.loaded = true;
        this.error = null;
      },
      error: (cause: unknown) => {
        // IndexedDB can be unavailable outright: private browsing on some
        // Android builds, or storage evicted under pressure. The screen must
        // degrade to an error state, not a permanent spinner.
        this.error = cause instanceof Error ? cause : new Error(String(cause));
        this.loaded = true;
      },
    });
    return () => this.stop();
  }

  stop(): void {
    this.#subscription?.unsubscribe();
    this.#subscription = null;
  }
}
