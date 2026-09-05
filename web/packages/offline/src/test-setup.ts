// packages/offline/src/test-setup.ts
//
// db.ts opens IndexedDB at import time; Node has none, so vitest needs this
// polyfill wired in before any test file runs.
import 'fake-indexeddb/auto';
