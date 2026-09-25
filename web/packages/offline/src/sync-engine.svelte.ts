// packages/offline/src/sync-engine.svelte.ts
//
// Wires drainOutbox to its trigger sources and exposes sync status as a rune.
//
// Triggers, all real, all in-package:
//   - the network coming back online
//   - the tab regaining foreground (visibilitychange)
//   - a periodic timer, as a fallback for the case BackgroundSync can't cover
//   - a manual `syncNow()` call (e.g. a "sync now" button)
//
// BackgroundSync itself -- registering a service worker sync event so the
// browser can drain the outbox while the tab is closed -- is app-side: it
// needs a service worker file, which is vite-pwa config in a later batch.
// What belongs here is feature detection, so that batch knows whether it can
// rely on BackgroundSync or must lean on the foreground timer below.

import { liveQuery, type Subscription } from 'dexie';
import { db } from './db';
import { outboxCounts, type OutboxCounts } from './outbox';
import { drainOutbox, type SendFn } from './drain';
import { network } from './network.svelte';

export type SyncState = 'offline' | 'syncing' | 'attention' | 'pending' | 'synced';

const isBrowser = typeof window !== 'undefined';
const LAST_SYNC_PREF_KEY = 'offline.lastSyncAt';

export function backgroundSyncSupported(): boolean {
  return (
    typeof ServiceWorkerRegistration !== 'undefined' && 'sync' in ServiceWorkerRegistration.prototype
  );
}

export class SyncEngine {
  counts = $state<OutboxCounts>({
    total: 0,
    pending: 0,
    syncing: 0,
    failed: 0,
    needsAttention: 0,
    blocked: 0,
  });
  draining = $state(false);
  lastSyncAt = $state<number | null>(null);

  state: SyncState = $derived.by(() => {
    if (!network.online) return 'offline';
    if (this.draining) return 'syncing';
    if (this.counts.needsAttention > 0 || this.counts.blocked > 0) return 'attention';
    if (this.counts.total > 0) return 'pending';
    return 'synced';
  });

  #send: SendFn;
  #intervalMs: number;
  #timer: ReturnType<typeof setInterval> | null = null;
  #countsSub: Subscription | null = null;
  #disposers: Array<() => void> = [];

  constructor(send: SendFn, intervalMs = 45_000) {
    this.#send = send;
    this.#intervalMs = intervalMs;
  }

  /** Begin observing counts and wire the trigger sources. Returns a disposer. */
  start(): () => void {
    this.stop();

    void db.prefs.get(LAST_SYNC_PREF_KEY).then((row) => {
      if (row) this.lastSyncAt = row.value as number;
    });

    this.#countsSub = liveQuery(() => outboxCounts()).subscribe((c) => (this.counts = c));

    if (isBrowser) {
      const onOnline = () => void this.syncNow();
      window.addEventListener('online', onOnline);
      this.#disposers.push(() => window.removeEventListener('online', onOnline));

      const onVisible = () => {
        if (document.visibilityState === 'visible') void this.syncNow();
      };
      document.addEventListener('visibilitychange', onVisible);
      this.#disposers.push(() => document.removeEventListener('visibilitychange', onVisible));

      this.#timer = setInterval(() => void this.syncNow(), this.#intervalMs);
      this.#disposers.push(() => {
        if (this.#timer) clearInterval(this.#timer);
      });
    }

    return () => this.stop();
  }

  stop(): void {
    for (const dispose of this.#disposers.splice(0)) dispose();
    this.#timer = null;
    this.#countsSub?.unsubscribe();
    this.#countsSub = null;
  }

  /** Drain now. Safe to call from a button; concurrent calls collapse to one. */
  async syncNow(manual = false): Promise<void> {
    if (!network.online) return;
    this.draining = true;
    try {
      await drainOutbox(this.#send);
      if (manual) {
        const now = Date.now();
        this.lastSyncAt = now;
        await db.prefs.put({ key: LAST_SYNC_PREF_KEY, value: now, updatedAt: now });
      }
    } finally {
      this.draining = false;
    }
  }
}
