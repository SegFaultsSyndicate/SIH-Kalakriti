// apps/artisan/src/lib/sync.ts
//
// One SyncEngine for the whole app: +layout.svelte starts it, the home
// screen's sync indicator and the manual "Sync now" action both read it.

import { SyncEngine } from '@kalakriti/offline';
import { sendOutboxEntry } from './outbox-send';

export const syncEngine = new SyncEngine(sendOutboxEntry);
