// packages/offline/src/index.ts
export {
  db,
  KalakritiDatabase,
  type CacheRecord,
  type DraftRecord,
  type MediaRecord,
  type OutboxEntry,
  type OutboxKind,
  type OutboxStatus,
  type PrefRecord,
} from './db';
export {
  dependencySatisfied,
  discard,
  enqueue,
  entriesWithStatus,
  isMediaReferenced,
  markStatus,
  outboxCounts,
  queuedEntries,
  retryFailed,
  type EnqueueInput,
  type OutboxCounts,
} from './outbox';
export { OutboxView } from './outbox.svelte';
export { network, NetworkStatus } from './network.svelte';
export { backoffMs, drainOutbox, type DrainOptions, type SendFn, type SendResult } from './drain';
export { SyncEngine, backgroundSyncSupported, type SyncState } from './sync-engine.svelte';
export {
  evictIfNearQuota,
  evictOldestCachedMedia,
  requestPersistentStorage,
  storageEstimate,
} from './storage';
export {
  detectConflict,
  resolveConflict,
  type Conflict,
  type ConflictResolution,
} from './conflict';
export { default as ConflictBanner } from './ConflictBanner.svelte';
export { applyOptimistic, type OptimisticOptions } from './optimistic';
export { getCached, setCached } from './cache';
export { getPref, setPref } from './prefs';
export { saveDataEnabled, isSlowConnection, shouldConserveData } from './connection';
