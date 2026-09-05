// packages/api/src/test-setup.ts
//
// auth.ts talks to IndexedDB at import time (Dexie opens the DB lazily but
// still needs the global to exist); Node has no IndexedDB, so vitest needs
// this polyfill wired in before any test file runs.
import 'fake-indexeddb/auto';
